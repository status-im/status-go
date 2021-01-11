package protocol

import (
	"context"
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) SaveContact(contact *Contact) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.saveContact(contact)
}

func (m *Messenger) AddContact(ctx context.Context, pubKey string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	contact, ok := m.allContacts[pubKey]
	if !ok {
		var err error
		contact, err = buildContactFromPkString(pubKey)
		if err != nil {
			return nil, err
		}
	}

	if !contact.IsAdded() {
		contact.SystemTags = append(contact.SystemTags, contactAdded)
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

	m.allContacts[contact.ID] = contact

	// And we re-register for push notications
	err = m.reregisterForPushNotifications()
	if err != nil {
		return nil, err
	}

	// Create the corresponding profile chat
	profileChatID := buildProfileChatID(contact.ID)
	profileChat, ok := m.allChats[profileChatID]

	if !ok {
		profileChat = CreateProfileChat(profileChatID, contact.ID, m.getTimesource())
	}

	filters, err := m.Join(profileChat)
	if err != nil {
		return nil, err
	}

	// Finally we send a contact update so they are notified we added them
	// TODO: ens and picture are both blank for now
	response, err := m.sendContactUpdate(context.Background(), pubKey, "", "")
	if err != nil {
		return nil, err
	}

	response.Filters = filters
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

	return response, nil
}

func (m *Messenger) RemoveContact(ctx context.Context, pubKey string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	var response *MessengerResponse

	contact, ok := m.allContacts[pubKey]
	if !ok {
		return nil, ErrContactNotFound
	}

	contact.Remove()

	err := m.persistence.SaveContact(contact, nil)
	if err != nil {
		return nil, err
	}

	m.allContacts[contact.ID] = contact

	// And we re-register for push notications
	err = m.reregisterForPushNotifications()
	if err != nil {
		return nil, err
	}

	// Create the corresponding profile chat
	profileChatID := buildProfileChatID(contact.ID)
	_, ok = m.allChats[profileChatID]

	if ok {
		chatResponse, err := m.deactivateChat(profileChatID)
		if err != nil {
			return nil, err
		}
		err = response.Merge(chatResponse)
		if err != nil {
			return nil, err
		}
	}

	response.Contacts = []*Contact{contact}
	return response, nil
}

func (m *Messenger) Contacts() []*Contact {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	var contacts []*Contact
	for _, contact := range m.allContacts {
		if contact.HasCustomFields() {
			contacts = append(contacts, contact)
		}
	}
	return contacts
}

// GetContactByID assumes pubKey includes 0x prefix
func (m *Messenger) GetContactByID(pubKey string) *Contact {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.allContacts[pubKey]
}

func (m *Messenger) BlockContact(contact *Contact) ([]*Chat, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	chats, err := m.persistence.BlockContact(contact)
	if err != nil {
		return nil, err
	}

	m.allContacts[contact.ID] = contact
	for _, chat := range chats {
		m.allChats[chat.ID] = chat
	}
	delete(m.allChats, contact.ID)

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

	if m.isNewContact(contact) || m.hasNicknameChanged(contact) {
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

	m.allContacts[contact.ID] = contact

	// Reregister only when data has changed
	if shouldReregisterForPushNotifications {
		return m.reregisterForPushNotifications()
	}

	return nil
}

// Send contact updates to all contacts added by us
func (m *Messenger) SendContactUpdates(ctx context.Context, ensName, profileImage string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	myID := contactIDFromPublicKey(&m.identity.PublicKey)

	if _, err := m.sendContactUpdate(ctx, myID, ensName, profileImage); err != nil {
		return err
	}

	// TODO: This should not be sending paired messages, as we do it above
	for _, contact := range m.allContacts {
		if contact.IsAdded() {
			if _, err := m.sendContactUpdate(ctx, contact.ID, ensName, profileImage); err != nil {
				return err
			}
		}
	}
	return nil
}

// NOTE: this endpoint does not add the contact, the reason being is that currently
// that's left as a responsibility to the client, which will call both `SendContactUpdate`
// and `SaveContact` with the correct system tag.
// Ideally we have a single endpoint that does both, but probably best to bring `ENS` name
// on the messenger first.

// SendContactUpdate sends a contact update to a user and adds the user to contacts
func (m *Messenger) SendContactUpdate(ctx context.Context, chatID, ensName, profileImage string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.sendContactUpdate(ctx, chatID, ensName, profileImage)
}

func (m *Messenger) sendContactUpdate(ctx context.Context, chatID, ensName, profileImage string) (*MessengerResponse, error) {
	var response MessengerResponse

	contact, ok := m.allContacts[chatID]
	if !ok {
		var err error
		contact, err = buildContactFromPkString(chatID)
		if err != nil {
			return nil, err
		}
	}

	chat, ok := m.allChats[chatID]
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return nil, err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats[chat.ID] = chat
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
	previousContact, ok := m.allContacts[contact.ID]
	return contact.IsAdded() && (!ok || !previousContact.IsAdded())
}

func (m *Messenger) hasNicknameChanged(contact *Contact) bool {
	previousContact, ok := m.allContacts[contact.ID]
	if !ok {
		return false
	}
	return contact.LocalNickname != previousContact.LocalNickname
}

func (m *Messenger) removedContact(contact *Contact) bool {
	previousContact, ok := m.allContacts[contact.ID]
	if !ok {
		return false
	}
	return previousContact.IsAdded() && !contact.IsAdded()
}
