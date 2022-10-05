package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
)

func (m *Messenger) acceptContactRequest(requestID string, syncing bool) (*MessengerResponse, error) {
	contactRequest, err := m.persistence.MessageByID(requestID)
	if err != nil {
		m.logger.Error("could not find contact request message", zap.Error(err))
		return nil, err
	}

	m.logger.Info("acceptContactRequest")
	return m.addContact(contactRequest.From, "", "", "", contactRequest.ID, syncing)
}

func (m *Messenger) AcceptContactRequest(ctx context.Context, request *requests.AcceptContactRequest) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	response, err := m.acceptContactRequest(request.ID.String(), false)
	if err != nil {
		return nil, err
	}

	err = m.syncContactRequestDecision(ctx, request.ID.String(), true)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) SendContactRequest(ctx context.Context, request *requests.SendContactRequest) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	chatID := request.ID.String()

	response, err := m.addContact(chatID, "", "", "", "", false)
	if err != nil {
		return nil, err
	}

	publicKey, err := common.HexToPubkey(chatID)
	if err != nil {
		return nil, err
	}

	// A valid added chat is required.
	_, ok := m.allChats.Load(chatID)
	if !ok {
		// Create a one to one chat and set active to false
		chat := CreateOneToOneChat(chatID, publicKey, m.getTimesource())
		chat.Active = false
		err = m.initChatSyncFields(chat)
		if err != nil {
			return nil, err
		}
		err = m.saveChat(chat)
		if err != nil {
			return nil, err
		}
	}

	chatMessage := &common.Message{}
	chatMessage.ChatId = chatID
	chatMessage.Text = request.Message
	chatMessage.ContentType = protobuf.ChatMessage_CONTACT_REQUEST
	chatMessage.ContactRequestState = common.ContactRequestStatePending

	messageResponse, err := m.sendChatMessage(ctx, chatMessage)
	if err != nil {
		return nil, err
	}

	err = response.Merge(messageResponse)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// NOTE: This sets HasAddedUs to false, so next time we receive a contact request it will be reset to true
func (m *Messenger) RejectContactRequest(ctx context.Context, request *requests.RejectContactRequest) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	pubKey := request.ID.String()
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		var err error
		contact, err = buildContactFromPkString(pubKey)
		if err != nil {
			return nil, err
		}
	}

	contact.HasAddedUs = false

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return nil, err
	}

	m.allContacts.Store(contact.ID, contact)

	response := &MessengerResponse{}
	response.Contacts = []*Contact{contact}

	return response, nil
}

func (m *Messenger) dismissContactRequest(requestID string, syncing bool) (*MessengerResponse, error) {
	m.logger.Info("dismissContactRequest")
	contactRequest, err := m.persistence.MessageByID(requestID)
	if err != nil {
		return nil, err
	}

	contact, ok := m.allContacts.Load(contactRequest.From)
	if !ok {
		var err error
		contact, err = buildContactFromPkString(contactRequest.From)
		if err != nil {
			return nil, err
		}
	}

	response := &MessengerResponse{}

	if !syncing {
		contact.DismissContactRequest()
		err = m.persistence.SaveContact(contact, nil)
		if err != nil {
			return nil, err
		}

		response.AddContact(contact)
	}
	contactRequest.ContactRequestState = common.ContactRequestStateDismissed

	err = m.persistence.SetContactRequestState(contactRequest.ID, contactRequest.ContactRequestState)
	if err != nil {
		return nil, err
	}

	notification, err := m.persistence.GetActivityCenterNotificationByID(types.FromHex(contactRequest.ID))
	if err != nil {
		return nil, err
	}

	if notification != nil {
		err := m.persistence.UpdateActivityCenterNotificationMessage(notification.ID, contactRequest)
		if err != nil {
			return nil, err
		}
		notification.Message = contactRequest

		response.AddActivityCenterNotification(notification)
	}

	response.AddMessage(contactRequest)

	return response, nil
}

