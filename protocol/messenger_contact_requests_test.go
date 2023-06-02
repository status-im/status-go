package protocol

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerContactRequestSuite(t *testing.T) {
	suite.Run(t, new(MessengerContactRequestSuite))
}

type MessengerContactRequestSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerContactRequestSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerContactRequestSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerContactRequestSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerContactRequestSuite) findFirstByContentType(messages []*common.Message, contentType protobuf.ChatMessage_ContentType) *common.Message {
	for _, message := range messages {
		if message.ContentType == contentType {
			return message
		}
	}
	return nil
}

func (s *MessengerContactRequestSuite) sendContactRequest(request *requests.SendContactRequest, messenger *Messenger) {
	// Send contact request
	resp, err := messenger.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Check CR and mutual state update messages
	s.Require().Len(resp.Messages(), 2)

	contactRequest := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequest)

	mutualStateUpdate := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_STATE_UPDATE)
	s.Require().NotNil(mutualStateUpdate)

	s.Require().Equal(common.ContactRequestStatePending, contactRequest.ContactRequestState)
	s.Require().Equal(request.Message, contactRequest.Text)
	s.Require().NotNil(mutualStateUpdate.ID)
	s.Require().Equal(mutualStateUpdate.From, messenger.myHexIdentity())
	s.Require().Equal(mutualStateUpdate.ChatId, request.ID)
	s.Require().Equal(mutualStateUpdate.Text, "You sent a contact request to @"+request.ID)

	// Check pending notification
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().Equal(contactRequest.ID, resp.ActivityCenterNotifications()[0].Message.ID)
	s.Require().Equal(contactRequest.ContactRequestState, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, true)

	// Check contacts
	s.Require().Len(resp.Contacts, 1)
	contact := resp.Contacts[0]
	s.Require().False(contact.mutual())

	// Make sure it's not returned as coming from us
	contactRequests, _, err := messenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 0)

	// Make sure contact is added on the sender side
	contacts := messenger.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)
	s.Require().NotNil(contacts[0].DisplayName)

	// Check contact's primary name matches notifiaction's name
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Name, contacts[0].PrimaryName())

	// Make sure update message was saved properly
	mutualStateUpdateFromDb, err := messenger.persistence.MessageByID(mutualStateUpdate.ID)
	s.Require().NoError(err)
	s.Require().Equal(mutualStateUpdateFromDb.From, messenger.myHexIdentity())
	s.Require().Equal(mutualStateUpdateFromDb.ChatId, request.ID)
	s.Require().Equal(mutualStateUpdateFromDb.Text, "You sent a contact request to @"+request.ID)
}

func (s *MessengerContactRequestSuite) receiveContactRequest(messageText string, theirMessenger *Messenger) *common.Message {
	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) == 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Check CR and mutual state update messages
	s.Require().Len(resp.Messages(), 2)

	contactRequest := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequest)

	mutualStateUpdate := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_STATE_UPDATE)
	s.Require().NotNil(mutualStateUpdate)

	s.Require().Equal(common.ContactRequestStatePending, contactRequest.ContactRequestState)
	s.Require().Equal(messageText, contactRequest.Text)
	s.Require().Equal(mutualStateUpdate.From, contactRequest.From)
	s.Require().Equal(mutualStateUpdate.ChatId, contactRequest.From)
	s.Require().Equal(mutualStateUpdate.Text, "@"+contactRequest.From+" sent you a contact request")

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().Equal(contactRequest.ID, resp.ActivityCenterNotifications()[0].Message.ID)
	s.Require().Equal(contactRequest.ContactRequestState, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, false)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Accepted, false)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Dismissed, false)

	notifications, err := theirMessenger.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Cursor:        "",
		Limit:         10,
		ActivityTypes: []ActivityCenterType{ActivityCenterNotificationTypeContactRequest},
		ReadType:      ActivityCenterQueryParamsReadUnread,
	},
	)
	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 1)
	s.Require().Equal(contactRequest.ID, notifications.Notifications[0].Message.ID)
	s.Require().Equal(contactRequest.ContactRequestState, notifications.Notifications[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	contact := resp.Contacts[0]
	s.Require().Equal(ContactRequestStateReceived, contact.ContactRequestRemoteState)

	// Check contact's primary name matches notifiaction's name
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Name, contact.PrimaryName())

	// Make sure it's the latest pending contact requests
	contactRequests, _, err := theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Greater(len(contactRequests), 0)
	s.Require().Equal(contactRequests[0].ID, contactRequest.ID)

	return contactRequest
}

