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
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerDeleteMessageSuite(t *testing.T) {
	suite.Run(t, new(MessengerDeleteMessageSuite))
}

type MessengerDeleteMessageSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerDeleteMessageSuite) SetupTest() {
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

func (s *MessengerDeleteMessageSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerDeleteMessageSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerDeleteMessageSuite) TestDeleteMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	messageID := sendResponse.Messages()[0].ID

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	ogMessage := sendResponse.Messages()[0]

	sendResponse, err = theirMessenger.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 0)
	s.Require().Len(sendResponse.RemovedMessages(), 1)
	s.Require().Equal(messageID, sendResponse.RemovedMessages()[0].MessageID)
	s.Require().Equal(sendResponse.RemovedMessages()[0].DeletedBy, "")
	s.Require().Len(sendResponse.Chats(), 1)
	// LastMessage is removed
	s.Require().Nil(sendResponse.Chats()[0].LastMessage)

	// Main instance user attempts to delete the message it received from theirMessenger
	_, err = s.m.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().ErrorContains(err, "Chat not found")
}

func (s *MessengerDeleteMessageSuite) TestDeleteMessagePreviousLastMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage1 := buildTestMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage1)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	inputMessage2 := buildTestMessage(*theirChat)
	sendResponse, err = theirMessenger.SendChatMessage(context.Background(), inputMessage2)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	messageID := sendResponse.Messages()[0].ID

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 2)

	ogMessage := sendResponse.Messages()[0]

	sendResponse, err = theirMessenger.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 0)
	s.Require().Len(sendResponse.RemovedMessages(), 1)
	s.Require().Equal(messageID, sendResponse.RemovedMessages()[0].MessageID)
	s.Require().Len(sendResponse.Chats(), 1)
	// LastMessage is updated to previous message
	s.Require().NotNil(sendResponse.Chats()[0].LastMessage)
	s.Require().Equal(inputMessage1.ID, sendResponse.Chats()[0].LastMessage.ID)

}

func (s *MessengerDeleteMessageSuite) TestDeleteWrongMessageType() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestGapMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	ogMessage := sendResponse.Messages()[0]

	// Delete should not work
	_, err = theirMessenger.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().Equal(ErrInvalidDeleteTypeAuthor, err)
}

// TODO fix activity center notifications not being deleted when a message is deleted

func (s *MessengerDeleteMessageSuite) TestDeleteMessageFirstThenMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	contact, err := BuildContactFromPublicKey(&theirMessenger.identity.PublicKey)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)
	messageID := "message-id"

	inputMessage := buildTestMessage(*theirChat)
	inputMessage.Clock = 1
	deleteMessage := DeleteMessage{
		DeleteMessage: protobuf.DeleteMessage{
			Clock:       2,
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			MessageId:   messageID,
			ChatId:      theirChat.ID,
		},
		From: common.PubkeyToHex(&theirMessenger.identity.PublicKey),
	}

	state := &ReceivedMessageState{
		Response: &MessengerResponse{},
	}

	// Handle Delete first
	err = s.m.HandleDeleteMessage(state, deleteMessage)
	s.Require().NoError(err)

	// // Handle chat message
	state = &ReceivedMessageState{
		Response: &MessengerResponse{},
		CurrentMessageState: &CurrentMessageState{
			Message:          inputMessage.ChatMessage,
			MessageID:        messageID,
			WhisperTimestamp: s.m.getTimesource().GetCurrentTime(),
			Contact:          contact,
			PublicKey:        &theirMessenger.identity.PublicKey,
		},
	}
	err = s.m.HandleChatMessage(state)
	s.Require().NoError(err)
	s.Require().Len(state.Response.Messages(), 0) // Message should not be added to response
	s.Require().Len(state.Response.RemovedMessages(), 0)
	s.Require().Nil(state.Response.Chats()[0].LastMessage)
}
