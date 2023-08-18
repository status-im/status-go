package protocol

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"

	// "github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerShareMessageSuite(t *testing.T) {
	suite.Run(t, new(MessengerShareMessageSuite))
}

type MessengerShareMessageSuite struct {
	suite.Suite
	m          *Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerShareMessageSuite) SetupTest() {
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

func (s *MessengerShareMessageSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerShareMessageSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func buildImageMessage(s *MessengerShareMessageSuite, chat Chat) *common.Message {
	file, err := os.Open("../_assets/tests/test.jpg")
	s.Require().NoError(err)
	defer file.Close()

	payload, err := ioutil.ReadAll(file)
	s.Require().NoError(err)

	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := common.NewMessage()
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_IMAGE
	message.Text = "An image"

	image := protobuf.ImageMessage{
		Payload: payload,
		Type:    protobuf.ImageType_JPEG,
		AlbumId: "some-album-id",
		Width:   1200,
		Height:  1000,
	}
	message.Payload = &protobuf.ChatMessage_Image{Image: &image}
	return message
}

func (s *MessengerShareMessageSuite) TestImageMessageSharing() {
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

	inputMessage := buildImageMessage(s, *ourChat)
	err = s.m.SaveChat(ourChat)
	s.NoError(err)
	response, err := s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Equal(1, len(response.Messages()), "it returns the message")

	outputMessage := response.Messages()[0]

	MessageID := outputMessage.ID

	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(response.Messages()[0].Text, "An image")

	shareResponse, err := s.m.ShareImageMessage(
		&requests.ShareImageMessage{
			MessageID: MessageID,
			Users:     []types.HexBytes{common.PubkeyToHexBytes(&theirMessenger.identity.PublicKey)},
		},
	)

	s.NoError(err)
	s.Require().NotNil(shareResponse)
	s.Require().Len(shareResponse.Messages(), 1)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(response.Messages()[0].Text, "This message has been shared with you")
}
