package protocol

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
)

func TestMessengerRawMessageResendTestSuite(t *testing.T) {
	suite.Run(t, new(MessengerRawMessageResendTest))
}

type MessengerRawMessageResendTest struct {
	suite.Suite
	logger *zap.Logger

	aliceMessenger *Messenger
	bobMessenger   *Messenger

	aliceWaku types.Waku
	bobWaku   types.Waku

	mockedBalances communities.BalancesByChain
}

func (s *MessengerRawMessageResendTest) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	s.mockedBalances = make(communities.BalancesByChain)

	wakuNodes := CreateWakuV2Network(&s.Suite, s.logger, []string{"alice", "bob"})

	s.aliceWaku = wakuNodes[0]
	s.aliceMessenger = newTestCommunitiesMessenger(&s.Suite, s.aliceWaku, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			name:   "alice",
			logger: s.logger,
		},
		walletAddresses: []string{aliceAddress1},
		password:        accountPassword,
		mockedBalances:  &s.mockedBalances,
	})

	_, err := s.aliceMessenger.Start()
	s.Require().NoError(err)

	s.bobWaku = wakuNodes[1]
	s.bobMessenger = newTestCommunitiesMessenger(&s.Suite, s.bobWaku, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			name:   "bob",
			logger: s.logger,
		},
		walletAddresses: []string{bobAddress},
		password:        bobPassword,
		mockedBalances:  &s.mockedBalances,
	})

	_, err = s.bobMessenger.Start()
	s.Require().NoError(err)

	community, _ := createOnRequestCommunity(&s.Suite, s.aliceMessenger)
	advertiseCommunityToUserOldWay(&s.Suite, community, s.aliceMessenger, s.bobMessenger)
	joinOnRequestCommunity(&s.Suite, community.ID(), s.aliceMessenger, s.bobMessenger, bobPassword, []string{bobAddress})
}

func (s *MessengerRawMessageResendTest) TearDownTest() {
	TearDownMessenger(&s.Suite, s.aliceMessenger)
	TearDownMessenger(&s.Suite, s.bobMessenger)
	if s.aliceWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.aliceWaku).Stop())
	}
	if s.bobWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.bobWaku).Stop())
	}
	_ = s.logger.Sync()
}

func (s *MessengerRawMessageResendTest) waitForMessageSent(messageID string) {
	err := tt.RetryWithBackOff(func() error {
		rawMessage, err := s.bobMessenger.RawMessageByID(messageID)
		s.Require().NoError(err)
		s.Require().NotNil(rawMessage)
		if rawMessage.SendCount > 0 {
			return nil
		}
		return errors.New("raw message should be sent finally")
	})
	s.Require().NoError(err)
}

// TestMessageSent tests if ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN is in state `sent` without resending
func (s *MessengerRawMessageResendTest) TestMessageSent() {
	ids, err := s.bobMessenger.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN)
	s.Require().NoError(err)
	// one request to join to control node and another to privileged member
	s.Require().Len(ids, 2)

	s.waitForMessageSent(ids[0])
	s.waitForMessageSent(ids[1])
}

// TestMessageResend tests if ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN is resent
func (s *MessengerRawMessageResendTest) TestMessageResend() {
	ids, err := s.bobMessenger.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN)
	s.Require().NoError(err)
	s.Require().Len(ids, 2)
	// wait for Sent status for already sent message to make sure that sent message was delivered
	// before testing resend
	s.waitForMessageSent(ids[0])
	s.waitForMessageSent(ids[1])

	rawMessage := s.GetRequestToJoinToControlNodeRawMessage(ids)
	s.Require().NotNil(rawMessage)
	s.Require().NoError(s.bobMessenger.UpdateRawMessageSent(rawMessage.ID, false))
	s.Require().NoError(s.bobMessenger.UpdateRawMessageLastSent(rawMessage.ID, 0))

	err = tt.RetryWithBackOff(func() error {
		msg, err := s.bobMessenger.RawMessageByID(rawMessage.ID)
		s.Require().NoError(err)
		s.Require().NotNil(msg)
		if msg.SendCount < 2 {
			return errors.New("message ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN was not resent yet")
		}
		return nil
	})
	s.Require().NoError(err)

	waitOnMessengerResponse(&s.Suite, func(r *MessengerResponse) error {
		if len(r.RequestsToJoinCommunity()) > 0 {
			return nil
		}
		return errors.New("community request to join not received")
	}, s.aliceMessenger)
}

func (s *MessengerRawMessageResendTest) TestInvalidRawMessageToWatchDoesNotProduceResendLoop() {
	ids, err := s.bobMessenger.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN)
	s.Require().NoError(err)
	s.Require().Len(ids, 2)

	s.waitForMessageSent(ids[0])
	s.waitForMessageSent(ids[1])

	rawMessage := s.GetRequestToJoinToControlNodeRawMessage(ids)
	s.Require().NotNil(rawMessage)

	requestToJoinProto := &protobuf.CommunityRequestToJoin{}
	err = proto.Unmarshal(rawMessage.Payload, requestToJoinProto)
	s.Require().NoError(err)

	requestToJoinProto.DisplayName = "invalid_ID"
	payload, err := proto.Marshal(requestToJoinProto)
	s.Require().NoError(err)
	rawMessage.Payload = payload

	_, err = s.bobMessenger.AddRawMessageToWatch(rawMessage)
	s.Require().Error(err, common.ErrModifiedRawMessage)

	// simulate storing msg with modified payload, but old message ID
	_, err = s.bobMessenger.UpsertRawMessageToWatch(rawMessage)
	s.Require().NoError(err)
	s.Require().NoError(s.bobMessenger.UpdateRawMessageSent(rawMessage.ID, false))
	s.Require().NoError(s.bobMessenger.UpdateRawMessageLastSent(rawMessage.ID, 0))

	// check counter increased for invalid message to escape the loop
	err = tt.RetryWithBackOff(func() error {
		msg, err := s.bobMessenger.RawMessageByID(rawMessage.ID)
		s.Require().NoError(err)
		s.Require().NotNil(msg)
		if msg.SendCount < 2 {
			return errors.New("message ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN was not resent yet")
		}
		return nil
	})
	s.Require().NoError(err)
}

func (s *MessengerRawMessageResendTest) GetRequestToJoinToControlNodeRawMessage(ids []string) *common.RawMessage {
	for _, messageID := range ids {
		rawMessage, err := s.bobMessenger.RawMessageByID(messageID)
		s.Require().NoError(err)
		s.Require().NotNil(rawMessage)

		if rawMessage.ResendMethod == common.ResendMethodSendCommunityMessage {
			return rawMessage
		}
	}

	s.Require().FailNow("rawMessage was not found")
	return nil
}