func (s *MessengerContactRequestSuite) acceptContactRequest(contactRequest *common.Message, sender *Messenger, receiver *Messenger) {
	// Accept contact request, receiver side
	resp, err := receiver.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(contactRequest.ID)})
	s.Require().NoError(err)

	// Chack updated contact request message and mutual state update
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 2)

	contactRequestMsg := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequestMsg)

	mutualStateUpdate := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_STATE_UPDATE)
	s.Require().NotNil(mutualStateUpdate)

	s.Require().Equal(contactRequestMsg.ID, contactRequest.ID)
	s.Require().Equal(common.ContactRequestStateAccepted, contactRequestMsg.ContactRequestState)
	// Reversed for add contact update
	s.Require().Equal(mutualStateUpdate.From, contactRequestMsg.ChatId)
	s.Require().Equal(mutualStateUpdate.ChatId, contactRequestMsg.From)
	s.Require().Equal(mutualStateUpdate.Text, "You added  @"+contactRequestMsg.From+" as a contact")

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequest.ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Check contact's primary name matches notifiaction's name
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Name, resp.Contacts[0].PrimaryName())

	// Check we have active chat in the response
	s.Require().Len(resp.Chats(), 2)
	s.Require().True(resp.Chats()[0].Active) // This is unactive profile chat
	s.Require().True(resp.Chats()[1].Active)

	// Make sure the sender is added to our contacts
	contacts := receiver.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := receiver.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		sender,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) == 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, true)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Accepted, true)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Dismissed, false)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)

	// Make sure the message is updated, sender s2de
	s.Require().Len(resp.Messages(), 2)

	contactRequestMsg = s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequestMsg)

	mutualStateUpdate = s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_STATE_UPDATE)
	s.Require().NotNil(mutualStateUpdate)

	s.Require().Equal(contactRequest.ID, contactRequestMsg.ID)
	s.Require().Equal(contactRequest.Text, contactRequestMsg.Text)
	s.Require().Equal(common.ContactRequestStateAccepted, contactRequestMsg.ContactRequestState)

	s.Require().Equal(mutualStateUpdate.From, contactRequestMsg.ChatId)
	s.Require().Equal(mutualStateUpdate.ChatId, contactRequestMsg.ChatId)
	s.Require().Equal(mutualStateUpdate.Text, "@"+mutualStateUpdate.From+" added you as a contact")

	// Make sure we consider them a mutual contact, sender side
	mutualContacts = s.m.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	contact := resp.Contacts[0]
	s.Require().True(contact.mutual())

	// Check contact's primary name matches notifiaction's name
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Name, contact.PrimaryName())

	// Sender's side chat should be active after the accepting the CR
	chat, ok := s.m.allChats.Load(contact.ID)
	s.Require().True(ok)
	s.Require().NotNil(chat)
	s.Require().True(chat.Active)

	// Receiver's side chat should be also active after the accepting the CR
	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	chat, ok = receiver.allChats.Load(myID)
	s.Require().True(ok)
	s.Require().NotNil(chat)
	s.Require().True(chat.Active)
}

func (s *MessengerContactRequestSuite) declineContactRequest(contactRequest *common.Message, theirMessenger *Messenger) {
	// Dismiss contact request, receiver side
	resp, err := theirMessenger.DeclineContactRequest(context.Background(), &requests.DeclineContactRequest{ID: types.Hex2Bytes(contactRequest.ID)})
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateDismissed, resp.Contacts[0].ContactRequestLocalState)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequest.ID)
	s.Require().Equal(common.ContactRequestStateDismissed, resp.Messages()[0].ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequest.ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, true)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Accepted, false)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Dismissed, true)
	s.Require().Equal(common.ContactRequestStateDismissed, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check contact's primary name matches notifiaction's name
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Name, resp.Contacts[0].PrimaryName())

	// Make sure the sender is not added to our contacts
	contacts := theirMessenger.AddedContacts()
	s.Require().Len(contacts, 0)
}

