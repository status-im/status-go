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
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerEditMessageSuite(t *testing.T) {
	suite.Run(t, new(MessengerEditMessageSuite))
}

type MessengerEditMessageSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerEditMessageSuite) SetupTest() {
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

func (s *MessengerEditMessageSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerEditMessageSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerEditMessageSuite) TestEditMessage() {
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

func (s *MessengerEditMessageSuite) TestEditMessageActivityCenter() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
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
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	// Check we can fetch the notification
	paginationResponse, err := s.m.ActivityCenterNotifications("", 10)
	s.Require().NoError(err)
	s.Require().Len(paginationResponse.Notifications, 1)

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

	// Check we can fetch the notification
	paginationResponse, err = s.m.ActivityCenterNotifications("", 10)
	s.Require().NoError(err)
	s.Require().Len(paginationResponse.Notifications, 1)

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

	response = &MessengerResponse{}

	err = s.m.HandleEditMessage(response, editMessage)
	// It should error as the user can't edit this message
	s.Require().Error(err)

	// Edit with a newer clock value

	response = &MessengerResponse{}

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

	err = s.m.HandleEditMessage(response, editMessage)
	s.Require().NoError(err)
	// It save the edit
	s.Require().Len(response.Messages(), 1)
	s.Require().Len(response.Chats(), 1)
	s.Require().NotNil(response.Chats()[0].LastMessage)
	s.Require().NotEmpty(response.Chats()[0].LastMessage.EditedAt)

	editedMessage = response.Messages()[0]

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

	response = &MessengerResponse{}

	err = s.m.HandleEditMessage(response, editMessage)
	// It should error as the user can't edit this message
	s.Require().NoError(err)
	// It discards the edit
	s.Require().Len(response.Messages(), 0)
}

func (s *MessengerEditMessageSuite) TestEditMessageFirstEditsThenMessage() {
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

	response := &MessengerResponse{}

	// Handle edit first
	err = s.m.HandleEditMessage(response, editMessage)
	s.Require().NoError(err)

	// Handle chat message
	response = &MessengerResponse{}
	state := &ReceivedMessageState{
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

	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "id", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourChat := response.Chats()[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

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
