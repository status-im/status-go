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
	s.Require().NoError(shh.Start(nil))

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

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().NotEmpty(response.Messages()[0].EditedAt)

	// Main instance user attempts to edit the message it received from theirMessenger
	editedMessage = &requests.EditMessage{
		ID:   messageID,
		Text: "edited-again text",
	}
	_, err = s.m.EditMessage(context.Background(), editedMessage)

	s.Require().Equal(ErrInvalidEditAuthor, err)
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
			Clock:     editedMessage.Clock + 2,
			Text:      "some text",
			MessageId: editedMessage.ID,
			ChatId:    chat.ID,
		},
		From: contact.ID,
	}

	err = s.m.HandleEditMessage(response, editMessage)
	// It should error as the user can't edit this message
	s.Require().NoError(err)
	// It save the edit
	s.Require().Len(response.Messages(), 1)

	editedMessage = response.Messages()[0]

	// In-between edit
	editMessage = EditMessage{
		EditMessage: protobuf.EditMessage{
			Clock:     editedMessage.Clock + 1,
			Text:      "some other text",
			MessageId: editedMessage.ID,
			ChatId:    chat.ID,
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
