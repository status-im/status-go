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
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

const publicChatName = "status"

func TestMessengerSyncChatRemovedSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncChatRemovedSuite))
}

type MessengerSyncChatRemovedSuite struct {
	suite.Suite
	privateKey *ecdsa.PrivateKey
	alice1     *Messenger
	alice2     *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerSyncChatRemovedSuite) newMessenger() *Messenger {
	if s.privateKey == nil {
		privateKey, err := crypto.GenerateKey()
		s.Require().NoError(err)

		s.privateKey = privateKey
	}

	messenger, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerSyncChatRemovedSuite) SetupTest() {
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

func (s *MessengerSyncChatRemovedSuite) TearDownTest() {
	s.Require().NoError(s.alice1.Shutdown())
	s.Require().NoError(s.alice2.Shutdown())
	_ = s.logger.Sync()
}

func (s *MessengerSyncChatRemovedSuite) Pair() {
	err := s.alice2.SetInstallationMetadata(s.alice2.installationID, &multidevice.InstallationMetadata{
		Name:       "alice2",
		DeviceType: "alice2",
	})
	s.Require().NoError(err)
	response, err := s.alice2.SendPairInstallation(context.Background())
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

func (s *MessengerSyncChatRemovedSuite) TestRemovePubChat() {
	chat := CreatePublicChat(publicChatName, s.alice1.transport)
	err := s.alice1.SaveChat(chat)
	s.Require().NoError(err)

	chat = CreatePublicChat(publicChatName, s.alice2.transport)
	err = s.alice2.SaveChat(chat)
	s.Require().NoError(err)

	s.Pair()

	_, err = s.alice1.deactivateChat(publicChatName, true)
	s.Require().NoError(err)

	var allChats []*Chat
	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err := s.alice2.RetrieveAll()
		if err != nil {
			return err
		}

		allChats = append(allChats, response.Chats()...)
		if len(allChats) >= 1 {
			return nil
		}

		return errors.New("Not received all chats & contacts")
	})
	s.Require().NoError(err)

	var statusChat *Chat
	for _, c := range allChats {
		if c.ID == publicChatName {
			statusChat = c
		}
	}

	s.Require().NotNil(statusChat)
	s.Require().False(statusChat.Active)
}
