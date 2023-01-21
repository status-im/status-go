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
	return m.addContact(contactRequest.From, "", "", "", contactRequest.ID, syncing, false)
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

	err = m.syncContactRequestDecision(ctx, request.ID.String(), true, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) declineContactRequest(requestID string, syncing bool) (*MessengerResponse, error) {
	m.logger.Info("declineContactRequest")
	contactRequest, err := m.persistence.MessageByID(requestID)
	if err != nil {
		return nil, err
	}

	contact, err := m.BuildContact(contactRequest.From)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}

	if !syncing {
		_, clock, err := m.getOneToOneAndNextClock(contact)
		if err != nil {
			return nil, err
		}

		contact.DismissContactRequest(clock)
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

	// update notification with the correct status
	notification, err := m.persistence.GetActivityCenterNotificationByID(types.FromHex(contactRequest.ID))
	if err != nil {
		return nil, err
	}
	if notification != nil {
		notification.Message = contactRequest
		notification.Read = true
		notification.Dismissed = true

		err = m.persistence.SaveActivityCenterNotification(notification)
		if err != nil {
			m.logger.Error("failed to save notification", zap.Error(err))
			return nil, err
		}

		response.AddActivityCenterNotification(notification)
	}
	response.AddMessage(contactRequest)
	return response, nil
}

func (m *Messenger) DeclineContactRequest(ctx context.Context, request *requests.DeclineContactRequest) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	response, err := m.declineContactRequest(request.ID.String(), false)
	if err != nil {
		return nil, err
	}

	err = m.syncContactRequestDecision(ctx, request.ID.String(), false, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) cancelOutgoingContactRequest(ctx context.Context, ID string) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	// remove contact
	err := m.removeContact(ctx, response, ID, true)
	if err != nil {
		return nil, err
	}

	// remove notification
	notificationID := types.FromHex(defaultContactRequestID(ID))
	notification, err := m.persistence.GetActivityCenterNotificationByID(notificationID)
	if err != nil {
		return nil, err
	}

	if notification != nil {
		err := m.persistence.DeleteActivityCenterNotification(notificationID)
		if err != nil {
			return nil, err
		}
	}

	// retract contact
	clock, _ := m.getLastClockWithRelatedChat()
	retractContactRequest := &protobuf.RetractContactRequest{
		Clock: clock,
	}
	encodedMessage, err := proto.Marshal(retractContactRequest)
	if err != nil {
		return nil, err
	}

	_, err = m.dispatchMessage(context.Background(), common.RawMessage{
		LocalChatID:         ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_RETRACT_CONTACT_REQUEST,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) CancelOutgoingContactRequest(ctx context.Context, request *requests.CancelOutgoingContactRequest) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	response, err := m.cancelOutgoingContactRequest(ctx, request.ID.String())
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

	response, err := m.addContact(chatID, "", "", "", "", false, true)
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

	_, clock, err := m.getOneToOneAndNextClock(contact)
	if err != nil {
		return nil, err
	}

	contact.AcceptContactRequest(clock)

	messageID := defaultContactRequestID(common.PubkeyToHex(&m.identity.PublicKey))
	acceptContactRequest := &protobuf.AcceptContactRequest{
		Id:    messageID,
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
		notification.Message = contactRequest
		notification.Read = true
		notification.Accepted = true

		err = m.persistence.SaveActivityCenterNotification(notification)
		if err != nil {
			m.logger.Error("failed to save notification", zap.Error(err))
			return nil, err
		}

		response.AddActivityCenterNotification(notification)
	}

	response.AddMessage(contactRequest)
	response.AddContact(contact)

	return response, nil
}

