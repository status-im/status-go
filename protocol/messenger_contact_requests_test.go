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
	"github.com/status-im/status-go/protocol/common"
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

func (s *MessengerContactRequestSuite) TestReceiveAndAcceptContactRequest() {

	messageText := "hello!"
	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

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

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0 },
		"no messages",
	)

        // Check contact request has been received
	s.Require().NoError(err)
	contactRequest, err := theirMessenger.persistence.GetReceivedContactRequest(myID)
	s.Require().NoError(err)
	s.Require().NotNil(contactRequest)

        // Check activity center notification is of the right type
        s.Require().Len(resp.ActivityCenterNotifications(), 1)
        s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
        s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
        s.Require().Equal(common.ContactRequestStatePending,resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

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

        // Make sure the sender is added to our contacts
	contacts = theirMessenger.AddedContacts()
	s.Require().Len(contacts, 1)

        // Make sure we consider them a mutual contact, receiver side
	mutualContacts := theirMessenger.MutualContacts()
	s.Require().Len(mutualContacts, 1)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 && len(r.ActivityCenterNotifications()) > 0 },
		"no messages",
	)

        // Check activity center notification is of the right type
        s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
        s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
        s.Require().Equal(common.ContactRequestStateAccepted,resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

        // Make sure the message is updated, sender side
        s.Require().NotNil(resp)
        s.Require().Len(resp.Messages(), 1)
        s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
        s.Require().Equal(common.ContactRequestStateAccepted, resp.Messages()[0].ContactRequestState)

        // Make sure we consider them a mutual contact, sender side
	mutualContacts = s.m.MutualContacts()
	s.Require().Len(mutualContacts, 1)
}

func (s *MessengerContactRequestSuite) TestReceiveAndDeclineContactRequest() {

	messageText := "hello!"
	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

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

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.ActivityCenterNotifications()) > 0 },
		"no messages",
	)
        s.Require().NoError(err)

        // Check activity center notification is of the right type
        s.Require().Equal(ActivityCenterNotificationTypeContactRequest, resp.ActivityCenterNotifications()[0].Type)
        s.Require().NotNil(resp.ActivityCenterNotifications()[0].Message)
        s.Require().Equal(common.ContactRequestStatePending,resp.ActivityCenterNotifications()[0].Message.ContactRequestState)

        // Check contact request has been received
	s.Require().NoError(err)
	contactRequest, err := theirMessenger.persistence.GetReceivedContactRequest(myID)
	s.Require().NoError(err)
	s.Require().NotNil(contactRequest)

        // Make sure it's the pending contact requests
	contactRequests, _, err = theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 1)
	s.Require().Equal(contactRequests[0].ContactRequestState, common.ContactRequestStatePending)

        // Decline contact request, receiver side
	resp, err = theirMessenger.DeclineContactRequest(context.Background(), &requests.DeclineContactRequest{ID: types.Hex2Bytes(contactRequests[0].ID)})
	s.Require().NoError(err)

        // Make sure the message is updated
        s.Require().NotNil(resp)
        s.Require().Len(resp.Messages(), 1)
        s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
        s.Require().Equal(common.ContactRequestStateDeclined, resp.Messages()[0].ContactRequestState)

        // Make sure the sender is not added to our contacts
	contacts = theirMessenger.AddedContacts()
	s.Require().Len(contacts, 0)

	// Wait for the message to reach its destination
	resp, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)

        // Make sure the message is updated, sender side
        s.Require().NotNil(resp)
        s.Require().Len(resp.Messages(), 1)
        s.Require().Equal(resp.Messages()[0].ID, contactRequests[0].ID)
        s.Require().Equal(common.ContactRequestStateDeclined, resp.Messages()[0].ContactRequestState)
}
