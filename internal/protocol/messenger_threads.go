package protocol

import (
	"database/sql"
	"errors"

	"github.com/status-im/status-go/internal/protocol/common"
)

func (m *Messenger) ThreadsByChatID(chatID string) ([]*Thread, error) {
	if !m.featureFlags.Threads {
		return nil, ErrThreadFeatureDisabled
	}
	return m.persistence.ThreadsByChatID(chatID)
}

func (m *Messenger) canCreateThread(chat *Chat) error {
	if !chat.SupportsThreads() {
		return ErrThreadsNotSupportedForChatType
	}

	if chat.ChatType != ChatTypeCommunityChat {
		return nil
	}

	community, err := m.communitiesManager.GetByIDString(chat.CommunityID)
	if err != nil {
		return err
	}

	if !community.AllowsAllMembersToCreateThread() && !community.IsPrivilegedMember(&m.identity.PublicKey) {
		return errors.New("only admins can create threads in this community")
	}

	return nil
}

// CreateThread creates thread metadata for an existing parent message in a chat.
// The parent message must already be stored locally. Permission rules (admin-only
// vs all-members) are enforced for community chats.
func (m *Messenger) CreateThread(chatID string, parentMessageID string) (*MessengerResponse, error) {
	if !m.featureFlags.Threads {
		return nil, ErrThreadFeatureDisabled
	}

	if chatID == "" || parentMessageID == "" {
		return nil, errors.New("chatID and parentMessageID are required")
	}

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, ErrChatNotFoundError
	}

	// Check if thread already exists for this parent message
	_, threadErr := m.persistence.ThreadByID(chatID, parentMessageID)
	if threadErr == nil {
		return nil, errors.New("thread already exists for this message")
	}
	if !errors.Is(threadErr, common.ErrRecordNotFound) {
		return nil, threadErr
	}

	if err := m.canCreateThread(chat); err != nil {
		return nil, err
	}

	parentMsg, msgErr := m.persistence.MessageByID(parentMessageID)
	if errors.Is(msgErr, sql.ErrNoRows) || errors.Is(msgErr, common.ErrRecordNotFound) {
		return nil, errors.New("parent message not found")
	}
	if msgErr != nil {
		return nil, msgErr
	}
	if parentMsg.LocalChatID != chatID {
		return nil, errors.New("parent message not found")
	}

	name := normalizeThreadName(parentMsg.Text)
	if err := m.persistence.UpsertThread(parentMessageID, chatID, parentMessageID, name); err != nil {
		return nil, err
	}

	thread, err := m.persistence.ThreadByID(chatID, parentMessageID)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddThread(thread)
	return response, nil
}

func (m *Messenger) addThreadsToResponse(response *MessengerResponse, messages []*common.Message) error {
	if !m.featureFlags.Threads || response == nil || len(messages) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	for _, message := range messages {
		if message == nil {
			continue
		}

		threadID := message.GetThreadId()
		if threadID == "" {
			continue
		}

		chatID := message.LocalChatID
		if chatID == "" {
			continue
		}

		key := threadKey(chatID, threadID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		thread, err := m.persistence.ThreadByID(chatID, threadID)
		if errors.Is(err, common.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}

		response.AddThread(thread)
	}

	return nil
}
