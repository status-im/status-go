package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerEditMessageSuite(t *testing.T) {
	suite.Run(t, new(MessengerEditMessageSuite))
}

type MessengerEditMessageSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerEditMessageSuite) TestEditMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck

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
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	ogMessage := sendResponse.Messages()[0]

	messageID, err := types.DecodeHex(ogMessage.ID)
	s.Require().NoError(err)

	editedText := "edited text"
	editedMessage := &requests.EditMessage{
		ID:   messageID,
		Text: editedText,
	}

	sendResponse, err = theirMessenger.EditMessage(context.Background(), editedMessage)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)
	s.Require().NotEmpty(sendResponse.Messages()[0].EditedAt)
	s.Require().Equal(sendResponse.Messages()[0].Text, editedText)
	s.Require().Len(sendResponse.Chats(), 1)
	s.Require().NotNil(sendResponse.Chats()[0].LastMessage)
	s.Require().NotEmpty(sendResponse.Chats()[0].LastMessage.EditedAt)

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().NotEmpty(response.Messages()[0].EditedAt)
	s.Require().False(response.Messages()[0].New)

	// Main instance user attempts to edit the message it received from theirMessenger
	editedMessage = &requests.EditMessage{
		ID:   messageID,
		Text: "edited-again text",
	}
	_, err = s.m.EditMessage(context.Background(), editedMessage)

	s.Require().Equal(ErrInvalidEditOrDeleteAuthor, err)
}

func (s *MessengerEditMessageSuite) TestEditMessageEdgeCases() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck

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
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	chat := response.Chats()[0]
	editedMessage := sendResponse.Messages()[0]

	newContactKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	wrongContact, err := BuildContactFromPublicKey(&newContactKey.PublicKey)
	s.Require().NoError(err)

	editMessage := EditMessage{
		EditMessage: protobuf.EditMessage{
			Clock:     editedMessage.Clock + 1,
			Text:      "some text",
			MessageId: editedMessage.ID,
			ChatId:    chat.ID,
		},
		From: wrongContact.ID,
	}

	state := &ReceivedMessageState{
		Response: &MessengerResponse{},
		AllChats: &chatMap{},
	}
	state.AllChats.Store(ourChat.ID, ourChat)

	err = s.m.HandleEditMessage(state, editMessage)
	// It should error as the user can't edit this message
	s.Require().Error(err)

	// Edit with a newer clock value

	contact, err := BuildContactFromPublicKey(&theirMessenger.identity.PublicKey)
	s.Require().NoError(err)

	editMessage = EditMessage{
		EditMessage: protobuf.EditMessage{
			Clock:       editedMessage.Clock + 2,
			Text:        "some text",
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			MessageId:   editedMessage.ID,
			ChatId:      chat.ID,
		},
		From: contact.ID,
	}

	err = s.m.HandleEditMessage(state, editMessage)
	s.Require().NoError(err)
	// It save the edit
	s.Require().Len(state.Response.Messages(), 1)
	s.Require().Len(state.Response.Chats(), 1)
	s.Require().NotNil(state.Response.Chats()[0].LastMessage)
	s.Require().NotEmpty(state.Response.Chats()[0].LastMessage.EditedAt)

	editedMessage = state.Response.Messages()[0]

	// In-between edit
	editMessage = EditMessage{
		EditMessage: protobuf.EditMessage{
			Clock:       editedMessage.Clock + 1,
			Text:        "some other text",
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			MessageId:   editedMessage.ID,
			ChatId:      chat.ID,
		},
		From: contact.ID,
	}

	state.Response = &MessengerResponse{}

	err = s.m.HandleEditMessage(state, editMessage)
	// It should error as the user can't edit this message
	s.Require().NoError(err)
	// It discards the edit
	s.Require().Len(state.Response.Messages(), 0)
}

func (s *MessengerEditMessageSuite) TestEditMessageFirstEditsThenMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck

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
	editMessage := EditMessage{
		EditMessage: protobuf.EditMessage{
			Clock:       2,
			Text:        "some text",
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			MessageId:   messageID,
			ChatId:      theirChat.ID,
		},
		From: common.PubkeyToHex(&theirMessenger.identity.PublicKey),
	}
	state := &ReceivedMessageState{
		Response: &MessengerResponse{},
	}

	// Handle edit first
	err = s.m.HandleEditMessage(state, editMessage)
	s.Require().NoError(err)

	// Handle chat message
	response := &MessengerResponse{}
	state = &ReceivedMessageState{
		Response: response,
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
	s.Require().Len(response.Messages(), 1)

	editedMessage := response.Messages()[0]

	s.Require().Equal(uint64(2), editedMessage.EditedAt)
}

