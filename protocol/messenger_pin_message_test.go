package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestMessengerPinMessageSuite(t *testing.T) {
	suite.Run(t, new(MessengerPinMessageSuite))
}

type MessengerPinMessageSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerPinMessageSuite) TestPinMessage() {
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

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)

	pinMessage := &common.PinMessage{
		LocalChatID: theirChat.ID,
	}
	pinMessage.MessageId = inputMessage.ID
	pinMessage.Pinned = true
	pinMessage.ChatId = theirChat.ID
	sendResponse, err = theirMessenger.SendPinMessage(context.Background(), pinMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.PinMessages(), 1)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			// we pull as well the original pinned message
			return len(r.PinMessages()) == 1 && len(r.Messages()) == 2
		},
		"pin message not received",
	)
	s.Require().NoError(err)

	receivedPinMessage := response.PinMessages()[0]
	s.Require().True(receivedPinMessage.Pinned)
	s.Require().NotNil(receivedPinMessage.Message)
	s.Require().Equal(inputMessage.ID, receivedPinMessage.Message.Message.ID)
	messages := response.Messages()

	var originalMessage *common.Message
	var notificationMessage *common.Message

	if messages[0].ID == inputMessage.ID {
		originalMessage = messages[0]
		notificationMessage = messages[1]
	} else {
		originalMessage = messages[1]
		notificationMessage = messages[0]
	}

	s.Require().Equal(theirMessenger.myHexIdentity(), notificationMessage.From)
	s.Require().Equal(inputMessage.ID, notificationMessage.ResponseTo)
	s.Require().Equal(protobuf.ChatMessage_SYSTEM_MESSAGE_PINNED_MESSAGE, notificationMessage.ContentType)

	s.Require().Equal(inputMessage.ID, originalMessage.ID)
}

func (s *MessengerPinMessageSuite) TestPinMessageOutOfOrder() {
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

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Equal(response.Chats()[0].UnviewedMessagesCount, uint(1))

	// Unpin with higher clock

	unpinMessage := &protobuf.PinMessage{
		ChatId:      theirChat.ID,
		MessageId:   inputMessage.ID,
		Pinned:      false,
		Clock:       3,
		MessageType: protobuf.MessageType_ONE_TO_ONE,
	}

	handlePinMessageResponse := &MessengerResponse{}
	err = theirMessenger.handlePinMessage(
		&Contact{ID: s.m.myHexIdentity()},
		1000,
		handlePinMessageResponse,
		unpinMessage,
	)
	s.Require().NoError(err)

	s.Require().Len(handlePinMessageResponse.PinMessages(), 1)
	s.Require().False(handlePinMessageResponse.PinMessages()[0].Pinned)
	s.Require().Len(handlePinMessageResponse.Chats(), 1)
	s.Require().Equal(handlePinMessageResponse.Chats()[0].UnviewedMessagesCount, uint(0))

	// Pin with lower clock in response

	pinMessage := &protobuf.PinMessage{
		ChatId:      theirChat.ID,
		MessageId:   inputMessage.ID,
		Pinned:      true,
		Clock:       2,
		MessageType: protobuf.MessageType_ONE_TO_ONE,
	}

	err = theirMessenger.handlePinMessage(
		&Contact{ID: s.m.myHexIdentity()},
		1000,
		handlePinMessageResponse,
		pinMessage,
	)
	s.Require().NoError(err)

	s.Require().Len(handlePinMessageResponse.PinMessages(), 1)
	s.Require().False(handlePinMessageResponse.PinMessages()[0].Pinned)
	s.Require().Len(handlePinMessageResponse.Chats(), 1)
	s.Require().Equal(handlePinMessageResponse.Chats()[0].UnviewedMessagesCount, uint(0))

	// Pin with lower clock not in response

	handlePinMessageResponse = &MessengerResponse{}
	pinMessage = &protobuf.PinMessage{
		ChatId:      theirChat.ID,
		MessageId:   inputMessage.ID,
		Pinned:      true,
		Clock:       1,
		MessageType: protobuf.MessageType_ONE_TO_ONE,
	}

	err = theirMessenger.handlePinMessage(
		&Contact{ID: s.m.myHexIdentity()},
		1000,
		handlePinMessageResponse,
		pinMessage,
	)
	s.Require().NoError(err)

	s.Require().Len(handlePinMessageResponse.PinMessages(), 0)
}
