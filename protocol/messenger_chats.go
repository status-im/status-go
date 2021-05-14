package protocol

import (
	"context"
	"errors"

	"github.com/status-im/status-go/eth-node/types"
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

func (m *Messenger) ActiveChats() []*Chat {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var chats []*Chat

	m.allChats.Range(func(chatID string, c *Chat) bool {
		if c.Active {
			chats = append(chats, c)
		}
		return true
	})

	return chats
}

func (m *Messenger) CreatePublicChat(request *requests.CreatePublicChat) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	chatID := request.ID

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		chat = CreatePublicChat(chatID, m.getTimesource())

	}
	chat.Active = true

	// Save topics
	_, err := m.Join(chat)
	if err != nil {
		return nil, err
	}

	// Store chat
	m.allChats.Store(chat.ID, chat)

	willSync, err := m.scheduleSyncChat(chat)
	if err != nil {
		return nil, err
	}

	// We set the synced to, synced from to the default time
	if !willSync {
		timestamp := uint32(m.getTimesource().GetCurrentTime()/1000) - defaultSyncInterval
		chat.SyncedTo = timestamp
		chat.SyncedFrom = timestamp
	}

	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}

	// Sync if it was created
	if !ok {
		if err := m.syncPublicChat(context.Background(), chat); err != nil {
			return nil, err
		}
	}

	response := &MessengerResponse{}
	response.AddChat(chat)

	return response, nil
}

func (m *Messenger) CreateProfileChat(request *requests.CreateProfileChat) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	publicKey, err := common.HexToPubkey(request.ID)
	if err != nil {
		return nil, err
	}

	chat := m.buildProfileChat(request.ID)

	chat.Active = true

	// Save topics
	_, err = m.Join(chat)
	if err != nil {
		return nil, err
	}

	// Check contact code
	filter, err := m.transport.JoinPrivate(publicKey)
	if err != nil {
		return nil, err
	}

	// Store chat
	m.allChats.Store(chat.ID, chat)

	response := &MessengerResponse{}
	response.AddChat(chat)

	willSync, err := m.scheduleSyncChat(chat)
	if err != nil {
		return nil, err
	}

	// We set the synced to, synced from to the default time
	if !willSync {
		timestamp := uint32(m.getTimesource().GetCurrentTime()/1000) - defaultSyncInterval
		chat.SyncedTo = timestamp
		chat.SyncedFrom = timestamp
	}

	_, err = m.scheduleSyncFilters([]*transport.Filter{filter})
	if err != nil {
		return nil, err
	}

	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}

	return response, nil
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

	// TODO(Samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chatID, chat)

	response := &MessengerResponse{}
	response.AddChat(chat)

	willSync, err := m.scheduleSyncFilters(filters)
	if err != nil {
		return nil, err
	}

	// We set the synced to, synced from to the default time
	if !willSync {
		timestamp := uint32(m.getTimesource().GetCurrentTime()/1000) - defaultSyncInterval
		chat.SyncedTo = timestamp
		chat.SyncedFrom = timestamp
	}

	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}
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

func (m *Messenger) DeactivateChat(request *requests.DeactivateChat) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	return m.deactivateChat(request.ID)
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

		err = m.transport.ClearProcessedMessageIDsCache()
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
		// We clear all notifications so it pops up again
		if !chat.Active {
			err := m.persistence.DeleteActivityCenterNotification(types.FromHex(chat.ID))
			if err != nil {
				return err
			}
		}
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

func (m *Messenger) buildProfileChat(id string) *Chat {
	// Create the corresponding profile chat
	profileChatID := buildProfileChatID(id)
	profileChat, ok := m.allChats.Load(profileChatID)

	if !ok {
		profileChat = CreateProfileChat(id, m.getTimesource())
	}

	return profileChat

}

func (m *Messenger) ensureTimelineChat() error {
	chat, err := m.persistence.Chat(timelineChatID)
	if err != nil {
		return err
	}

	if chat != nil {
		return nil
	}

	chat = CreateTimelineChat(m.getTimesource())
	m.allChats.Store(timelineChatID, chat)
	return m.saveChat(chat)
}

func (m *Messenger) ensureMyOwnProfileChat() error {
	chatID := common.PubkeyToHex(&m.identity.PublicKey)
	_, ok := m.allChats.Load(chatID)
	if ok {
		return nil
	}

	chat := m.buildProfileChat(chatID)

	chat.Active = true

	// Save topics
	_, err := m.Join(chat)
	if err != nil {
		return err
	}

	return m.saveChat(chat)
}

func (m *Messenger) ClearHistory(request *requests.ClearHistory) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	return m.clearHistory(request.ID)
}

func (m *Messenger) clearHistory(id string) (*MessengerResponse, error) {
	chat, ok := m.allChats.Load(id)
	if !ok {
		return nil, ErrChatNotFound
	}

	clock, _ := chat.NextClockAndTimestamp(m.transport)

	err := m.persistence.ClearHistory(chat, clock)
	if err != nil {
		return nil, err
	}

	if chat.Public() {

		err = m.transport.ClearProcessedMessageIDsCache()
		if err != nil {
			return nil, err
		}
	}

	m.allChats.Store(id, chat)

	response := &MessengerResponse{}
	response.AddChat(chat)
	return response, nil
}
