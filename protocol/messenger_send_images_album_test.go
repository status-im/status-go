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

func TestMessengerSendImagesAlbumSuite(t *testing.T) {
	suite.Run(t, new(MessengerSendImagesAlbumSuite))
}

type MessengerSendImagesAlbumSuite struct {
	suite.Suite
	m          *Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerSendImagesAlbumSuite) SetupTest() {
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

func (s *MessengerSendImagesAlbumSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerSendImagesAlbumSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerSendImagesAlbumSuite) TestAlbumImageMessagesSend() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	const messageCount = 3
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		image, err := buildImageWithoutAlbumIDMessage(*ourChat)
		s.NoError(err)
		album = append(album, image)
	}

	err = s.m.SaveChat(ourChat)
	s.NoError(err)
	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)

	// Check that album count was the number of the images sent
	imagesCount := uint32(0)
	for _, message := range response.Messages() {
		if message.ContentType == protobuf.ChatMessage_IMAGE {
			imagesCount++
		}
	}
	for _, message := range response.Messages() {
		s.Require().NotNil(message.GetImage())
		s.Require().Equal(message.GetImage().AlbumImagesCount, imagesCount)
	}

	s.Require().Equal(messageCount, len(response.Messages()), "it returns the messages")
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), messageCount)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) == 3 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)
	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image, "Message.ID=%s", message.ID)
		s.Require().Equal(image.AlbumImagesCount, imagesCount)
		s.Require().NotEmpty(image.AlbumId, "Message.ID=%s", message.ID)
	}
}

func (s *MessengerSendImagesAlbumSuite) TestAlbumImageMessagesWithMentionSend() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	const messageCount = 3
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		outgoingMessage, err := buildImageWithoutAlbumIDMessage(*ourChat)
		s.NoError(err)
		outgoingMessage.Mentioned = true
		outgoingMessage.Text = "hey @" + common.PubkeyToHex(&theirMessenger.identity.PublicKey)
		album = append(album, outgoingMessage)
	}

	err = s.m.SaveChat(ourChat)
	s.NoError(err)
	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)
	s.Require().Equal(messageCount, len(response.Messages()), "it returns the messages")
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), messageCount)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)

	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image, "Message.ID=%s", message.ID)
		s.Require().NotEmpty(image.AlbumId, "Message.ID=%s", message.ID)
	}

	s.Require().Equal(uint(1), response.Chats()[0].UnviewedMessagesCount, "Just one unread message")
}

func (s *MessengerSendImagesAlbumSuite) TestAlbumImageEditText() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	const messageCount = 3
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		outgoingMessage, err := buildImageWithoutAlbumIDMessage(*ourChat)
		s.NoError(err)
		outgoingMessage.Text = "You can edit me now"
		album = append(album, outgoingMessage)
	}

	err = s.m.SaveChat(ourChat)
	s.NoError(err)
	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)
	s.Require().Equal(messageCount, len(response.Messages()), "it returns the messages")
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), messageCount)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)

	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image, "Message.ID=%s", message.ID)
		s.Require().NotEmpty(image.AlbumId, "Message.ID=%s", message.ID)
	}

	firstMessageID, err := types.DecodeHex(album[0].ID)
	s.Require().NoError(err)

	editedText := "edited"
	editedMessage := &requests.EditMessage{
		ID:   firstMessageID,
		Text: editedText,
	}

	sendResponse, err := s.m.EditMessage(context.Background(), editedMessage)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), messageCount)

	for _, message := range sendResponse.Messages() {
		s.Require().NotEmpty(message.EditedAt)
		s.Require().Equal(message.Text, editedText)
	}

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)

	for _, message := range response.Messages() {
		s.Require().NotEmpty(message.EditedAt)
		s.Require().Equal(message.Text, editedText)
	}
}
