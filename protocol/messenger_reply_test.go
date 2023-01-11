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
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerReplySuite(t *testing.T) {
	suite.Run(t, new(MessengerReplySuite))
}

type MessengerReplySuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerReplySuite) SetupTest() {
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

func (s *MessengerReplySuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerReplySuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerReplySuite) TestReceiveReply() {
	alice := s.m
	alice.account = &multiaccounts.Account{KeyUID: "0xdeadbeef"}
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	bob, err := newMessengerWithKey(s.shh, key, s.logger, nil)
	s.Require().NoError(err)

	chatID := statusChatID

	chat := CreatePublicChat(chatID, alice.transport)

	err = alice.SaveChat(chat)
	s.Require().NoError(err)

	_, err = alice.Join(chat)
	s.Require().NoError(err)

	err = bob.SaveChat(chat)
	s.Require().NoError(err)

	_, err = bob.Join(chat)
	s.Require().NoError(err)

	// Send chat message from alice to bob
	message := buildTestMessage(*chat)
	_, err = alice.SendChatMessage(context.Background(), message)
	s.NoError(err)

	// Wait for message to arrive to bob
	response, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Messages(), 1)
	s.Require().False(response.Messages()[0].Replied)

	messageID := response.Messages()[0].ID

	// Bob replies to the message
	replyMessage := buildTestMessage(*chat)
	replyMessage.ResponseTo = messageID
	_, err = bob.SendChatMessage(context.Background(), replyMessage)
	s.NoError(err)

	// Wait for reply to arrive to alice
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	messages := response.Messages()
	s.Require().Len(messages, 2)
	//* Verify that the reply reponds to the original message
	messageToCheck := messages[0]
	if messageToCheck.ResponseTo == "" || messageToCheck.ResponseTo != messageID {
		// We need to use the second message in the response. They got out of order by accident
		messageToCheck = messages[1]
	}
	s.Require().True(messageToCheck.ResponseTo == messageID)
	// Verify that it's replied
	s.Require().True(messageToCheck.Replied)

	s.Require().NoError(bob.Shutdown())
}
