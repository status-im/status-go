package protocol

import (
	"context"
	"errors"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
)

func (m *Messenger) Chats() []*Chat {
	var chats []*Chat

	m.allChats.Range(func(chatID string, chat *Chat) (shouldContinue bool) {
		chats = append(chats, chat)
		return true
	})

	return chats
}

func (m *Messenger) CreateOneToOneChat(request *requests.CreateOneToOneChat) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	chatID := request.ID.String()
	pk, err := common.HexToPubkey(chatID)
	if err != nil {
		return nil, err
	}

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		chat = CreateOneToOneChat(chatID, pk, m.getTimesource())
	}
	chat.Active = true

	filters, err := m.Join(chat)
	if err != nil {
		return nil, err
	}

	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}

	// TODO(Samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chatID, chat)

	response := &MessengerResponse{
		Filters: filters,
	}
	response.AddChat(chat)

	return response, nil

}

func (m *Messenger) DeleteChat(chatID string) error {
	return m.deleteChat(chatID)
}

func (m *Messenger) deleteChat(chatID string) error {
	err := m.persistence.DeleteChat(chatID)
	if err != nil {
		return err
	}
	chat, ok := m.allChats.Load(chatID)

	if ok && chat.Active && chat.Public() {
		m.allChats.Delete(chatID)
		return m.reregisterForPushNotifications()
	}

	return nil
}

func (m *Messenger) SaveChat(chat *Chat) error {
	return m.saveChat(chat)
}

func (m *Messenger) DeactivateChat(chatID string) (*MessengerResponse, error) {
	return m.deactivateChat(chatID)
}

func (m *Messenger) deactivateChat(chatID string) (*MessengerResponse, error) {
	var response MessengerResponse
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, ErrChatNotFound
	}

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	err := m.persistence.DeactivateChat(chat, clock)

	if err != nil {
		return nil, err
	}

	// We re-register as our options have changed and we don't want to
	// receive PN from mentions in this chat anymore
	if chat.Public() {
		err := m.reregisterForPushNotifications()
		if err != nil {
			return nil, err
		}
	}

	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chatID, chat)

	response.AddChat(chat)
	// TODO: Remove filters

	return &response, nil
}

func (m *Messenger) saveChats(chats []*Chat) error {
	err := m.persistence.SaveChats(chats)
	if err != nil {
		return err
	}
	for _, chat := range chats {
		m.allChats.Store(chat.ID, chat)
	}

	return nil

}

func (m *Messenger) saveChat(chat *Chat) error {
	previousChat, ok := m.allChats.Load(chat.ID)
	if chat.OneToOne() {
		name, identicon, err := generateAliasAndIdenticon(chat.ID)
		if err != nil {
			return err
		}

		chat.Alias = name
		chat.Identicon = identicon
	}
	// Sync chat if it's a new active public chat, but not a timeline chat
	if !ok && chat.Active && chat.Public() && !chat.ProfileUpdates() && !chat.Timeline() {

		if err := m.syncPublicChat(context.Background(), chat); err != nil {
			return err
		}
	}

	// We check if it's a new chat, or chat.Active has changed
	// we check here, but we only re-register once the chat has been
	// saved an added
	shouldRegisterForPushNotifications := chat.Public() && (!ok && chat.Active) || (ok && chat.Active != previousChat.Active)

	err := m.persistence.SaveChat(*chat)
	if err != nil {
		return err
	}
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chat.ID, chat)

	if shouldRegisterForPushNotifications {
		// Re-register for push notifications, as we want to receive mentions
		if err := m.reregisterForPushNotifications(); err != nil {
			return err
		}

	}

	return nil
}

func (m *Messenger) Join(chat *Chat) ([]*transport.Filter, error) {
	switch chat.ChatType {
	case ChatTypeOneToOne:
		pk, err := chat.PublicKey()
		if err != nil {
			return nil, err
		}

		f, err := m.transport.JoinPrivate(pk)
		if err != nil {
			return nil, err
		}

		return []*transport.Filter{f}, nil
	case ChatTypePrivateGroupChat:
		members, err := chat.MembersAsPublicKeys()
		if err != nil {
			return nil, err
		}
		return m.transport.JoinGroup(members)
	case ChatTypePublic, ChatTypeProfile, ChatTypeTimeline:
		f, err := m.transport.JoinPublic(chat.ID)
		if err != nil {
			return nil, err
		}
		return []*transport.Filter{f}, nil
	default:
		return nil, errors.New("chat is neither public nor private")
	}
}