func (s *MessengerContactRequestSuite) retractContactRequest(contactID string, theirMessenger *Messenger) {
	resp, err := s.m.RetractContactRequest(&requests.RetractContactRequest{ID: types.Hex2Bytes(contactID)})
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Len(resp.Contacts, 1)
	s.Require().False(resp.Contacts[0].hasAddedUs())
	s.Require().False(resp.Contacts[0].added())

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestRemoteState)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Len(resp.Contacts, 1)

	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	s.Require().Equal(myID, resp.Contacts[0].ID)

	s.Require().False(resp.Contacts[0].added())
	s.Require().False(resp.Contacts[0].hasAddedUs())
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestRemoteState)

	// Check pending notification
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRemoved, resp.ActivityCenterNotifications()[0].Type)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, false)
}

func (s *MessengerContactRequestSuite) syncInstallationContactV2FromContact(contact *Contact) protobuf.SyncInstallationContactV2 {
	return protobuf.SyncInstallationContactV2{
		LastUpdatedLocally:        contact.LastUpdatedLocally,
		LastUpdated:               contact.LastUpdated,
		Id:                        contact.ID,
		DisplayName:               contact.DisplayName,
		EnsName:                   contact.EnsName,
		LocalNickname:             contact.LocalNickname,
		Added:                     contact.added(),
		Blocked:                   contact.Blocked,
		Muted:                     false,
		HasAddedUs:                contact.hasAddedUs(),
		Removed:                   contact.Removed,
		ContactRequestLocalState:  int64(contact.ContactRequestLocalState),
		ContactRequestRemoteState: int64(contact.ContactRequestRemoteState),
		ContactRequestRemoteClock: int64(contact.ContactRequestRemoteClock),
		ContactRequestLocalClock:  int64(contact.ContactRequestLocalClock),
		VerificationStatus:        int64(contact.VerificationStatus),
		TrustStatus:               int64(contact.TrustStatus),
	}
}

func (s *MessengerContactRequestSuite) TestReceiveAndAcceptContactRequest() { //nolint: unused
	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageText,
	}
	s.sendContactRequest(request, s.m)
	contactRequest := s.receiveContactRequest(messageText, theirMessenger)
	s.acceptContactRequest(contactRequest, s.m, theirMessenger)
}

func (s *MessengerContactRequestSuite) TestReceiveAndDismissContactRequest() {
	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageText,
	}
	s.sendContactRequest(request, s.m)
	contactRequest := s.receiveContactRequest(messageText, theirMessenger)
	s.declineContactRequest(contactRequest, theirMessenger)
}

func (s *MessengerContactRequestSuite) TestReceiveAcceptAndRetractContactRequest() { //nolint: unused
	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	s.Require().NoError(theirMessenger.settings.SaveSettingField(settings.MutualContactEnabled, true))

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageText,
	}
	s.sendContactRequest(request, s.m)
	contactRequest := s.receiveContactRequest(messageText, theirMessenger)
	s.acceptContactRequest(contactRequest, s.m, theirMessenger)
	s.retractContactRequest(contactID, theirMessenger)
}

