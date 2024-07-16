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
			BridgeName:      "discord",
			UserName:        "user1",
			UserAvatar:      "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
			UserID:          "123",
			Content:         "text1",
			MessageID:       "456",
			ParentMessageID: "789",
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
	s.Require().Equal(receivedBridgeMessagePayload.UserAvatar, "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==")
	s.Require().Equal(receivedBridgeMessagePayload.UserID, "123")
	s.Require().Equal(receivedBridgeMessagePayload.MessageID, "456")
	s.Require().Equal(receivedBridgeMessagePayload.ParentMessageID, "789")
}

func (s *BridgeMessageSuite) TestSearchForDiscordMessages() {
	//send bridged message
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(chat)
	s.NoError(err)

	bridgeMessage := buildTestMessage(*chat)
	bridgeMessage.ContentType = protobuf.ChatMessage_BRIDGE_MESSAGE
	bridgeMessage.Payload = &protobuf.ChatMessage_BridgeMessage{
		BridgeMessage: &protobuf.BridgeMessage{
			BridgeName:      "discord",
			UserName:        "user1",
			UserAvatar:      "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
			UserID:          "123",
			Content:         "bridged discord message",
			MessageID:       "456",
			ParentMessageID: "789",
		},
	}

	_, err = s.m.SendChatMessage(context.Background(), bridgeMessage)
	s.NoError(err)

	// Search for the message
	messages, err := s.m.AllMessageByChatIDWhichMatchTerm(chat.ID, "bridged", true)
	s.NoError(err)
	s.Require().Len(messages, 1)

	//send discord import message
	discordMessage := &protobuf.DiscordMessage{
		Id:        "discordMessageID",
		Type:      "Default",
		Timestamp: "123456",
		Content:   "discord import message",
		Author: &protobuf.DiscordMessageAuthor{
			Id: "2",
		},
		Reference: &protobuf.DiscordMessageReference{},
	}

	err = s.m.persistence.SaveDiscordMessage(discordMessage)
	s.NoError(err)
	bridgeMessage = buildTestMessage(*chat)
	bridgeMessage.ContentType = protobuf.ChatMessage_DISCORD_MESSAGE
	bridgeMessage.Payload = &protobuf.ChatMessage_DiscordMessage{
		DiscordMessage: discordMessage,
	}

	_, err = s.m.SendChatMessage(context.Background(), bridgeMessage)
	s.NoError(err)

	// Search for the message
	messages, err = s.m.AllMessageByChatIDWhichMatchTerm(chat.ID, "import", true)
	s.NoError(err)
	s.Require().Len(messages, 1)

	// Search for discord messages
	messages, err = s.m.AllMessageByChatIDWhichMatchTerm(chat.ID, "discord", true)
	s.NoError(err)
	s.Require().Len(messages, 2)

	// Search for discord messages using AllMessagesFromChatsAndCommunitiesWhichMatchTerm
	chatIDs := make([]string, 1)
	chatIDs = append(chatIDs, chat.ID)
	messages, err = s.m.AllMessagesFromChatsAndCommunitiesWhichMatchTerm(make([]string, 0), chatIDs, "discord", true)
	s.NoError(err)
	s.Require().Len(messages, 2)

	// Same with case insensitive
	messages, err = s.m.AllMessagesFromChatsAndCommunitiesWhichMatchTerm(make([]string, 0), chatIDs, "discord", false)
	s.NoError(err)
	s.Require().Len(messages, 2)
}
