package protocol

import (
	"github.com/pkg/errors"

	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

// HandleMembershipUpdate updates a Chat instance according to the membership updates.
// It retrieves chat, if exists, and merges membership updates from the message.
// Finally, the Chat is updated with the new group events.
func HandleMembershipUpdate(chat *Chat, m *v1protocol.MembershipUpdateMessage, myIdentity string, translations map[protobuf.MembershipUpdateEvent_EventType]string) (*Chat, []*Message, error) {
	if chat == nil {
		if len(m.Events) == 0 {
			return nil, nil, errors.New("can't create new group chat without events")
		}
		group, err := v1protocol.NewGroupWithEvents(m.ChatID, m.Events)
		if err != nil {
			return nil, nil, err
		}

		// A new chat must contain us
		if !group.IsMember(myIdentity) {
			return nil, nil, errors.New("can't create a new group chat without us being a member")
		}
		newChat := createGroupChat()
		newChat.updateChatFromProtocolGroup(group)
		return &newChat, buildSystemMessages(m.Events, translations), nil
	}
	existingGroup, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create a Group from Chat")
	}
	updateGroup, err := v1protocol.NewGroupWithEvents(m.ChatID, m.Events)
	if err != nil {
		return nil, nil, errors.Wrap(err, "invalid membership update")
	}
	merged := v1protocol.MergeMembershipUpdateEvents(existingGroup.Events(), updateGroup.Events())
	newGroup, err := v1protocol.NewGroup(chat.ID, merged)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create a group with new membership updates")
	}
	chat.updateChatFromProtocolGroup(newGroup)
	return chat, buildSystemMessages(m.Events, translations), nil
}
