package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerMuteSuite(t *testing.T) {
	suite.Run(t, new(MessengerMuteSuite))
}

type MessengerMuteSuite struct {
	MessengerBaseTestSuite
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