func (m *Messenger) DismissContactRequest(ctx context.Context, request *requests.DismissContactRequest) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	response, err := m.dismissContactRequest(request.ID.String(), false)
	if err != nil {
		return nil, err
	}

	err = m.syncContactRequestDecision(ctx, request.ID.String(), false)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) updateAcceptedContactRequest(response *MessengerResponse, contactRequestID string) (*MessengerResponse, error) {
	contactRequest, err := m.persistence.MessageByID(contactRequestID)
	if err != nil {
		return nil, err
	}

	contactRequest.ContactRequestState = common.ContactRequestStateAccepted

	err = m.persistence.SetContactRequestState(contactRequest.ID, contactRequest.ContactRequestState)
	if err != nil {
		return nil, err
	}

	contact, _ := m.allContacts.Load(contactRequest.From)
	contact.AcceptContactRequest()

	chat, ok := m.allChats.Load(contact.ID)
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return nil, err
		}

		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		chat.Active = false
		if err := m.saveChat(chat); err != nil {
			return nil, err
		}
	}
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	acceptContactRequest := &protobuf.AcceptContactRequest{
		Id:    contactRequest.ID,
		Clock: clock,
	}
	encodedMessage, err := proto.Marshal(acceptContactRequest)
	if err != nil {
		return nil, err
	}

	_, err = m.dispatchMessage(context.Background(), common.RawMessage{
		LocalChatID:         contactRequest.From,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_ACCEPT_CONTACT_REQUEST,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	notification, err := m.persistence.GetActivityCenterNotificationByID(types.FromHex(contactRequest.ID))
	if err != nil {
		return nil, err
	}

	if response == nil {
		response = &MessengerResponse{}
	}

	if notification != nil {
		err := m.persistence.UpdateActivityCenterNotificationMessage(notification.ID, contactRequest)
		if err != nil {
			return nil, err
		}
		notification.Message = contactRequest

		response.AddActivityCenterNotification(notification)
	}

	response.AddMessage(contactRequest)

	return response, nil
}

func (m *Messenger) addContact(pubKey, ensName, nickname, displayName, contactRequestID string, syncing bool) (*MessengerResponse, error) {
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		var err error
		contact, err = buildContactFromPkString(pubKey)
		if err != nil {
			return nil, err
		}
	}

	if ensName != "" {
		clock := m.getTimesource().GetCurrentTime()
		err := m.ensVerifier.ENSVerified(pubKey, ensName, clock)
		if err != nil {
			return nil, err
		}
	}

	if err := m.addENSNameToContact(contact); err != nil {
		return nil, err
	}

	if len(nickname) != 0 {
		contact.LocalNickname = nickname
	}

	if len(displayName) != 0 {
		contact.DisplayName = displayName
	}

	if !contact.Added {
		contact.Add()
	}
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()

	contact.ContactRequestSent()

	if !syncing {
		// We sync the contact with the other devices
		err := m.syncContact(context.Background(), contact)
		if err != nil {
			return nil, err
		}
	}

	err := m.persistence.SaveContact(contact, nil)
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
	err = m.resetLastPublishedTimeForChatIdentity()
	if err != nil {
		return nil, err
	}

	// Create the corresponding chat
	profileChat := m.buildProfileChat(contact.ID)

	_, err = m.Join(profileChat)
	if err != nil {
		return nil, err
	}
	if err := m.saveChat(profileChat); err != nil {
		return nil, err
	}

	// Fetch contact code
	publicKey, err := contact.PublicKey()
	if err != nil {
		return nil, err
	}
	filter, err := m.transport.JoinPrivate(publicKey)
	if err != nil {
		return nil, err
	}
	_, err = m.scheduleSyncFilters([]*transport.Filter{filter})
	if err != nil {
		return nil, err
	}

	ensName, err = m.settings.ENSName()
	if err != nil {
		return nil, err
	}

	displayName, err = m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	// Finally we send a contact update so they are notified we added them
	response, err := m.sendContactUpdate(context.Background(), pubKey, displayName, ensName, "")
	if err != nil {
		return nil, err
	}

	if len(contactRequestID) != 0 {
		response, err = m.updateAcceptedContactRequest(response, contactRequestID)
		if err != nil {
			return nil, err
		}
	}

	// Send profile picture with contact request
	chat, ok := m.allChats.Load(contact.ID)
	if !ok {
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		chat.Active = false
		if err := m.saveChat(chat); err != nil {
			return nil, err
		}
	}

	err = m.handleStandaloneChatIdentity(chat)
	if err != nil {
		return nil, err
	}

	response.AddChat(profileChat)

	_, err = m.transport.InitFilters([]string{profileChat.ID}, []*ecdsa.PublicKey{publicKey})
	if err != nil {
		return nil, err
	}

	// Publish contact code
	err = m.publishContactCode()
	if err != nil {
		return nil, err
	}

	response.AddContact(contact)

	return response, nil
}
func (m *Messenger) AddContact(ctx context.Context, request *requests.AddContact) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	return m.addContact(request.ID.String(), request.ENSName, request.Nickname, request.DisplayName, "", false)
}

