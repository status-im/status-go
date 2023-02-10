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

// NOTE(cammellos): Disabling for hotfix
func (s *MessengerContactRequestSuite) testReceiveAndAcceptContactRequest() { //nolint: unused

	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      types.Hex2Bytes(contactID),
		Message: messageText,
	}

	// Send contact request
	resp, err := s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(common.ContactRequestStatePending, resp.Messages()[0].ContactRequestState)

	// Make sure it's not returned as coming from us
	contactRequests, _, err := s.m.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 0)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStatePending, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, false)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Accepted, false)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Dismissed, false)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Make sure it's the pending contact requests
	contactRequests, _, err = theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 1)
	s.Require().Equal(contactRequests[0].ContactRequestState, common.ContactRequestStatePending)

	// Accept contact request, receiver side
	resp, err = theirMessenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(contactRequests[0].ID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequests[0].ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the sender is added to our contacts
	contacts = theirMessenger.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := theirMessenger.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, true)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Accepted, true)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Dismissed, false)

	// Make sure the message is updated, sender s2de
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts = s.m.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())
}

func (s *MessengerContactRequestSuite) TestReceiveAndDismissContactRequest() {

	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      types.Hex2Bytes(contactID),
		Message: messageText,
	}

	// Send contact request
	resp, err := s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(common.ContactRequestStatePending, resp.Messages()[0].ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateSent, resp.Contacts[0].ContactRequestLocalState)

	// Make sure it's not returned as coming from us
	contactRequests, _, err := s.m.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 0)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStatePending, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, false)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Accepted, false)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Dismissed, false)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Check contact request has been received
	s.Require().NoError(err)

	// Make sure it's the pending contact requests
	contactRequests, _, err = theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 1)
	s.Require().Equal(contactRequests[0].ContactRequestState, common.ContactRequestStatePending)

	// Dismiss contact request, receiver side
	resp, err = theirMessenger.DismissContactRequest(context.Background(), &requests.DismissContactRequest{ID: types.Hex2Bytes(contactRequests[0].ID)})
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateDismissed, resp.Contacts[0].ContactRequestLocalState)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateDismissed, resp.Messages()[0].ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequests[0].ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Read, true)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Accepted, false)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].Dismissed, true)
	s.Require().Equal(common.ContactRequestStateDismissed, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Make sure the sender is not added to our contacts
	contacts = theirMessenger.AddedContacts()
	s.Require().Len(contacts, 0)
}

// NOTE(cammellos): Disabling for hotfix
func (s *MessengerContactRequestSuite) testReceiveAcceptAndRetractContactRequest() { //nolint: unused

	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	s.Require().NoError(theirMessenger.settings.SaveSettingField(settings.MutualContactEnabled, true))

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      types.Hex2Bytes(contactID),
		Message: messageText,
	}

	// Send contact request
	resp, err := s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateSent, resp.Contacts[0].ContactRequestLocalState)

	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(common.ContactRequestStatePending, resp.Messages()[0].ContactRequestState)

	// Make sure it's not returned as coming from us
	contactRequests, _, err := s.m.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 0)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStatePending, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Make sure it's the pending contact requests
	contactRequests, _, err = theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 1)
	s.Require().Equal(contactRequests[0].ContactRequestState, common.ContactRequestStatePending)

	cid := resp.ActivityCenterNotifications()[0].Message.ID
	// Accept contact request, receiver side
	resp, err = theirMessenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(cid)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequests[0].ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Make sure the sender is added to our contacts
	contacts = theirMessenger.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := theirMessenger.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0 && len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the message is updated, sender side
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts = s.m.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	resp, err = s.m.RetractContactRequest(&requests.RetractContactRequest{ContactID: types.Hex2Bytes(contactID)})
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
			return len(r.Contacts) > 0
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

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestRemoteState)
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