func (m *Messenger) addContact(pubKey, ensName, nickname, displayName, contactRequestID string, syncing bool, sendContactUpdate bool) (*MessengerResponse, error) {
	contact, err := m.BuildContact(pubKey)
	if err != nil {
		return nil, err
	}

	_, clock, err := m.getOneToOneAndNextClock(contact)
	if err != nil {
		return nil, err
	}

	if ensName != "" {
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

	contact.LastUpdatedLocally = clock
	contact.ContactRequestSent(clock)

	if !syncing {
		// We sync the contact with the other devices
		err := m.syncContact(context.Background(), contact, m.dispatchMessage)
		if err != nil {
			return nil, err
		}
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

	// Get ENS name of a current user
	ensName, err = m.settings.ENSName()
	if err != nil {
		return nil, err
	}

	// Get display name of a current user
	displayName, err = m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}

	if sendContactUpdate {
		response, err = m.sendContactUpdate(context.Background(), pubKey, displayName, ensName, "", m.dispatchMessage)
		if err != nil {
			return nil, err
		}
	} else if len(contactRequestID) != 0 {
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

	// Sends a standalone ChatIdentity message
	err = m.handleStandaloneChatIdentity(chat)
	if err != nil {
		return nil, err
	}

	// Add chat
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

	// Add outgoing contact request notification
	if len(contactRequestID) == 0 {
		err = m.createOutgoingContactRequestNotification(response, contact, profileChat)
		if err != nil {
			return nil, err
		}
	}

	// Add contact
	response.AddContact(contact)

	return response, nil
}

func (m *Messenger) generateContactRequest(clock uint64, timestamp uint64, contact *Contact) (*common.Message, error) {
	if contact == nil {
		return nil, errors.New("contact cannot be nil")
	}

	contactRequest := &common.Message{}
	contactRequest.WhisperTimestamp = timestamp
	contactRequest.Seen = false
	contactRequest.Text = "Please add me to your contacts"
	contactRequest.From = contact.ID
	contactRequest.LocalChatID = contact.ID
	contactRequest.ContentType = protobuf.ChatMessage_CONTACT_REQUEST
	contactRequest.Clock = clock
	contactRequest.ID = defaultContactRequestID(contact.ID)
	contactRequest.ContactRequestState = common.ContactRequestStatePending

	err := contactRequest.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	return contactRequest, err
}

func (m *Messenger) createOutgoingContactRequestNotification(response *MessengerResponse, contact *Contact, chat *Chat) error {
	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	contactRequest, err := m.generateContactRequest(clock, timestamp, contact)
	if err != nil {
		return err
	}

	response.AddMessage(contactRequest)
	err = m.persistence.SaveMessages([]*common.Message{contactRequest})
	if err != nil {
		return err
	}

	notification := &ActivityCenterNotification{
		ID:        types.FromHex(contactRequest.ID),
		Type:      ActivityCenterNotificationTypeContactRequest,
		Name:      contact.CanonicalName(),
		Author:    common.PubkeyToHex(&m.identity.PublicKey),
		Message:   contactRequest,
		Timestamp: m.getTimesource().GetCurrentTime(),
		ChatID:    contact.ID,
	}

	return m.addActivityCenterNotification(response, notification)
}

func (m *Messenger) AddContact(ctx context.Context, request *requests.AddContact) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	return m.addContact(request.ID.String(), request.ENSName, request.Nickname, request.DisplayName, "", false, true)
}

func (m *Messenger) resetLastPublishedTimeForChatIdentity() error {
	// Reset last published time for ChatIdentity so new contact can receive data
	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	m.logger.Debug("contact state changed ResetWhenChatIdentityLastPublished")
	return m.persistence.ResetWhenChatIdentityLastPublished(contactCodeTopic)
}

func (m *Messenger) removeContact(ctx context.Context, response *MessengerResponse, pubKey string, sync bool) error {
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		return ErrContactNotFound
	}

	_, clock, err := m.getOneToOneAndNextClock(contact)
	if err != nil {
		return err
	}

	contact.RetractContactRequest(clock)
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	if sync {
		err = m.syncContact(context.Background(), contact, m.dispatchMessage)
		if err != nil {
			return err
		}
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
		chatResponse, err := m.deactivateChat(profileChatID, 0, false, true)
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

	err := m.removeContact(ctx, response, pubKey, true)
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
		if contact.added() {
			contacts = append(contacts, contact)
		}
		return true
	})
	return contacts
}