func (m *Messenger) resetLastPublishedTimeForChatIdentity() error {
	// Reset last published time for ChatIdentity so new contact can receive data
	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	m.logger.Debug("contact state changed ResetWhenChatIdentityLastPublished")
	return m.persistence.ResetWhenChatIdentityLastPublished(contactCodeTopic)
}

func (m *Messenger) removeContact(ctx context.Context, response *MessengerResponse, pubKey string) error {
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		return ErrContactNotFound
	}

	contact.RetractContactRequest()
	contact.Remove()
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()

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
		chatResponse, err := m.deactivateChat(profileChatID, 0, false)
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

func (m *Messenger) AddedContacts() []*Contact {
	var contacts []*Contact
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.Added {
			contacts = append(contacts, contact)
		}
		return true
	})
	return contacts
}

func (m *Messenger) MutualContacts() []*Contact {
	var contacts []*Contact
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.Added && contact.HasAddedUs {
			contacts = append(contacts, contact)
		}
		return true
	})
	return contacts
}

func (m *Messenger) BlockedContacts() []*Contact {
	var contacts []*Contact
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.Blocked {
			contacts = append(contacts, contact)
		}
		return true
	})
	return contacts
}

// GetContactByID assumes pubKey includes 0x prefix
func (m *Messenger) GetContactByID(pubKey string) *Contact {
	contact, _ := m.allContacts.Load(pubKey)
	return contact
}

func (m *Messenger) SetContactLocalNickname(request *requests.SetContactLocalNickname) (*MessengerResponse, error) {

	if err := request.Validate(); err != nil {
		return nil, err
	}

	pubKey := request.ID.String()
	nickname := request.Nickname

	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		var err error
		contact, err = buildContactFromPkString(pubKey)
		if err != nil {
			return nil, err
		}
	}

	if err := m.addENSNameToContact(contact); err != nil {
		return nil, err
	}

	clock := m.getTimesource().GetCurrentTime()
	contact.LocalNickname = nickname
	contact.LastUpdatedLocally = clock

	err := m.persistence.SaveContact(contact, nil)
	if err != nil {
		return nil, err
	}

	m.allContacts.Store(contact.ID, contact)

	response := &MessengerResponse{}
	response.Contacts = []*Contact{contact}

	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) blockContact(contactID string, isDesktopFunc bool) ([]*Chat, error) {
	contact, ok := m.allContacts.Load(contactID)
	if !ok {
		var err error
		contact, err = buildContactFromPkString(contactID)
		if err != nil {
			return nil, err
		}

	}
	if isDesktopFunc {
		contact.BlockDesktop()
	} else {
		contact.Block()
	}

	err := m.sendRetractContactRequest(contact)
	if err != nil {
		return nil, err
	}

	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()

	chats, err := m.persistence.BlockContact(contact, isDesktopFunc)
	if err != nil {
		return nil, err
	}

	m.allContacts.Store(contact.ID, contact)
	for _, chat := range chats {
		m.allChats.Store(chat.ID, chat)
	}

	if !isDesktopFunc {
		m.allChats.Delete(contact.ID)
		m.allChats.Delete(buildProfileChatID(contact.ID))
	}

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

func (m *Messenger) BlockContact(contactID string) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	chats, err := m.blockContact(contactID, false)
	if err != nil {
		return nil, err
	}
	response.AddChats(chats)

	response, err = m.DeclineAllPendingGroupInvitesFromUser(response, contactID)
	if err != nil {
		return nil, err
	}

	err = m.persistence.DismissAllActivityCenterNotificationsFromUser(contactID)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// The same function as the one above.
func (m *Messenger) BlockContactDesktop(contactID string) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	chats, err := m.blockContact(contactID, true)
	if err != nil {
		return nil, err
	}
	response.AddChats(chats)

	response, err = m.DeclineAllPendingGroupInvitesFromUser(response, contactID)
	if err != nil {
		return nil, err
	}

	err = m.persistence.DismissAllActivityCenterNotificationsFromUser(contactID)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) UnblockContact(contactID string) error {
	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Blocked {
		return nil
	}

	contact.Unblock()
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()

	err := m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	m.allContacts.Store(contact.ID, contact)

	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return err
	}

	// re-register for push notifications
	err = m.reregisterForPushNotifications()
	if err != nil {
		return err
	}

	return nil
}