// NOTE(cammellos): disabling for hotfix
func (s *MessengerContactRequestSuite) testReceiveAndAcceptContactRequestTwice() { //nolint: unused

	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      types.Hex2Bytes(contactID),
		Message: messageText,
	}

	// Send contact request
	resp, err := s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(common.ContactRequestStatePending, resp.Messages()[0].ContactRequestState)

	// Make sure it's not returned as coming from us
	contactRequests, _, err := s.m.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 0)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStatePending, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Make sure it's the pending contact requests
	contactRequests, _, err = theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 1)
	s.Require().Equal(contactRequests[0].ContactRequestState, common.ContactRequestStatePending)

	// Accept contact request, receiver side
	resp, err = theirMessenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(contactRequests[0].ID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequests[0].ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the sender is added to our contacts
	contacts = theirMessenger.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := theirMessenger.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Make sure the message is updated, sender s2de
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts = s.m.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Resend contact request with higher clock value
	resp, err = s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) == 1 && r.Messages()[0].ID == resp.Messages()[0].ID
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Nothing should have changed, on both sides
	mutualContacts = s.m.MutualContacts()
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
	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

	request := &requests.SendContactRequest{
		ID:      types.Hex2Bytes(contactID),
		Message: messageText,
	}

	// Send contact request
	resp, err := s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(common.ContactRequestStatePending, resp.Messages()[0].ContactRequestState)

	// Make sure it's not returned as coming from us
	contactRequests, _, err := s.m.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 0)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			contactRequests, _, err := theirMessenger.PendingContactRequests("", 10)
			if err != nil {
				return false
			}
			return len(contactRequests) == 1
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStatePending, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Make sure it's the pending contact requests
	contactRequests, _, err = theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 1)
	s.Require().Equal(contactRequests[0].ContactRequestState, common.ContactRequestStatePending)

	// Accept latest contact request, receiver side
	resp, err = theirMessenger.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequests[0].ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the sender is added to our contacts
	contacts = theirMessenger.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := theirMessenger.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure the message is updated, sender side
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 2)
	// TODO(cammellos): This code duplicates contact requests
	// this is a known issue, we want to merge this quickly
	// for RC(1.21), but will be addresse immediately after
	/*
		s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
		s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

		// Check activity center notification is of the right type
		s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
		s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
		s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

		// Make sure we consider them a mutual contact, sender side
		mutualContacts = s.m.MutualContacts()
		s.Require().Len(mutualContacts, 1)

		// Check the contact state is correctly set
		s.Require().Len(resp.Contacts, 1)
		s.Require().True(resp.Contacts[0].mutual()) */
}

func (s *MessengerContactRequestSuite) TestDismissLatestContactRequestForContact() {

	messageText := "hello!"

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

	request := &requests.SendContactRequest{
		ID:      types.Hex2Bytes(contactID),
		Message: messageText,
	}

	// Send contact request
	resp, err := s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(common.ContactRequestStatePending, resp.Messages()[0].ContactRequestState)

	// Make sure it's not returned as coming from us
	contactRequests, _, err := s.m.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 0)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStatePending, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Make sure it's the pending contact requests
	contactRequests, _, err = theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 1)
	s.Require().Equal(contactRequests[0].ContactRequestState, common.ContactRequestStatePending)

	// Dismiss latest contact request, receiver side
	resp, err = theirMessenger.DismissLatestContactRequestForContact(context.Background(), &requests.DismissLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)
	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
	s.Require().Equal(common.ContactRequestStateDismissed, resp.Messages()[0].ContactRequestState)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), contactRequests[0].ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateDismissed, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

}

func (s *MessengerContactRequestSuite) TestReceiveAndAcceptLegacyContactRequest() {

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.AddContact{
		ID: types.Hex2Bytes(contactID),
	}

	// Send contact request
	resp, err := s.m.AddContact(context.Background(), request)
	s.Require().NoError(err)

	s.Require().NotNil(resp)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)

	s.Require().NoError(err)

	notification := resp.ActivityCenterNotifications()[0]

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, notification.Type)
	s.Require().NotNil(notification.Type)
	s.Require().Equal(common.ContactRequestStatePending, notification.Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Accept contact request, receiver side
	resp, err = theirMessenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(notification.Message.ID)})
	s.Require().NoError(err)

	// Make sure the message is updated
	s.Require().NotNil(resp)

	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(resp.ActivityCenterNotifications()[0].ID.String(), notification.Message.ID)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStateAccepted, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the sender is added to our contacts
	contacts = theirMessenger.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := theirMessenger.MutualContacts()
	s.Require().Len(mutualContacts, 1)
}

