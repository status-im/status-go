package protocol

import (
	"context"
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
)

func (m *Messenger) SaveContact(contact *Contact) error {
	return m.saveContact(contact)
}

func (m *Messenger) AddContact(ctx context.Context, pubKey string) (*MessengerResponse, error) {
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		var err error
		contact, err = buildContactFromPkString(pubKey)
		if err != nil {
			return nil, err
		}
	}

	if !contact.IsAdded() {
		contact.Added = true
	}

	// We sync the contact with the other devices
	err := m.syncContact(context.Background(), contact)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return nil, err
	}

	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allContacts.Store(contact.ID, contact)

	// And we re-register for push notications
	err = m.reregisterForPushNotifications()
	if err != nil {
		return nil, err
	}

	// Reset last published time for ChatIdentity so new contact can receive data
	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	m.logger.Debug("contact state changed ResetWhenChatIdentityLastPublished")
	err = m.persistence.ResetWhenChatIdentityLastPublished(contactCodeTopic)
	if err != nil {
		m.logger.Error("ResetWhenChatIdentityLastPublished error", zap.Error(err))
		return nil, err
	}

	// Create the corresponding chat
	profileChat := m.buildProfileChat(contact.ID)

	_, err = m.Join(profileChat)
	if err != nil {
		return nil, err
	}

	// Finally we send a contact update so they are notified we added them
	// TODO: ens and picture are both blank for now
	response, err := m.sendContactUpdate(context.Background(), pubKey, "", "")
	if err != nil {
		return nil, err
	}

	response.AddChat(profileChat)

	publicKey, err := contact.PublicKey()
	if err != nil {
		return nil, err
	}

	// TODO: Add filters to response
	_, err = m.transport.InitFilters([]string{profileChat.ID}, []*ecdsa.PublicKey{publicKey})
	if err != nil {
		return nil, err
	}

	// Publish contact code
	err = m.publishContactCode()
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) removeContact(ctx context.Context, response *MessengerResponse, pubKey string) error {
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		return ErrContactNotFound
	}

	contact.Remove()

	err := m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return err
	}

	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allContacts.Store(contact.ID, contact)

	// And we re-register for push notications
	err = m.reregisterForPushNotifications()
	if err != nil {
		return err
	}

	// Create the corresponding profile chat
	profileChatID := buildProfileChatID(contact.ID)
	_, ok = m.allChats.Load(profileChatID)

	if ok {
		chatResponse, err := m.deactivateChat(profileChatID)
		if err != nil {
			return err
		}
		err = response.Merge(chatResponse)
		if err != nil {
			return err
		}
	}

	response.Contacts = []*Contact{contact}
	return nil
}
func (m *Messenger) RemoveContact(ctx context.Context, pubKey string) (*MessengerResponse, error) {
	response := new(MessengerResponse)

	err := m.removeContact(ctx, response, pubKey)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) Contacts() []*Contact {
	var contacts []*Contact
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		contacts = append(contacts, contact)
		return true
	})
	return contacts
}

// GetContactByID assumes pubKey includes 0x prefix
func (m *Messenger) GetContactByID(pubKey string) *Contact {
	contact, _ := m.allContacts.Load(pubKey)
	return contact
}

