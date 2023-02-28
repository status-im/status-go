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
	_, err := s.alice1.Start()
	s.Require().NoError(err)
	_, err = s.alice2.Start()
	s.Require().NoError(err)
}

func (s *MessengerDeleteMessageForMeSuite) TearDownTest() {
	s.Require().NoError(s.alice1.Shutdown())
	s.Require().NoError(s.alice2.Shutdown())
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
	messageID := response.Messages()[0].ID
	s.Require().NoError(err)

	var receivedPubChatMessage *common.Message
	err = tt.RetryWithBackOff(func() error {
		var err error
		_, err = s.alice1.RetrieveAll()
		if err != nil {
			return err
		}

		response, err := s.alice2.RetrieveAll()
		if err != nil {
			return err
		}

		messages := response.Messages()
		if len(messages) > 0 {
			receivedPubChatMessage = messages[0]
			return nil
		}

		return errors.New("Not received all messages")
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
	s.Require().Nil(response.Chats()[0].LastMessage)

	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err := s.alice2.RetrieveAll()
		if err != nil {
			return err
		}

		_, err = otherMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.messages) > 0 {
			receivedPubChatMessage = response.Messages()[0]
			return nil
		}

		return errors.New("Not received all messages")
	})
	s.Require().NoError(err)

	deletedForMeMessage, err := s.alice2.MessageByID(messageID)
	s.Require().NoError(err)
	s.Require().True(deletedForMeMessage.DeletedForMe)

	// no DeletedForMe in others' message
	otherMessage, err := otherMessenger.MessageByID(messageID)
	s.Require().NoError(err)
	s.Require().False(otherMessage.DeletedForMe)
}