func (s *MessengerContactRequestSuite) TestLegacyContactRequestNotifications() {

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.AddContact{
		ID: types.Hex2Bytes(contactID),
	}

	// Send legacy contact request
	resp, err := s.m.AddContact(context.Background(), request)
	s.Require().NoError(err)

	s.Require().NotNil(resp)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)

	s.Require().NoError(err)

	notification := resp.ActivityCenterNotifications()[0]

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, notification.Type)
	s.Require().NotNil(notification.Type)
	s.Require().Equal(common.ContactRequestStatePending, notification.Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)
}

func (s *MessengerContactRequestSuite) TestReceiveMultipleLegacy() {

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	s.Require().NoError(theirMessenger.settings.SaveSettingField(settings.MutualContactEnabled, true))
	s.Require().NoError(s.m.settings.SaveSettingField(settings.MutualContactEnabled, true))

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.AddContact{
		ID: types.Hex2Bytes(contactID),
	}

	// Send legacy contact request
	resp, err := s.m.AddContact(context.Background(), request)
	s.Require().NoError(err)

	s.Require().NotNil(resp)

	// Make sure contact is added on the sender side
	contacts := s.m.AddedContacts()
	s.Require().Len(contacts, 1)
	s.Require().Equal(ContactRequestStateSent, contacts[0].ContactRequestLocalState)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)

	s.Require().NoError(err)

	notification := resp.ActivityCenterNotifications()[0]

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, notification.Type)
	s.Require().NotNil(notification.Type)
	s.Require().Equal(common.ContactRequestStatePending, notification.Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Remove contact

	_, err = s.m.RetractContactRequest(&requests.RetractContactRequest{ContactID: types.Hex2Bytes(contactID)})
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure it's not a contact anymore
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestRemoteState)

	// Re-add user
	resp, err = s.m.AddContact(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)

	s.Require().NoError(err)

	notification = resp.ActivityCenterNotifications()[0]

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, notification.Type)
	s.Require().NotNil(notification.Type)
	s.Require().Equal(common.ContactRequestStatePending, notification.Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

}

// NOTE(cammellos): Disabling for hotfix
func (s *MessengerContactRequestSuite) testAcceptLatestLegacyContactRequestForContact() { // nolint: unused

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

	request := &requests.AddContact{
		ID: types.Hex2Bytes(contactID),
	}

	// Send contact request
	_, err = s.m.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStatePending, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Accept latest contact request, receiver side
	resp, err = theirMessenger.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

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
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts = s.m.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())
}