func (m *Messenger) MutualContacts() []*Contact {
	var contacts []*Contact
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.mutual() {
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

	contact, err := m.BuildContact(pubKey)
	if err != nil {
		return nil, err
	}

	if err := m.addENSNameToContact(contact); err != nil {
		return nil, err
	}

	clock := m.getTimesource().GetCurrentTime()
	contact.LocalNickname = nickname
	contact.LastUpdatedLocally = clock

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return nil, err
	}

	m.allContacts.Store(contact.ID, contact)

	response := &MessengerResponse{}
	response.Contacts = []*Contact{contact}

	err = m.syncContact(context.Background(), contact, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) blockContact(contactID string, isDesktopFunc bool) ([]*Chat, error) {
	contact, err := m.BuildContact(contactID)
	if err != nil {
		return nil, err
	}

	_, clock, err := m.getOneToOneAndNextClock(contact)
	if err != nil {
		return nil, err
	}

	contact.Block(clock)

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

	err = m.syncContact(context.Background(), contact, m.dispatchMessage)
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

	_, clock, err := m.getOneToOneAndNextClock(contact)
	if err != nil {
		return err
	}

	contact.Unblock(clock)

	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	m.allContacts.Store(contact.ID, contact)

	err = m.syncContact(context.Background(), contact, m.dispatchMessage)
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

	if _, err = m.sendContactUpdate(ctx, myID, displayName, ensName, profileImage, m.dispatchMessage); err != nil {
		return err
	}

	// TODO: This should not be sending paired messages, as we do it above
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.added() {
			if _, err = m.sendContactUpdate(ctx, contact.ID, displayName, ensName, profileImage, m.dispatchMessage); err != nil {
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

	return m.sendContactUpdate(ctx, chatID, displayName, ensName, profileImage, m.dispatchMessage)
}

func (m *Messenger) sendContactUpdate(ctx context.Context, chatID, displayName, ensName, profileImage string, rawMessageHandler RawMessageHandler) (*MessengerResponse, error) {
	var response MessengerResponse

	contact, ok := m.allContacts.Load(chatID)
	if !ok || !contact.added() {
		return nil, nil
	}

	chat, clock, err := m.getOneToOneAndNextClock(contact)
	if err != nil {
		return nil, err
	}

	contactUpdate := &protobuf.ContactUpdate{
		Clock:               clock,
		DisplayName:         displayName,
		EnsName:             ensName,
		ProfileImage:        profileImage,
		ContactRequestClock: contact.ContactRequestLocalClock,
	}
	encodedMessage, err := proto.Marshal(contactUpdate)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_CONTACT_UPDATE,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
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
	contact, ok := m.allContacts.Load(request.ID.String())
	if !ok {
		return nil, errors.New("contact not found")
	}
	response := &MessengerResponse{}
	err = m.removeContact(context.Background(), response, contact.ID, true)
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
	_, clock, err := m.getOneToOneAndNextClock(contact)
	if err != nil {
		return err
	}
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

	return m.DeclineContactRequest(ctx, &requests.DeclineContactRequest{ID: types.Hex2Bytes(contactRequestID)})
}

func (m *Messenger) PendingContactRequests(cursor string, limit int) ([]*common.Message, string, error) {
	return m.persistence.PendingContactRequests(cursor, limit)
}

func defaultContactRequestID(contactID string) string {
	return "0x" + types.Bytes2Hex(append(types.Hex2Bytes(contactID), 0x20))
}

func (m *Messenger) BuildContact(contactID string) (*Contact, error) {
	contact, ok := m.allContacts.Load(contactID)
	if !ok {
		var err error
		contact, err = buildContactFromPkString(contactID)
		if err != nil {
			return nil, err
		}
	}
	return contact, nil
}
