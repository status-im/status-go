package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

type MessengerSyncContactRequestDecisionSuite struct {
	MessengerBaseTestSuite
	m2 *Messenger
}

func TestMessengerSyncContactRequestDecision(t *testing.T) {
	suite.Run(t, new(MessengerSyncContactRequestDecisionSuite))
}

func (s *MessengerSyncContactRequestDecisionSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()

	m2, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	s.m2 = m2

	PairDevices(&s.Suite, m2, s.m)
	PairDevices(&s.Suite, s.m, m2)
}

func (s *MessengerSyncContactRequestDecisionSuite) TearDownTest() {
	s.Require().NoError(s.m2.Shutdown())
	s.MessengerBaseTestSuite.TearDownTest()
}

func (s *MessengerSyncContactRequestDecisionSuite) createUserB() *Messenger {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	userB, err := newMessengerWithKey(s.shh, key, s.logger, nil)
	s.Require().NoError(err)
	return userB
}

func (s *MessengerSyncContactRequestDecisionSuite) TestSyncAcceptContactRequest() {
	userB := s.createUserB()
	defer func() {
		s.Require().NoError(userB.Shutdown())
	}()

	numM1DispatchedAcceptContactRequest := 0
	numM2DispatchedAcceptContactRequest := 0
	s.m.dispatchMessageTestCallback = func(message common.RawMessage) {
		if message.MessageType == protobuf.ApplicationMetadataMessage_ACCEPT_CONTACT_REQUEST {
			numM1DispatchedAcceptContactRequest++
		}
	}
	s.m2.dispatchMessageTestCallback = func(message common.RawMessage) {
		if message.MessageType == protobuf.ApplicationMetadataMessage_ACCEPT_CONTACT_REQUEST {
			numM2DispatchedAcceptContactRequest++
		}
	}
	// send contact request to m/m2, m and m2 are paired
	request := &requests.AddContact{ID: common.PubkeyToHex(&s.m2.identity.PublicKey)}
	_, err := userB.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// check m and m2 received contact request
	var contactRequestMessageID types.HexBytes
	receivedContactRequestCondition := func(r *MessengerResponse) bool {
		for _, n := range r.ActivityCenterNotifications() {
			if n.Type == ActivityCenterNotificationTypeContactRequest {
				contactRequestMessageID = n.ID
				return true
			}
		}
		return false
	}
	_, err = WaitOnMessengerResponse(s.m, receivedContactRequestCondition, "contact request not received on device 1")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.m2, receivedContactRequestCondition, "contact request not received on device 2")
	s.Require().NoError(err)

	// m accept contact request from userB
	_, err = s.m.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: contactRequestMessageID})
	s.Require().NoError(err)

	// check sync contact request decision processed for m2
	_, err = WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		return len(r.Contacts) > 0
	}, "contact request not accepted on device 2")
	s.Require().NoError(err)

	s.Require().Equal(1, numM1DispatchedAcceptContactRequest, "we should dispatch only 1 accept contact request message")
	s.Require().Equal(0, numM2DispatchedAcceptContactRequest, "we should not dispatch accept contact request message")
}
