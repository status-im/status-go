package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerDeleteMessageForMeSuite(t *testing.T) {
	suite.Run(t, new(MessengerDeleteMessageForMeSuite))
}

type MessengerDeleteMessageForMeSuite struct {
	suite.Suite
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	alice1     *Messenger
	alice2     *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerDeleteMessageForMeSuite) newMessenger() *Messenger {
	if s.privateKey == nil {
		privateKey, err := crypto.GenerateKey()
		s.Require().NoError(err)

		s.privateKey = privateKey
	}

	messenger, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerDeleteMessageForMeSuite) otherNewMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerDeleteMessageForMeSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.alice1 = s.newMessenger()
	s.alice2 = s.newMessenger()
}

func (s *MessengerDeleteMessageForMeSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.alice1)
	TearDownMessenger(&s.Suite, s.alice2)
	_ = s.logger.Sync()
}

func (s *MessengerDeleteMessageForMeSuite) Pair() {
	err := s.alice2.SetInstallationMetadata(s.alice2.installationID, &multidevice.InstallationMetadata{
		Name:       "alice2",
		DeviceType: "alice2",
	})
	s.Require().NoError(err)
	response, err := s.alice2.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.alice1,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(s.alice2.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("alice2", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("alice2", actualInstallation.InstallationMetadata.DeviceType)

	err = s.alice1.EnableInstallation(s.alice2.installationID)
	s.Require().NoError(err)
}

func (s *MessengerDeleteMessageForMeSuite) TestDeleteMessageForMe() {
	s.Pair()
	chatID := "foobarsynctest"
	_, err := s.alice1.createPublicChat(chatID, &MessengerResponse{})
	s.Require().NoError(err)

	_, err = s.alice2.createPublicChat(chatID, &MessengerResponse{})
	s.Require().NoError(err)

	otherMessenger := s.otherNewMessenger()
	_, err = otherMessenger.createPublicChat(chatID, &MessengerResponse{})
	s.Require().NoError(err)

	chat := otherMessenger.Chat(chatID)
	message := buildTestMessage(*chat)

	response, err := otherMessenger.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)
	messageID := response.Messages()[0].ID

	var receivedPubChatMessage *common.Message
	var alice1ReceivedMessage, alice2ReceivedMessage bool
	var notReceivedMessageError = errors.New("not received all messages")
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice1.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Messages()) > 0 {
			alice1ReceivedMessage = true
		}

		response, err = s.alice2.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Messages()) > 0 {
			alice2ReceivedMessage = true
		}

		messages := response.Messages()
		if len(messages) > 0 {
			receivedPubChatMessage = messages[0]
			if alice1ReceivedMessage && alice2ReceivedMessage {
				return nil
			}
		}

		return notReceivedMessageError
	})
	s.Require().NoError(err)
	s.Require().Equal(receivedPubChatMessage.ChatId, chatID)
	s.Require().Equal(receivedPubChatMessage.ID, messageID)
	s.Require().False(receivedPubChatMessage.DeletedForMe)

	// message synced to alice1
	alice1Msg, err := s.alice1.MessageByID(messageID)
	s.Require().NoError(err)
	s.Require().False(alice1Msg.DeletedForMe)

	response, err = s.alice1.DeleteMessageForMeAndSync(context.Background(), chatID, messageID)
	s.Require().NoError(err)
	s.Require().True(response.Messages()[0].DeletedForMe)
	s.Require().Equal(response.Chats()[0].LastMessage.ID, messageID)
	s.Require().Equal(response.Chats()[0].LastMessage.DeletedForMe, true)

	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice2.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.messages) > 0 {
			return nil
		}

		return notReceivedMessageError
	})
	s.Require().NoError(err)

	deletedForMeMessage, err := s.alice2.MessageByID(messageID)
	s.Require().NoError(err)
	s.Require().True(deletedForMeMessage.DeletedForMe)

	// no DeletedForMe in others' message
	err = tt.RetryWithBackOff(func() error {
		response, err = otherMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.messages) > 0 {
			return nil
		}

		return notReceivedMessageError
	})
	s.Require().ErrorIs(err, notReceivedMessageError)
	otherMessage, err := otherMessenger.MessageByID(messageID)
	s.Require().NoError(err)
	s.Require().False(otherMessage.DeletedForMe)
}

func (s *MessengerDeleteMessageForMeSuite) TestDeleteImageMessageFromReceiverSide() {

	alice := s.otherNewMessenger()
	defer TearDownMessenger(&s.Suite, alice)

	bob := s.otherNewMessenger()
	defer TearDownMessenger(&s.Suite, bob)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, alice.transport)
	err := alice.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &alice.identity.PublicKey, alice.transport)
	err = bob.SaveChat(ourChat)
	s.Require().NoError(err)

	messageCount := 3
	var album []*common.Message
	for i := 0; i < messageCount; i++ {
		image, err := buildImageWithoutAlbumIDMessage(*ourChat)
		s.NoError(err)
		album = append(album, image)
	}

	response, err := bob.SendChatMessages(context.Background(), album)
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
		alice,
		func(r *MessengerResponse) bool { return len(r.messages) == messageCount },
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

	messages := response.Messages()
	firstMessageID := messages[0].ID
	localChatID := messages[0].LocalChatID
	sendResponse, err := alice.DeleteMessageForMeAndSync(context.Background(), localChatID, firstMessageID)
	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 3)
	s.Require().Len(sendResponse.Chats(), 1)

	// LastMessage marked as deleted
	s.Require().Equal(sendResponse.Chats()[0].LastMessage.ID, album[2].ID)
	s.Require().Equal(sendResponse.Chats()[0].LastMessage.DeletedForMe, true)
}
