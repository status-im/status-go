package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestBridgeMessageSuite(t *testing.T) {
	suite.Run(t, new(BridgeMessageSuite))
}

type BridgeMessageSuite struct {
	MessengerBaseTestSuite
}

func (s *BridgeMessageSuite) TestSendBridgeMessage() {
	alice := s.m
	alice.account = &multiaccounts.Account{KeyUID: "0xdeadbeef"}
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	bob, err := newMessengerWithKey(s.shh, key, s.logger, nil)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, bob)

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

	// Send bridge message

	bridgeMessage := buildTestMessage(*chat)
	bridgeMessage.ContentType = protobuf.ChatMessage_BRIDGE_MESSAGE
	bridgeMessage.Payload = &protobuf.ChatMessage_BridgeMessage{
		BridgeMessage: &protobuf.BridgeMessage{
			BridgeName: "discord",
			UserName:   "user1",
			Content:    "text1",
		},
	}

	_, err = bob.SendChatMessage(context.Background(), bridgeMessage)
	s.NoError(err)

	// Wait for the bridge message to arrive to alice
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no bridge message",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	receivedBridgeMessage := response.Messages()[0]
	s.Require().Equal(receivedBridgeMessage.ContentType, protobuf.ChatMessage_BRIDGE_MESSAGE)
	receivedBridgeMessagePayload := receivedBridgeMessage.GetBridgeMessage()
	s.Require().NotNil(receivedBridgeMessagePayload)
	s.Require().Equal(receivedBridgeMessagePayload.BridgeName, "discord")
	s.Require().Equal(receivedBridgeMessagePayload.UserName, "user1")
	s.Require().Equal(receivedBridgeMessagePayload.Content, "text1")
}