func (s *MessengerContactRequestSuite) TestReceiveAndAcceptContactRequestTwice() { //nolint: unused
	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageText,
	}
	s.sendContactRequest(request, s.m)
	contactRequest := s.receiveContactRequest(messageText, theirMessenger)
	s.acceptContactRequest(contactRequest, s.m, theirMessenger)

	// Resend contact request with higher clock value
	resp, err := s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Check CR and mutual state update messages
	s.Require().Len(resp.Messages(), 2)

	contactRequest = s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequest)

	mutualStateUpdate := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_STATE_UPDATE)
	s.Require().NotNil(mutualStateUpdate)

	s.Require().Equal(common.ContactRequestStateAccepted, contactRequest.ContactRequestState)
	s.Require().Equal(request.Message, contactRequest.Text)
	s.Require().Equal(mutualStateUpdate.From, s.m.myHexIdentity())
	s.Require().Equal(mutualStateUpdate.ChatId, request.ID)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(resp.Messages(), 1)

	contactRequest = s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequest)

	s.Require().Equal(common.ContactRequestStateAccepted, contactRequest.ContactRequestState)
	s.Require().Equal(request.Message, contactRequest.Text)

	// Nothing should have changed, on both sides
	mutualContacts := s.m.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	mutualContacts = theirMessenger.MutualContacts()
	s.Require().Len(mutualContacts, 1)
}

func (s *MessengerContactRequestSuite) TestAcceptLatestContactRequestForContact() {
	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageText,
	}
	s.sendContactRequest(request, s.m)
	contactRequest := s.receiveContactRequest(messageText, theirMessenger)

	// Accept latest contact request, receiver side
	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	resp, err := theirMessenger.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 2)

	contactRequestMsg := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequestMsg)

	mutualStateUpdate := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_STATE_UPDATE)
	s.Require().NotNil(mutualStateUpdate)

	s.Require().Equal(contactRequestMsg.ID, contactRequest.ID)
	s.Require().Equal(common.ContactRequestStateAccepted, contactRequestMsg.ContactRequestState)

	s.Require().Equal(mutualStateUpdate.From, contactRequest.ChatId)
	s.Require().Equal(mutualStateUpdate.ChatId, contactRequest.From)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequest.ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the sender is added to our contacts
	contacts := theirMessenger.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := theirMessenger.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) == 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure the message is updated, sender side
	s.Require().NotNil(resp)

	s.Require().Len(resp.Messages(), 2)

	contactRequestMsg = s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequestMsg)

	mutualStateUpdate = s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_STATE_UPDATE)
	s.Require().NotNil(mutualStateUpdate)

	s.Require().Equal(common.ContactRequestStateAccepted, contactRequestMsg.ContactRequestState)

	s.Require().Equal(mutualStateUpdate.From, contactRequest.ChatId)
	s.Require().Equal(mutualStateUpdate.ChatId, contactRequest.ChatId)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts = s.m.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())
}

func (s *MessengerContactRequestSuite) TestDismissLatestContactRequestForContact() {
	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageText,
	}
	s.sendContactRequest(request, s.m)
	contactRequest := s.receiveContactRequest(messageText, theirMessenger)

	// Dismiss latest contact request, receiver side
	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	resp, err := theirMessenger.DismissLatestContactRequestForContact(context.Background(), &requests.DismissLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequest.ID)
	s.Require().Equal(common.ContactRequestStateDismissed, resp.Messages()[0].ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequest.ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateDismissed, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
}

