package protocol

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
)

const publicChatName = "status"

func TestMessengerSyncChatSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncChatSuite))
}

type MessengerSyncChatSuite struct {
	MessengerBaseTestSuite
	alice1 *Messenger
	alice2 *Messenger
}

func (s *MessengerSyncChatSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()

	s.alice1 = s.m
	s.alice2 = s.anotherMessenger()
}

func (s *MessengerSyncChatSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.alice2)
	s.MessengerBaseTestSuite.TearDownTest()
}

func (s *MessengerSyncChatSuite) Pair() {
	err := s.alice2.SetInstallationMetadata(s.alice2.installationID, &multidevice.InstallationMetadata{
		Name:       "alice2",
		DeviceType: "alice2",
	})
	s.Require().NoError(err)
	response, err := s.alice2.SendPairInstallation(context.Background(), "", nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.alice1,
		func(r *MessengerResponse) bool { return len(r.Installations()) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations()[0]
	s.Require().Equal(s.alice2.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("alice2", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("alice2", actualInstallation.InstallationMetadata.DeviceType)

	_, err = s.alice1.EnableInstallation(s.alice2.installationID)
	s.Require().NoError(err)
}

func (s *MessengerSyncChatSuite) TestRemovePubChat() {
	chat := CreatePublicChat(publicChatName, s.alice1.getTimesource())
	err := s.alice1.SaveChat(chat)
	s.Require().NoError(err)

	chat = CreatePublicChat(publicChatName, s.alice2.getTimesource())
	err = s.alice2.SaveChat(chat)
	s.Require().NoError(err)

	s.Pair()

	_, err = s.alice1.deactivateChat(publicChatName, 0, true, true)
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

func (s *MessengerSyncChatSuite) TestMarkChatMessagesRead() {
	s.Pair()
	chatID := "foobarsynctest"
	_, err := s.alice1.createPublicChat(chatID, &MessengerResponse{})
	s.Require().NoError(err)

	_, err = s.alice2.createPublicChat(chatID, &MessengerResponse{})
	s.Require().NoError(err)

	otherMessenger := s.newMessenger()
	defer TearDownMessenger(&s.Suite, otherMessenger)

	_, err = otherMessenger.createPublicChat(chatID, &MessengerResponse{})
	s.Require().NoError(err)

	chat := otherMessenger.Chat(chatID)
	message := buildTestMessage(*chat)

	_, err = otherMessenger.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)

	var receivedPubChatMessage *common.Message
	err = tt.RetryWithBackOff(func() error {
		var err error
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

	alice2chat := s.alice2.Chat(chatID)
	s.Require().Equal(alice2chat.UnviewedMessagesCount, uint(1))

	_, err = s.alice1.MarkAllRead(context.TODO(), chatID)
	s.Require().NoError(err)

	var receivedChat *Chat
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err := s.alice2.RetrieveAll()
		if err != nil {
			return err
		}

		chats := response.Chats()
		if len(chats) > 0 {
			receivedChat = chats[0]
			return nil
		}

		return errors.New("Not received all messages")
	})
	s.Require().NoError(err)

	s.Require().Equal(receivedChat.ID, chatID)
	s.Require().Equal(receivedChat.UnviewedMessagesCount, uint(0))
}
