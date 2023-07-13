package protocol

import (
	"context"
	"errors"
	"testing"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/requests"

	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"

	"github.com/stretchr/testify/suite"
)

func TestMessengerSyncClearHistorySuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncClearHistory))
}

type MessengerSyncClearHistory struct {
	MessengerBaseTestSuite
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
