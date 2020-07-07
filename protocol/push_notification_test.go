package protocol

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/whisper/v6"
)

func TestMessengerPushNotificationSuite(t *testing.T) {
	suite.Run(t, new(MessengerPushNotificationSuite))
}

type MessengerPushNotificationSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Whisper service should be shared.
	shh      types.Whisper
	tmpFiles []*os.File // files to clean up
	logger   *zap.Logger
}

func (s *MessengerPushNotificationSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := whisper.DefaultConfig
	config.MinimumAcceptedPOW = 0
	shh := whisper.New(&config)
	s.shh = gethbridge.NewGethWhisperWrapper(shh)
	s.Require().NoError(shh.Start(nil))

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
}

func (s *MessengerPushNotificationSuite) newMessengerWithKey(shh types.Whisper, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	options := []Option{
		WithCustomLogger(s.logger),
		WithMessagesPersistenceEnabled(),
		WithDatabaseConfig(tmpFile.Name(), "some-key"),
		WithDatasync(),
	}
	m, err := NewMessenger(
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	s.tmpFiles = append(s.tmpFiles, tmpFile)

	return m
}

func (s *MessengerPushNotificationSuite) newMessenger(shh types.Whisper) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerPushNotificationSuite) TestReceivePushNotification() {
	deviceToken := "token"

	server := s.newMessenger(s.shh)
	client2 := s.newMessenger(s.shh)

	err := s.m.AddPushNotificationServer(context.Background(), &server.identity.PublicKey)
	s.Require().NoError(err)

	err = s.m.RegisterForPushNotifications(context.Background(), deviceToken)
	s.Require().NoError(err)

	info, err := client2.pushNotificationClient.RetrievePushNotificationInfo(&s.m.identity.PublicKey)
	s.Require().NoError(err)
	s.Require().NotNil(info)

	/*
		s.Require().Len(response.Contacts, 1)
		contact := response.Contacts[0]
		s.Require().True(contact.IsAdded())

		s.Require().Len(response.Chats, 1)
		chat := response.Chats[0]
		s.Require().False(chat.Active, "It does not create an active chat")

		// Wait for the message to reach its destination
		response, err = WaitOnMessengerResponse(
			s.m,
			func(r *MessengerResponse) bool { return len(r.Contacts) > 0 },
			"contact request not received",
		)
		s.Require().NoError(err)

		receivedContact := response.Contacts[0]
		s.Require().Equal(theirName, receivedContact.Name)
		s.Require().Equal(theirPicture, receivedContact.Photo)
		s.Require().False(receivedContact.ENSVerified)
		s.Require().True(receivedContact.HasBeenAdded())
		s.Require().NotEmpty(receivedContact.LastUpdated)

		newPicture := "new-picture"
		err = theirMessenger.SendPushNotifications(context.Background(), newName, newPicture)
		s.Require().NoError(err)

		// Wait for the message to reach its destination
		response, err = WaitOnMessengerResponse(
			s.m,
			func(r *MessengerResponse) bool {
				return len(r.Contacts) > 0 && response.Contacts[0].ID == theirContactID
			},
			"contact request not received",
		)

		s.Require().NoError(err)

		receivedContact = response.Contacts[0]
		s.Require().Equal(theirContactID, receivedContact.ID)
		s.Require().Equal(newName, receivedContact.Name)
		s.Require().Equal(newPicture, receivedContact.Photo)
		s.Require().False(receivedContact.ENSVerified)
		s.Require().True(receivedContact.HasBeenAdded())
		s.Require().NotEmpty(receivedContact.LastUpdated)
	*/
}
