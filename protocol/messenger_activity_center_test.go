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
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerActivityCenterMessageSuite(t *testing.T) {
	suite.Run(t, new(MessengerActivityCenterMessageSuite))
}

type MessengerActivityCenterMessageSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerActivityCenterMessageSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerActivityCenterMessageSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerActivityCenterMessageSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerActivityCenterMessageSuite) TestDeleteOneToOneChat() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	r := &requests.SendContactRequest{
		ID:      types.Hex2Bytes(s.m.myHexIdentity()),
		Message: "hello",
	}
	sendResponse, err := theirMessenger.SendContactRequest(context.Background(), r)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	// accept notification
	_, err = s.m.AcceptAllActivityCenterNotifications(context.Background())
	s.Require().NoError(err)

	request := &requests.DeactivateChat{ID: response.Chats()[0].ID}
	response, err = s.m.DeactivateChat(request)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	deletedChat := response.Chats()[0]
	s.Require().NotEmpty(deletedChat.DeletedAtClockValue)

	// Make sure deleted at clock value is greater
	theirChat.LastClockValue = deletedChat.DeletedAtClockValue + 1
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	// Send another message
	inputMessage := buildTestMessage(*theirChat)
	sendResponse, err = theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
}
