package protocol

import (
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
)

type MessengerResponse struct {
	Chats          []*Chat                     `json:"chats,omitempty"`
	Messages       []*common.Message           `json:"messages,omitempty"`
	Contacts       []*Contact                  `json:"contacts,omitempty"`
	Installations  []*multidevice.Installation `json:"installations,omitempty"`
	EmojiReactions []*EmojiReaction            `json:"emojiReactions,omitempty"`
	Invitations    []*GroupChatInvitation      `json:"invitations,omitempty"`
}

func (m *MessengerResponse) IsEmpty() bool {
	return len(m.Chats) == 0 && len(m.Messages) == 0 && len(m.Contacts) == 0 && len(m.Installations) == 0 && len(m.Invitations) == 0
}

func (m *MessengerResponse) Merge(response *MessengerResponse) error {
	if len(response.Contacts)+len(response.Installations)+len(response.EmojiReactions)+len(response.Invitations) != 0 {
		return ErrNotImplemented
	}

	for _, overrideChat := range response.Chats {
		var found = false
		for idx, chat := range m.Chats {
			if chat.ID == overrideChat.ID {
				m.Chats[idx] = overrideChat
				found = true
			}
		}
		if !found {
			m.Chats = append(m.Chats, overrideChat)
		}
	}

	for _, overrideMessage := range response.Messages {
		var found = false
		for idx, chat := range m.Messages {
			if chat.ID == overrideMessage.ID {
				m.Messages[idx] = overrideMessage
				found = true
			}
		}
		if !found {
			m.Messages = append(m.Messages, overrideMessage)
		}
	}

	return nil
}