func (s *MessengerContactRequestSuite) TestPairedDevicesRemoveContact() {
	messageText := "hello!"

	alice1 := s.m
	alice2, err := newMessengerWithKey(s.shh, s.m.identity, s.logger, nil)
	s.Require().NoError(err)

	_, err = alice2.Start()
	s.Require().NoError(err)

	prepAliceMessengersForPairing(&s.Suite, alice1, alice2)

	pairTwoDevices(&s.Suite, alice1, alice2)
	pairTwoDevices(&s.Suite, alice2, alice1)

	bob := s.newMessenger(s.shh)
	_, err = bob.Start()
	s.Require().NoError(err)

	// Alice sends a contact request to bob
	contactID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageText,
	}
	s.sendContactRequest(request, alice1)
	contactRequest := s.receiveContactRequest(messageText, bob)
	s.acceptContactRequest(contactRequest, alice1, bob)

	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		alice2,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts := alice2.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	s.retractContactRequest(contactID, bob)

	// Check on alice2 side
	resp, err = WaitOnMessengerResponse(
		alice2,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Len(resp.Contacts, 1)

	// Check the contact state is correctly set
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestRemoteState)
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob accepts the contact request
// 3) Alice restores state on a different device
// 4) Alice sends a contact request to bob
// Bob will need to help Alice recover her state, since as far as he can see
// that's an already accepted contact request
func (s *MessengerContactRequestSuite) TestAliceRecoverStateSendContactRequest() {
	messageText := "hello!"

	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))

	// Alice sends a contact request to bob
	request := &requests.SendContactRequest{
		ID:      bobID,
		Message: messageText,
	}
	s.sendContactRequest(request, alice1)

	contactRequest := s.receiveContactRequest(messageText, bob)
	s.Require().NotNil(contactRequest)

	// Bob accepts the contact request
	s.acceptContactRequest(contactRequest, alice1, bob)

	// Alice resets her device
	alice2, err := newMessengerWithKey(s.shh, s.m.identity, s.logger, nil)
	s.Require().NoError(err)

	_, err = alice2.Start()
	s.Require().NoError(err)

	// adds bob again to her device
	s.sendContactRequest(request, alice2)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Bob should be a mutual contact with alice, nothing has changed
	s.Require().Len(bob.MutualContacts(), 1)

	// Alice retrieves her messages, she should have been notified by
	// dear bobby that they were contacts
	resp, err := WaitOnMessengerResponse(
		alice2,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Len(resp.Contacts, 1)

	// Check the contact state is correctly set
	s.Require().True(resp.Contacts[0].mutual())
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob accepts the contact request
// 3) Alice restores state on a different device
// 4) Bob sends a message to alice
// Alice will show a contact request from bob
func (s *MessengerContactRequestSuite) TestAliceRecoverStateReceiveContactRequest() {
	messageText := "hello!"

	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))

	// Alice sends a contact request to bob
	request := &requests.SendContactRequest{
		ID:      bobID,
		Message: messageText,
	}
	s.sendContactRequest(request, alice1)

	contactRequest := s.receiveContactRequest(messageText, bob)
	s.Require().NotNil(contactRequest)

	// Bob accepts the contact request
	s.acceptContactRequest(contactRequest, alice1, bob)

	// Alice resets her device
	alice2, err := newMessengerWithKey(s.shh, s.m.identity, s.logger, nil)
	s.Require().NoError(err)

	_, err = alice2.Start()
	s.Require().NoError(err)

	// We want to facilitate the discovery of the x3dh bundl here, since bob does not know about alice device

	alice2Bundle, err := alice2.encryptor.GetBundle(alice2.identity)
	s.Require().NoError(err)

	_, err = bob.encryptor.ProcessPublicBundle(bob.identity, alice2Bundle)
	s.Require().NoError(err)

	// Bob sends a chat message to alice

	var chat Chat
	chats := bob.Chats()
	for i, c := range chats {
		if c.ID == alice1.myHexIdentity() && c.OneToOne() {
			chat = *chats[i]
		}
	}
	s.Require().NotNil(chat)

	inputMessage := buildTestMessage(chat)
	_, err = bob.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	// Alice retrieves the chat message, it should be
	resp, err := WaitOnMessengerResponse(
		alice2,
		func(r *MessengerResponse) bool {
			return len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().Len(resp.Contacts, 1)

	// Check the contact state is correctly set
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob accepts the contact request
// 3) Bob goes offline
// 4) Alice retracts the contact request
// 5) Alice adds bob back to her contacts
// 6) Bob goes online, they receive 4 and 5 in the correct order
func (s *MessengerContactRequestSuite) TestAliceOfflineRetractsAndAddsCorrectOrder() {
	messageText := "hello!"

	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))

	// Alice sends a contact request to bob
	request := &requests.SendContactRequest{
		ID:      bobID,
		Message: messageText,
	}
	s.sendContactRequest(request, alice1)

	contactRequest := s.receiveContactRequest(messageText, bob)
	s.Require().NotNil(contactRequest)

	// Bob accepts the contact request
	s.acceptContactRequest(contactRequest, alice1, bob)

	// Alice removes Bob from contacts
	_, err = alice1.RetractContactRequest(&requests.RetractContactRequest{ID: types.Hex2Bytes(bob.myHexIdentity())})
	s.Require().NoError(err)

	// Adds bob again to her device
	s.sendContactRequest(request, alice1)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob accepts the contact request
// 3) Bob goes offline
// 4) Alice retracts the contact request
// 5) Alice adds bob back to her contacts
// 6) Bob goes online, they receive 4 and 5 in the wrong order
func (s *MessengerContactRequestSuite) TestAliceOfflineRetractsAndAddsWrongOrder() {
	messageText := "hello!"

	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))

	// Alice sends a contact request to bob
	request := &requests.SendContactRequest{
		ID:      bobID,
		Message: messageText,
	}
	s.sendContactRequest(request, alice1)

	contactRequest := s.receiveContactRequest(messageText, bob)
	s.Require().NotNil(contactRequest)

	// Bob accepts the contact request
	s.acceptContactRequest(contactRequest, alice1, bob)

	// Alice removes Bob from contacts
	_, err = alice1.RetractContactRequest(&requests.RetractContactRequest{ID: types.Hex2Bytes(bob.myHexIdentity())})
	s.Require().NoError(err)

	// Adds bob again to her device
	s.sendContactRequest(request, alice1)

	// Get alice perspective of bob
	bobFromAlice := alice1.AddedContacts()[0]

	// Get bob perspective of alice
	aliceFromBob := bob.MutualContacts()[0]

	s.Require().NotNil(bobFromAlice)
	s.Require().NotNil(aliceFromBob)

	// We can't simulate out-of-order messages easily, so we need to do
	// things manually here
	result := aliceFromBob.ContactRequestPropagatedStateReceived(bobFromAlice.ContactRequestPropagatedState())
	s.Require().True(result.newContactRequestReceived)
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob accepts the contact request
// 3) Alice removes Bob from contacts
// 4) Make sure Alice and Bob are not mutual contacts
// 5) Alice sends new contact request
// 6) Bob accepts new contact request
func (s *MessengerContactRequestSuite) TestAliceResendsContactRequestAfterRemovingBobFromContacts() {
	messageTextFirst := "hello 1!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	// Alice sends a contact request to Bob
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageTextFirst,
	}
	s.sendContactRequest(request, s.m)

	// Bob accepts the contact request
	contactRequest := s.receiveContactRequest(messageTextFirst, theirMessenger)
	s.Require().NotNil(contactRequest)
	s.acceptContactRequest(contactRequest, s.m, theirMessenger)

	// Alice removes Bob from contacts
	s.retractContactRequest(contactID, theirMessenger)

	// Send new contact request
	messageTextSecond := "hello 2!"

	// Alice sends new contact request
	request = &requests.SendContactRequest{
		ID:      contactID,
		Message: messageTextSecond,
	}
	s.sendContactRequest(request, s.m)

	// Make sure bob and alice are not mutual after sending CR
	s.Require().Len(s.m.MutualContacts(), 0)
	s.Require().Len(theirMessenger.MutualContacts(), 0)

	// Bob accepts new contact request
	contactRequest = s.receiveContactRequest(messageTextSecond, theirMessenger)
	s.Require().NotNil(contactRequest)
	s.acceptContactRequest(contactRequest, s.m, theirMessenger)

	// Make sure bob and alice are not mutual after sending CR
	s.Require().Len(s.m.MutualContacts(), 1)
	s.Require().Len(theirMessenger.MutualContacts(), 1)
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob declines the contact request from Alice
// 3) Bob sends a contact request to Alice
// 4) Alice and Bob are mutual contacts (because Alice's CR is "pending" on her side), Both CRs are accepted
func (s *MessengerContactRequestSuite) TestBobSendsContactRequestAfterDecliningOneFromAlice() {
	messageTextAlice := "hello, Bobby!"

	alice := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))

	// Alice sends a contact request to bob
	requestFromAlice := &requests.SendContactRequest{
		ID:      bobID,
		Message: messageTextAlice,
	}
	s.sendContactRequest(requestFromAlice, alice)

	contactRequest := s.receiveContactRequest(messageTextAlice, bob)
	s.Require().NotNil(contactRequest)

	// Bob declines the contact request
	s.declineContactRequest(contactRequest, bob)

	messageTextBob := "hello, Alice!"

	aliceID := types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey))

	// Bob sends a contact request to Alice
	requestFromBob := &requests.SendContactRequest{
		ID:      aliceID,
		Message: messageTextBob,
	}

	// Send contact request
	resp, err := bob.SendContactRequest(context.Background(), requestFromBob)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Check CR message, it should be accepted
	s.Require().Len(resp.Messages(), 2)

	contactRequest = s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequest)

	mutualStateUpdate := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_STATE_UPDATE)
	s.Require().NotNil(mutualStateUpdate)

	s.Require().Equal(common.ContactRequestStateAccepted, contactRequest.ContactRequestState)
	s.Require().Equal(requestFromBob.Message, contactRequest.Text)

	s.Require().Equal(mutualStateUpdate.From, contactRequest.From)
	s.Require().Equal(mutualStateUpdate.ChatId, contactRequest.ChatId)

	// Check pending notification
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().Equal(contactRequest.ID, resp.ActivityCenterNotifications()[0].Message.ID)
	s.Require().Equal(contactRequest.ContactRequestState, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, true)

	// Check contacts Bob's side
	s.Require().Len(resp.Contacts, 1)
	contact := resp.Contacts[0]
	s.Require().True(contact.mutual())

	// Make sure contact is added on the sender side
	contacts := bob.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)
	s.Require().NotNil(contacts[0].DisplayName)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) == 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	contactRequestMsg := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequestMsg)

	// Check CR message, it should be accepted
	s.Require().Len(resp.Messages(), 2)

	contactRequest = s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequest)

	s.Require().Equal(common.ContactRequestStateAccepted, contactRequest.ContactRequestState)
	s.Require().Equal(requestFromBob.Message, contactRequest.Text)

	// Check pending notification
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().Equal(contactRequest.ID, resp.ActivityCenterNotifications()[0].Message.ID)
	s.Require().Equal(contactRequest.ContactRequestState, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, false)

	// Check contacts Alice's side
	s.Require().Len(resp.Contacts, 1)
	contact = resp.Contacts[0]
	s.Require().True(contact.mutual())

	// Make sure contact is added on the receiver's side
	contacts = alice.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)
	s.Require().NotNil(contacts[0].DisplayName)
	s.Require().True(contacts[0].mutual())
}

