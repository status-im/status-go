package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"testing"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/requests"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
)

func TestMessengerSyncClearHistorySuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncClearHistory))
}

type MessengerSyncClearHistory struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerSyncClearHistory) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	// We start the messenger in order to receive installations
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerSyncClearHistory) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerSyncClearHistory) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)

	return messenger
}

func (s *MessengerSyncClearHistory) pair() *Messenger {
	theirMessenger, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)

	err = theirMessenger.SetInstallationMetadata(theirMessenger.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err := theirMessenger.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(theirMessenger.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("their-name", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("their-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(theirMessenger.installationID)
	s.Require().NoError(err)

	return theirMessenger
}

func (s *MessengerSyncClearHistory) TestSyncClearHistory() {
	theirMessenger := s.pair()

	response, err := s.m.CreatePublicChat(&requests.CreatePublicChat{
		ID: publicChatName,
	})
	s.Require().NoError(err)
	chat, ok := response.chats[publicChatName]
	s.Require().True(ok)

	_, err = theirMessenger.CreatePublicChat(&requests.CreatePublicChat{
		ID: publicChatName,
	})
	s.Require().NoError(err)

	message := buildTestMessage(*chat)
	_, err = s.m.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)
	var receivedPubChatMessage *common.Message
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err := theirMessenger.RetrieveAll()
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
	s.Require().Equal(receivedPubChatMessage.ChatId, publicChatName)

	var messages []*common.Message
	messages, _, err = s.m.persistence.MessageByChatID(publicChatName, "", 10)
	s.Require().NoError(err)
	s.Require().True(len(messages) == 1)
	messages, _, err = theirMessenger.persistence.MessageByChatID(publicChatName, "", 10)
	s.Require().NoError(err)
	s.Require().True(len(messages) == 1)

	_, err = s.m.ClearHistory(&requests.ClearHistory{
		ID: publicChatName,
	})
	s.Require().NoError(err)
	err = tt.RetryWithBackOff(func() error {
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.clearedHistories) > 0 {
			return nil
		}
		return errors.New("Not received all clearedHistories")
	})
	s.Require().NoError(err)
	messages, _, err = theirMessenger.persistence.MessageByChatID(publicChatName, "", 10)
	s.Require().NoError(err)
	s.Require().True(len(messages) == 0)

	s.Require().NoError(theirMessenger.Shutdown())

}