// Send contact updates to all contacts added by us
func (m *Messenger) SendContactUpdates(ctx context.Context, ensName, profileImage string) (err error) {
	myID := contactIDFromPublicKey(&m.identity.PublicKey)

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	if _, err = m.sendContactUpdate(ctx, myID, displayName, ensName, profileImage); err != nil {
		return err
	}

	// TODO: This should not be sending paired messages, as we do it above
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.Added {
			if _, err = m.sendContactUpdate(ctx, contact.ID, displayName, ensName, profileImage); err != nil {
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
	displayName, err := m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	return m.sendContactUpdate(ctx, chatID, displayName, ensName, profileImage)
}

func (m *Messenger) sendContactUpdate(ctx context.Context, chatID, displayName, ensName, profileImage string) (*MessengerResponse, error) {
	var response MessengerResponse

	contact, ok := m.allContacts.Load(chatID)
	if !ok || !contact.Added {
		return nil, nil
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
		DisplayName:  displayName,
		EnsName:      ensName,
		ProfileImage: profileImage,
	}
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
	return &response, nil
}

func (m *Messenger) addENSNameToContact(contact *Contact) error {

	// Check if there's already a verified record
	ensRecord, err := m.ensVerifier.GetVerifiedRecord(contact.ID)
	if err != nil {
		return err
	}
	if ensRecord == nil {
		return nil
	}

	contact.EnsName = ensRecord.Name
	contact.ENSVerified = true

	return nil
}

func (m *Messenger) RetractContactRequest(request *requests.RetractContactRequest) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	contact, ok := m.allContacts.Load(request.ContactID.String())
	if !ok {
		return nil, errors.New("contact not found")
	}
	contact.HasAddedUs = false
	m.allContacts.Store(contact.ID, contact)
	response := &MessengerResponse{}
	err = m.removeContact(context.Background(), response, contact.ID)
	if err != nil {
		return nil, err
	}

	err = m.sendRetractContactRequest(contact)
	if err != nil {
		return nil, err
	}

	return response, err
}

// Send message to remote account to remove our contact from their end.
func (m *Messenger) sendRetractContactRequest(contact *Contact) error {
	chat, ok := m.allChats.Load(contact.ID)
	if !ok {
		pubKey, err := contact.PublicKey()
		if err != nil {
			return err
		}

		chat = OneToOneFromPublicKey(pubKey, m.getTimesource())
		chat.Active = false
		if err := m.saveChat(chat); err != nil {
			return err
		}
	}
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	retractContactRequest := &protobuf.RetractContactRequest{
		Clock: clock,
	}

	encodedMessage, err := proto.Marshal(retractContactRequest)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(context.Background(), common.RawMessage{
		LocalChatID:         contact.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_RETRACT_CONTACT_REQUEST,
		ResendAutomatically: true,
	})

	return err
}

func (m *Messenger) AcceptLatestContactRequestForContact(ctx context.Context, request *requests.AcceptLatestContactRequestForContact) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	contactRequestID, err := m.persistence.LatestPendingContactRequestIDForContact(request.ID.String())
	if err != nil {
		return nil, err
	}
	if contactRequestID == "" {
		contactRequestID = defaultContactRequestID(request.ID.String())
	}

	return m.AcceptContactRequest(ctx, &requests.AcceptContactRequest{ID: types.Hex2Bytes(contactRequestID)})
}

func (m *Messenger) DismissLatestContactRequestForContact(ctx context.Context, request *requests.DismissLatestContactRequestForContact) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	contactRequestID, err := m.persistence.LatestPendingContactRequestIDForContact(request.ID.String())
	if err != nil {
		return nil, err
	}

	if contactRequestID == "" {
		contactRequestID = defaultContactRequestID(request.ID.String())
	}

	return m.DismissContactRequest(ctx, &requests.DismissContactRequest{ID: types.Hex2Bytes(contactRequestID)})
}

func (m *Messenger) PendingContactRequests(cursor string, limit int) ([]*common.Message, string, error) {
	return m.persistence.PendingContactRequests(cursor, limit)
}

func defaultContactRequestID(contactID string) string {
	return "0x" + types.Bytes2Hex(append(types.Hex2Bytes(contactID), 0x20))
}