func (m *Messenger) BlockContact(contact *Contact) ([]*Chat, error) {
	contact.Block()
	chats, err := m.persistence.BlockContact(contact)
	if err != nil {
		return nil, err
	}

	m.allContacts.Store(contact.ID, contact)
	for _, chat := range chats {
		m.allChats.Store(chat.ID, chat)
	}
	m.allChats.Delete(contact.ID)

	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return nil, err
	}

	// re-register for push notifications
	err = m.reregisterForPushNotifications()
	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (m *Messenger) saveContact(contact *Contact) error {
	name, identicon, err := generateAliasAndIdenticon(contact.ID)
	if err != nil {
		return err
	}

	contact.Identicon = identicon
	contact.Alias = name

	if m.shouldSyncContact(contact) {
		if m.isNewContact(contact) {
			publicKey, err := contact.PublicKey()
			if err != nil {
				return err
			}
			filter, err := m.transport.JoinPrivate(publicKey)
			if err != nil {
				return err
			}
			m.scheduleSyncFilter(filter)
		}
		err := m.syncContact(context.Background(), contact)
		if err != nil {
			return err
		}
	}

	// We check if it should re-register with the push notification server
	shouldReregisterForPushNotifications := (m.isNewContact(contact) || m.removedContact(contact))

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	m.allContacts.Store(contact.ID, contact)

	// Reregister only when data has changed
	if shouldReregisterForPushNotifications {
		// Reset last published time for ChatIdentity so new contact can receive data
		contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
		m.logger.Debug("contact state changed ResetWhenChatIdentityLastPublished")
		err = m.persistence.ResetWhenChatIdentityLastPublished(contactCodeTopic)
		if err != nil {
			m.logger.Error("ResetWhenChatIdentityLastPublished error", zap.Error(err))
			return err
		}
		// Publish contact code
		err := m.publishContactCode()
		if err != nil {
			return err
		}

		return m.reregisterForPushNotifications()
	}

	return nil
}

// Send contact updates to all contacts added by us
func (m *Messenger) SendContactUpdates(ctx context.Context, ensName, profileImage string) (err error) {
	myID := contactIDFromPublicKey(&m.identity.PublicKey)

	if _, err = m.sendContactUpdate(ctx, myID, ensName, profileImage); err != nil {
		return err
	}

	// TODO: This should not be sending paired messages, as we do it above
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.IsAdded() {
			if _, err = m.sendContactUpdate(ctx, contact.ID, ensName, profileImage); err != nil {
				return false
			}
		}
		return true
	})
	return err
}

// NOTE: this endpoint does not add the contact, the reason being is that currently
// that's left as a responsibility to the client, which will call both `SendContactUpdate`
// and `SaveContact` with the correct system tag.
// Ideally we have a single endpoint that does both, but probably best to bring `ENS` name
// on the messenger first.

// SendContactUpdate sends a contact update to a user and adds the user to contacts
func (m *Messenger) SendContactUpdate(ctx context.Context, chatID, ensName, profileImage string) (*MessengerResponse, error) {
	return m.sendContactUpdate(ctx, chatID, ensName, profileImage)
}

func (m *Messenger) sendContactUpdate(ctx context.Context, chatID, ensName, profileImage string) (*MessengerResponse, error) {
	var response MessengerResponse

	contact, ok := m.allContacts.Load(chatID)
	if !ok {
		var err error
		contact, err = buildContactFromPkString(chatID)
		if err != nil {
			return nil, err
		}
	}

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return nil, err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chat.ID, chat)
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	contactUpdate := &protobuf.ContactUpdate{
		Clock:        clock,
		EnsName:      ensName,
		ProfileImage: profileImage}
	encodedMessage, err := proto.Marshal(contactUpdate)
	if err != nil {
		return nil, err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_CONTACT_UPDATE,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	response.Contacts = []*Contact{contact}
	response.AddChat(chat)

	chat.LastClockValue = clock
	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}
	return &response, m.saveContact(contact)
}

func (m *Messenger) isNewContact(contact *Contact) bool {
	previousContact, ok := m.allContacts.Load(contact.ID)
	return contact.IsAdded() && (!ok || !previousContact.IsAdded())
}

func (m *Messenger) shouldSyncContact(contact *Contact) bool {
	previousContact, ok := m.allContacts.Load(contact.ID)
	if !ok {
		return contact.IsAdded()
	}

	return contact.LocalNickname != previousContact.LocalNickname ||
		contact.IsAdded() != previousContact.IsAdded() ||
		previousContact.IsBlocked() != contact.IsBlocked()
}

func (m *Messenger) removedContact(contact *Contact) bool {
	previousContact, ok := m.allContacts.Load(contact.ID)
	if !ok {
		return false
	}
	return previousContact.IsAdded() && !contact.IsAdded()
}
