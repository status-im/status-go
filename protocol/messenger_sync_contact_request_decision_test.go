package protocol

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
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

	// send contact request to m/m2, m and m2 are paired
	request := &requests.AddContact{ID: common.PubkeyToHex(&s.m2.identity.PublicKey)}
	_, err := userB.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// check m and m2 received contact request
	var contactRequestMessageID types.HexBytes
	_, err = WaitOnMessengerResponse(s.m, func(r *MessengerResponse) bool {
		if len(r.ActivityCenterNotifications()) > 0 {
			for _, n := range r.ActivityCenterNotifications() {
				if n.Type == ActivityCenterNotificationTypeContactRequest {
					contactRequestMessageID = n.ID
					return true
				}
			}
		}
		return false
	}, "contact request not received on device 1")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		if len(r.ActivityCenterNotifications()) > 0 {
			for _, n := range r.ActivityCenterNotifications() {
				if n.Type == ActivityCenterNotificationTypeContactRequest {
					return true
				}
			}
		}
		return false
	}, "contact request not received on device 2")
	s.Require().NoError(err)

	// m accept contact request from userB
	_, err = s.m.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: contactRequestMessageID})
	s.Require().NoError(err)

	// check sync contact request decision processed for m2
	_, err = WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		return len(r.Contacts) > 0
	}, "contact request not accepted on device 2")
	s.Require().NoError(err)

	numAcceptMessageReceived := 0
	retryError := errors.New("retry error")
	err = tt.RetryWithBackOff(func() error {
		r, err := userB.RetrieveAll()
		if err != nil {
			return err
		}
		for _, m := range r.Messages() {
			if m.GetContentType() == protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED {
				numAcceptMessageReceived++
			}
		}
		// always return error so that we retry until timeout
		return retryError
	})
	s.Require().ErrorIs(err, retryError)
	s.Require().Equal(1, numAcceptMessageReceived, "we should receive only 1 message(contact request accepted)")
}
