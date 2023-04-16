package protocol

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerMuteSuite(t *testing.T) {
	suite.Run(t, new(MessengerMuteSuite))
}

type MessengerMuteSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerMuteSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerMuteSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerMuteSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerMuteSuite) TestSetMute() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	theirMessenger, err := newMessengerWithKey(s.shh, key, s.logger, nil)
	s.Require().NoError(err)

	chatID := publicChatName

	chat := CreatePublicChat(chatID, s.m.transport)

	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	_, err = s.m.Join(chat)
	s.Require().NoError(err)

	err = theirMessenger.SaveChat(chat)
	s.Require().NoError(err)

	_, error := s.m.MuteChat(&requests.MuteChat{ChatID: chatID, MutedType: 5})
	s.NoError(error)

	allChats := s.m.Chats()
	s.Require().Len(allChats, 3)

	var actualChat *Chat

	for idx := range allChats {
		if chat.ID == allChats[idx].ID {
			actualChat = allChats[idx]
		}
	}

	s.Require().NotNil(actualChat)
	s.Require().True(actualChat.Muted)

	s.Require().NoError(s.m.UnmuteChat(chatID))

	allChats = s.m.Chats()

	for idx := range allChats {
		if chat.ID == allChats[idx].ID {
			actualChat = allChats[idx]
		}
	}

	s.Require().False(actualChat.Muted)
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerMuteSuite) TestSetMuteForDuration() {
	key, err := crypto.GenerateKey()
	mockTimeOneMinuteAgo := time.Now().Add(-time.Minute)

	s.Require().NoError(err)

	theirMessenger, err := newMessengerWithKey(s.shh, key, s.logger, nil)
	s.Require().NoError(err)

	chatID := publicChatName

	chat := CreatePublicChat(chatID, s.m.transport)

	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	_, err = s.m.Join(chat)
	s.Require().NoError(err)

	err = theirMessenger.SaveChat(chat)
	s.Require().NoError(err)

	allChats := s.m.Chats()
	s.Require().Len(allChats, 3)

	var actualChat *Chat

	for idx := range allChats {
		if chat.ID == allChats[idx].ID {
			actualChat = allChats[idx]
		}
	}

	var contact *Contact
	if actualChat.OneToOne() {
		contact, _ = s.m.allContacts.Load(chatID)
	}

	_, error := s.m.muteChat(actualChat, contact, mockTimeOneMinuteAgo)
	s.NoError(error)
	// Mock Routine
	for _, chat := range allChats {
		chatMuteTill, chatMuteTillErr := time.Parse(time.RFC3339, chat.MuteTill.Format(time.RFC3339))
		currTime, currTimeErr := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if chatMuteTillErr != nil {
			s.logger.Info("err", zap.Any("Couldn't parse muteTill", err))
			return
		}
		if currTimeErr != nil {
			s.logger.Info("err", zap.Any("Couldn't parse current time", err))
			return
		}

		if currTime.After(chatMuteTill) && !chatMuteTill.Equal(time.Time{}) && chat.Muted {
			_ = s.m.UnmuteChat(chat.ID)
		}
	}

	s.Require().NotNil(actualChat)
	s.Require().False(actualChat.Muted)

	s.Require().NoError(theirMessenger.Shutdown())
}
