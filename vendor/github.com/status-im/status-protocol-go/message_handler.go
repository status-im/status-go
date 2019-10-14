package statusproto

import (
	"github.com/pkg/errors"

	protocol "github.com/status-im/status-protocol-go/v1"
)

type persistentMessageHandler struct {
	persistence *sqlitePersistence
}

func newPersistentMessageHandler(persistence *sqlitePersistence) *persistentMessageHandler {
	return &persistentMessageHandler{persistence: persistence}
}

// HandleMembershipUpdate updates a Chat instance according to the membership updates.
// It retrieves chat, if exists, and merges membership updates from the message.
// Finally, the Chat is updated with the new group events.
func (h *persistentMessageHandler) HandleMembershipUpdate(m protocol.MembershipUpdateMessage) error {
	chat, err := h.chatID(m.ChatID)
	switch err {
	case errChatNotFound:
		group, err := protocol.NewGroupWithMembershipUpdates(m.ChatID, m.Updates)
		if err != nil {
			return err
		}
		newChat := createGroupChat()
		newChat.updateChatFromProtocolGroup(group)
		chat = &newChat
	case nil:
		existingGroup, err := newProtocolGroupFromChat(chat)
		if err != nil {
			return errors.Wrap(err, "failed to create a Group from Chat")
		}
		updateGroup, err := protocol.NewGroupWithMembershipUpdates(m.ChatID, m.Updates)
		if err != nil {
			return errors.Wrap(err, "invalid membership update")
		}
		merged := protocol.MergeFlatMembershipUpdates(existingGroup.Updates(), updateGroup.Updates())
		newGroup, err := protocol.NewGroup(chat.ID, merged)
		if err != nil {
			return errors.Wrap(err, "failed to create a group with new membership updates")
		}
		chat.updateChatFromProtocolGroup(newGroup)
	default:
		return err
	}
	return h.persistence.SaveChat(*chat)
}

func (h *persistentMessageHandler) chatID(chatID string) (*Chat, error) {
	var chat *Chat
	chats, err := h.persistence.Chats()
	if err != nil {
		return nil, err
	}
	for _, ch := range chats {
		if chat.ID == chatID {
			chat = ch
			break
		}
	}
	if chat == nil {
		return nil, errChatNotFound
	}
	return chat, nil
}
