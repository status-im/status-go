package protocol

import (
	"context"
	"errors"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"

	"strings"
)

func (m *Messenger) Chats() []*Chat {
	var chats []*Chat

	m.allChats.Range(func(chatID string, chat *Chat) (shouldContinue bool) {
		chats = append(chats, chat)
		return true
	})

	return chats
}

func (m *Messenger) ChatsPreview() []*ChatPreview {
	var chats []*ChatPreview

	m.allChats.Range(func(chatID string, chat *Chat) (shouldContinue bool) {
		if chat.Active || chat.Muted {
			chatPreview := &ChatPreview{
				ID:                    chat.ID,
				Name:                  chat.Name,
				Description:           chat.Description,
				Color:                 chat.Color,
				Emoji:                 chat.Emoji,
				Active:                chat.Active,
				ChatType:              chat.ChatType,
				Timestamp:             chat.Timestamp,
				LastClockValue:        chat.LastClockValue,
				DeletedAtClockValue:   chat.DeletedAtClockValue,
				UnviewedMessagesCount: chat.UnviewedMessagesCount,
				UnviewedMentionsCount: chat.UnviewedMentionsCount,
				Alias:                 chat.Alias,
				Identicon:             chat.Identicon,
				Muted:                 chat.Muted,
				Profile:               chat.Profile,
				CommunityID:           chat.CommunityID,
				CategoryID:            chat.CategoryID,
				Joined:                chat.Joined,
				SyncedTo:              chat.SyncedTo,
				SyncedFrom:            chat.SyncedFrom,
				Highlight:             chat.Highlight,
			}
			if chat.LastMessage != nil {
				chatPreview.ContentType = chat.LastMessage.ContentType
				if chat.LastMessage.ContentType == protobuf.ChatMessage_TEXT_PLAIN {

					simplifiedText, err := chat.LastMessage.GetSimplifiedText("", nil)

					if err == nil {
						if len(simplifiedText) > 100 {
							chatPreview.Text = simplifiedText[:100]
						} else {
							chatPreview.Text = simplifiedText
						}
						if strings.Contains(chatPreview.Text, "0x") {
							//if there is a mention, we would like to send parsed text as well
							chatPreview.ParsedText = chat.LastMessage.ParsedText
						}
					}
				} else if chat.LastMessage.ContentType == protobuf.ChatMessage_EMOJI ||
					chat.LastMessage.ContentType == protobuf.ChatMessage_TRANSACTION_COMMAND {

					chatPreview.Text = chat.LastMessage.Text
					chatPreview.ParsedText = chat.LastMessage.ParsedText
				}
				if chat.LastMessage.ContentType == protobuf.ChatMessage_COMMUNITY {
					chatPreview.ContentCommunityID = chat.LastMessage.CommunityID
				}
			}

			chats = append(chats, chatPreview)
		}

		return true
	})

	return chats
}

func (m *Messenger) Chat(chatID string) *Chat {
	chat, _ := m.allChats.Load(chatID)

	return chat
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

func (m *Messenger) initChatSyncFields(chat *Chat) error {
	defaultSyncPeriod, err := m.settings.GetDefaultSyncPeriod()
	if err != nil {
		return err
	}
	timestamp := uint32(m.getTimesource().GetCurrentTime()/1000) - defaultSyncPeriod
	chat.SyncedTo = timestamp
	chat.SyncedFrom = timestamp

	return nil
}

func (m *Messenger) createPublicChat(chatID string, response *MessengerResponse) (*MessengerResponse, error) {
	chat, ok := m.allChats.Load(chatID)
	wasActive := false
	if !ok {
		chat = CreatePublicChat(chatID, m.getTimesource())

	} else {
		wasActive = chat.Active
	}
	chat.Active = true
	chat.DeletedAtClockValue = 0

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
		if err := m.initChatSyncFields(chat); err != nil {
			return nil, err
		}
	}

	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}

	// Sync if it was created
	if !ok || !wasActive {
		if err := m.syncPublicChat(context.Background(), chat); err != nil {
			return nil, err
		}
	}

	err = m.reregisterForPushNotifications()
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)

	return response, nil
}

func (m *Messenger) CreatePublicChat(request *requests.CreatePublicChat) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	chatID := request.ID
	response := &MessengerResponse{}

	return m.createPublicChat(chatID, response)
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
		if err := m.initChatSyncFields(chat); err != nil {
			return nil, err
		}
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

	response := &MessengerResponse{}

	ensName := request.ENSName
	if ensName != "" {
		clock := m.getTimesource().GetCurrentTime()
		err := m.ensVerifier.ENSVerified(chatID, ensName, clock)
		if err != nil {
			return nil, err
		}
		contact, ok := m.allContacts.Load(chatID)
		if !ok {
			var err error
			contact, err = buildContactFromPkString(chatID)
			if err != nil {
				return nil, err
			}
		}
		contact.EnsName = ensName
		contact.ENSVerified = true
		err = m.persistence.SaveContact(contact, nil)
		if err != nil {
			return nil, err
		}
		response.Contacts = []*Contact{contact}
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

	response.AddChat(chat)

	willSync, err := m.scheduleSyncFilters(filters)
	if err != nil {
		return nil, err
	}

	// We set the synced to, synced from to the default time
	if !willSync {
		if err := m.initChatSyncFields(chat); err != nil {
			return nil, err
		}
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

	return m.deactivateChat(request.ID, 0, true)
}

func (m *Messenger) deactivateChat(chatID string, deactivationClock uint64, shouldBeSynced bool) (*MessengerResponse, error) {
	var response MessengerResponse
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, ErrChatNotFound
	}

	// Reset mailserver last request to allow re-fetching messages if joining a chat again
	filters, err := m.filtersForChat(chatID)
	if err != nil && err != ErrNoFiltersForChat {
		return nil, err
	}

	if m.mailserversDatabase != nil {
		for _, filter := range filters {
			if !filter.Listen || filter.Ephemeral {
				continue
			}

			err := m.mailserversDatabase.ResetLastRequest(filter.Topic.String())
			if err != nil {
				return nil, err
			}
		}
	}

	if deactivationClock == 0 {
		deactivationClock, _ = chat.NextClockAndTimestamp(m.getTimesource())
	}

	err = m.persistence.DeactivateChat(chat, deactivationClock)

	if err != nil {
		return nil, err
	}

	// We re-register as our options have changed and we don't want to
	// receive PN from mentions in this chat anymore
	if chat.Public() || chat.ProfileUpdates() {
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

	if shouldBeSynced {
		err := m.syncChatRemoving(context.Background(), chat.ID)
		if err != nil {
			return nil, err
		}
	}

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
	_, ok := m.allChats.Load(chat.ID)
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

	err := m.persistence.SaveChat(*chat)
	if err != nil {
		return err
	}
	// We store the chat has it might not have been in the store in the first place
	m.allChats.Store(chat.ID, chat)

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

	err = m.syncClearHistory(context.Background(), chat)
	if err != nil {
		return nil, err
	}

	m.allChats.Store(id, chat)

	response := &MessengerResponse{}
	response.AddChat(chat)
	return response, nil
}