func (s *MessengerContactRequestSuite) TestPairedDevicesRemoveContact() {
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

	contactID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))
	myID := types.EncodeHex(crypto.FromECDSAPub(&alice1.identity.PublicKey))

	request := &requests.AddContact{
		ID: types.Hex2Bytes(contactID),
	}

	// Send contact request
	_, err = alice1.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// it should show up on device 2
	resp, err := WaitOnMessengerResponse(
		alice2,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(resp.Contacts[0].ContactRequestLocalState, ContactRequestStateSent)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check activity center notification is of the right type
	s.Require().Len(resp.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
	s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
	s.Require().Equal(common.ContactRequestStatePending, resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Accept latest contact request, receiver side
	resp, err = bob.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Make sure the sender is added to our contacts
	contacts := bob.AddedContacts()
	s.Require().Len(contacts, 1)

	// Make sure we consider them a mutual contact, receiver side
	mutualContacts := bob.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		alice1,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts = alice1.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		alice2,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts = alice2.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	resp, err = alice1.RetractContactRequest(&requests.RetractContactRequest{ContactID: types.Hex2Bytes(bob.myHexIdentity())})
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Len(resp.Contacts, 1)
	s.Require().False(resp.Contacts[0].hasAddedUs())
	s.Require().False(resp.Contacts[0].added())

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, resp.Contacts[0].ContactRequestRemoteState)

	// Check on bob side
	resp, err = WaitOnMessengerResponse(
		bob,
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
	// Alice sends a contact request to bob
	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))
	myID := types.EncodeHex(crypto.FromECDSAPub(&alice1.identity.PublicKey))

	request := &requests.AddContact{
		ID: types.Hex2Bytes(bobID),
	}

	_, err = alice1.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Bob accepts the contact request
	_, err = bob.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Alice receives the accepted confirmation
	resp, err = WaitOnMessengerResponse(
		alice1,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts := alice1.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	// Alice resets her device
	alice2, err := newMessengerWithKey(s.shh, s.m.identity, s.logger, nil)
	s.Require().NoError(err)

	_, err = alice2.Start()
	s.Require().NoError(err)

	// adds bob again to her device
	request = &requests.AddContact{
		ID: types.Hex2Bytes(bobID),
	}

	_, err = alice2.AddContact(context.Background(), request)
	s.Require().NoError(err)

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
	s.Require().True(resp.Contacts[0].mutual())
}

// The scenario tested is as follow:
// 1) Alice sends a contact request to Bob
// 2) Bob accepts the contact request
// 3) Alice restores state on a different device
// 4) Bob sends a message to alice
// Alice will show a contact request from bob
func (s *MessengerContactRequestSuite) TestAliceRecoverStateReceiveContactRequest() {
	// Alice sends a contact request to bob
	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))
	myID := types.EncodeHex(crypto.FromECDSAPub(&alice1.identity.PublicKey))

	request := &requests.AddContact{
		ID: types.Hex2Bytes(bobID),
	}

	_, err = alice1.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Bob accepts the contact request
	_, err = bob.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Alice receives the accepted confirmation
	resp, err = WaitOnMessengerResponse(
		alice1,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts := alice1.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

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
	resp, err = WaitOnMessengerResponse(
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
	// Alice sends a contact request to bob
	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))
	myID := types.EncodeHex(crypto.FromECDSAPub(&alice1.identity.PublicKey))

	request := &requests.AddContact{
		ID: types.Hex2Bytes(bobID),
	}

	_, err = alice1.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Bob accepts the contact request
	_, err = bob.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Alice receives the accepted confirmation
	resp, err = WaitOnMessengerResponse(
		alice1,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts := alice1.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	_, err = alice1.RetractContactRequest(&requests.RetractContactRequest{ContactID: types.Hex2Bytes(bob.myHexIdentity())})
	s.Require().NoError(err)

	// adds bob again to her device
	request = &requests.AddContact{
		ID: types.Hex2Bytes(bobID),
	}

	_, err = alice1.AddContact(context.Background(), request)
	s.Require().NoError(err)

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
	// Alice sends a contact request to bob
	alice1 := s.m

	bob := s.newMessenger(s.shh)
	_, err := bob.Start()
	s.Require().NoError(err)

	bobID := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))
	myID := types.EncodeHex(crypto.FromECDSAPub(&alice1.identity.PublicKey))

	request := &requests.AddContact{
		ID: types.Hex2Bytes(bobID),
	}

	_, err = alice1.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0
		},
		"no messages",
	)

	// Check contact request has been received
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().Equal(ContactRequestStateReceived, resp.Contacts[0].ContactRequestRemoteState)

	// Bob accepts the contact request
	_, err = bob.AcceptLatestContactRequestForContact(context.Background(), &requests.AcceptLatestContactRequestForContact{ID: types.Hex2Bytes(myID)})
	s.Require().NoError(err)

	// Alice receives the accepted confirmation
	resp, err = WaitOnMessengerResponse(
		alice1,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure we consider them a mutual contact, sender side
	mutualContacts := alice1.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())

	_, err = alice1.RetractContactRequest(&requests.RetractContactRequest{ContactID: types.Hex2Bytes(bob.myHexIdentity())})
	s.Require().NoError(err)

	// adds bob again to her device
	request = &requests.AddContact{
		ID: types.Hex2Bytes(bobID),
	}

	_, err = alice1.AddContact(context.Background(), request)
	s.Require().NoError(err)

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