// Test editing a message on an existing private group chat
func (s *MessengerEditMessageSuite) TestEditGroupChatMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck

	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "id", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourChat := response.Chats()[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	s.Require().NoError(makeMutualContact(s.m, &theirMessenger.identity.PublicKey))

	members := []string{common.PubkeyToHex(&theirMessenger.identity.PublicKey)}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	s.Require().False(response.Chats()[0].Active)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"no joining group event received",
	)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*ourChat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	sentMessage := sendResponse.Messages()[0]

	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	// Edit message

	messageID, err := types.DecodeHex(sentMessage.ID)
	s.Require().NoError(err)

	editedText := "edited text"
	editedMessage := &requests.EditMessage{
		ID:   messageID,
		Text: editedText,
	}

	_, err = theirMessenger.EditMessage(context.Background(), editedMessage)

	s.Require().NoError(err)

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().NotEmpty(response.Messages()[0].EditedAt)
	s.Require().False(response.Messages()[0].New)
}

func (s *MessengerEditMessageSuite) TestEditMessageWithMention() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*theirChat)
	// Send first message with no mention
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) == 1 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	// Make sure there is no mention at first
	s.Require().Equal(int(response.Chats()[0].UnviewedMessagesCount), 1)
	s.Require().Equal(int(response.Chats()[0].UnviewedMentionsCount), 0)
	s.Require().False(response.Messages()[0].Mentioned)

	ogMessage := sendResponse.Messages()[0]

	messageID, err := types.DecodeHex(ogMessage.ID)
	s.Require().NoError(err)

	// Edit the message and add a mention
	editedText := "edited text @" + common.PubkeyToHex(&s.privateKey.PublicKey)
	editedMessage := &requests.EditMessage{
		ID:   messageID,
		Text: editedText,
	}

	sendResponse, err = theirMessenger.EditMessage(context.Background(), editedMessage)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)
	s.Require().NotEmpty(sendResponse.Messages()[0].EditedAt)
	s.Require().Equal(sendResponse.Messages()[0].Text, editedText)
	s.Require().Len(sendResponse.Chats(), 1)
	s.Require().NotNil(sendResponse.Chats()[0].LastMessage)
	s.Require().NotEmpty(sendResponse.Chats()[0].LastMessage.EditedAt)
	s.Require().False(sendResponse.Messages()[0].Mentioned) // Sender is still not mentioned

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) == 1 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().NotEmpty(response.Messages()[0].EditedAt)
	s.Require().False(response.Messages()[0].New)
	// Receiver (us) is now mentioned
	s.Require().Equal(int(response.Chats()[0].UnviewedMessagesCount), 1)
	s.Require().Equal(int(response.Chats()[0].UnviewedMentionsCount), 1)
	s.Require().True(response.Messages()[0].Mentioned)

	// Edit the message again but remove the mention
	editedText = "edited text no mention"
	editedMessage = &requests.EditMessage{
		ID:   messageID,
		Text: editedText,
	}

	sendResponse, err = theirMessenger.EditMessage(context.Background(), editedMessage)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)
	s.Require().NotEmpty(sendResponse.Messages()[0].EditedAt)
	s.Require().Equal(sendResponse.Messages()[0].Text, editedText)
	s.Require().Len(sendResponse.Chats(), 1)
	s.Require().NotNil(sendResponse.Chats()[0].LastMessage)
	s.Require().NotEmpty(sendResponse.Chats()[0].LastMessage.EditedAt)
	s.Require().False(sendResponse.Messages()[0].Mentioned) // Sender is still not mentioned

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) == 1 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().NotEmpty(response.Messages()[0].EditedAt)
	s.Require().False(response.Messages()[0].New)
	// Receiver (us) is no longer mentioned
	s.Require().Equal(int(response.Chats()[0].UnviewedMessagesCount), 1) // We still have an unread message though
	s.Require().Equal(int(response.Chats()[0].UnviewedMentionsCount), 0)
	s.Require().False(response.Messages()[0].Mentioned)
}