func (s *MessengerContactRequestSuite) TestBuildContact() {
	contactID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	contact, err := s.m.BuildContact(&requests.BuildContact{PublicKey: contactID})
	s.Require().NoError(err)

	s.Require().Equal(contact.EnsName, "")
	s.Require().False(contact.ENSVerified)

	contact, err = s.m.BuildContact(&requests.BuildContact{PublicKey: contactID, ENSName: "foobar"})
	s.Require().NoError(err)

	s.Require().Equal(contact.EnsName, "foobar")
	s.Require().True(contact.ENSVerified)
}

func (s *MessengerContactRequestSuite) TestReceiveAcceptAndRetractContactRequestOutOfOrder() {
	message := protobuf.ChatMessage{
		Clock:       4,
		Timestamp:   1,
		Text:        "some text",
		ChatId:      common.PubkeyToHex(&s.m.identity.PublicKey),
		MessageType: protobuf.MessageType_ONE_TO_ONE,
		ContentType: protobuf.ChatMessage_CONTACT_REQUEST,
	}

	contactKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	contact, err := BuildContactFromPublicKey(&contactKey.PublicKey)
	s.Require().NoError(err)

	state := s.m.buildMessageState()

	state.CurrentMessageState = &CurrentMessageState{
		PublicKey:        &contactKey.PublicKey,
		MessageID:        "0xa",
		Message:          message,
		Contact:          contact,
		WhisperTimestamp: 1,
	}

	response := state.Response
	err = s.m.HandleChatMessage(state)
	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	contacts := s.m.Contacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, contacts[0].ContactRequestRemoteState)

	retract := protobuf.RetractContactRequest{
		Clock: 2,
	}
	err = s.m.HandleRetractContactRequest(state, retract)
	s.Require().NoError(err)

	// Nothing should have changed
	contacts = s.m.Contacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, contacts[0].ContactRequestRemoteState)
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob receives CR from Alice
// 3) Bob resets his device
// 4) Bob restores Alice's contact from backup, CR is created
// 5) Bob succesefully accepts restored contact request
// 6) Alice get notified properly
func (s *MessengerContactRequestSuite) TestBobRestoresIncomingContactRequestFromSyncInstallationContactV2() {
	messageText := "hello, Bobby!"

	alice := s.m

	bob1 := s.newMessenger(s.shh)
	_, err := bob1.Start()
	s.Require().NoError(err)

	aliceID := types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey))
	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob1.identity.PublicKey))

	// Alice sends a contact request to bob
	requestFromAlice := &requests.SendContactRequest{
		ID:      bobID,
		Message: messageText,
	}
	s.sendContactRequest(requestFromAlice, alice)

	// Bob receives CR from Alice
	contactRequest := s.receiveContactRequest(messageText, bob1)
	s.Require().NotNil(contactRequest)

	// Bob resets his device
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)

	_, err = bob2.Start()
	s.Require().NoError(err)

	// Get bob perspective of alice for backup
	aliceFromBob := bob1.Contacts()[0]
	state := bob2.buildMessageState()

	// Restore alice's contact from backup
	sync := s.syncInstallationContactV2FromContact(aliceFromBob)
	err = bob2.HandleSyncInstallationContact(state, sync)
	s.Require().NoError(err)

	// Accept latest CR for a contact
	resp, err := bob2.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(aliceID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 2)

	contactRequestMsg := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequestMsg)

	// NOTE: We don't restore CR message
	// s.Require().Equal(resp.Messages()[0].ID, contactRequest.ID)
	s.Require().Equal(common.ContactRequestStateAccepted, contactRequestMsg.ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the sender is added to our contacts
	contacts := bob2.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := bob2.MutualContacts()
	s.Require().Len(mutualContacts, 1)
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob receives CR from Alice
// 3) Alice resets her device
// 4) Alice restores Bob's contact from backup, CR is created
// 5) Bob accepts contact request
// 6) Alice get notified properly
func (s *MessengerContactRequestSuite) TestAliceRestoresOutgoingContactRequestFromSyncInstallationContactV2() {
	messageText := "hello, Bobby!"

	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	aliceID := types.EncodeHex(crypto.FromECDSAPub(&alice1.identity.PublicKey))
	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))

	// Alice sends a contact request to bob
	requestFromAlice := &requests.SendContactRequest{
		ID:      bobID,
		Message: messageText,
	}
	s.sendContactRequest(requestFromAlice, alice1)

	// Bob receives CR from Alice
	contactRequest := s.receiveContactRequest(messageText, bob)
	s.Require().NotNil(contactRequest)

	// Bob resets his device
	alice2, err := newMessengerWithKey(s.shh, alice1.identity, s.logger, nil)
	s.Require().NoError(err)

	_, err = alice2.Start()
	s.Require().NoError(err)

	// Get bob perspective of alice for backup
	bobFromAlice := alice1.Contacts()[0]
	state := alice2.buildMessageState()

	// Restore alice's contact from backup
	sync := s.syncInstallationContactV2FromContact(bobFromAlice)
	err = alice2.HandleSyncInstallationContact(state, sync)
	s.Require().NoError(err)

	// Accept latest CR for a contact
	resp, err := bob.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(aliceID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 2)

	contactRequestMsg := s.findFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequestMsg)

	// NOTE: We don't restore CR message
	// s.Require().Equal(resp.Messages()[0].ID, contactRequest.ID)
	s.Require().Equal(common.ContactRequestStateAccepted, contactRequestMsg.ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the sender is added to our contacts
	contacts := bob.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := bob.MutualContacts()
	s.Require().Len(mutualContacts, 1)
}
