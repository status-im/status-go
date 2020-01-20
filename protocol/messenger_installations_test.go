package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/whisper/v6"
)

func TestMessengerInstallationSuite(t *testing.T) {
	suite.Run(t, new(MessengerInstallationSuite))
}

type MessengerInstallationSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Whisper service should be shared.
	shh            types.Whisper
	tmpFiles       []*os.File // files to clean up
	logger         *zap.Logger
	installationID string
}

func (s *MessengerInstallationSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := whisper.DefaultConfig
	config.MinimumAcceptedPOW = 0
	shh := whisper.New(&config)
	s.shh = gethbridge.NewGethWhisperWrapper(shh)
	s.Require().NoError(shh.Start(nil))

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
}

func (s *MessengerInstallationSuite) newMessengerWithKey(shh types.Whisper, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	options := []Option{
		WithCustomLogger(s.logger),
		WithMessagesPersistenceEnabled(),
		WithDatabaseConfig(tmpFile.Name(), "some-key"),
		WithDatasync(),
	}
	installationID := uuid.New().String()
	m, err := NewMessenger(
		privateKey,
		&testNode{shh: shh},
		installationID,
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	s.tmpFiles = append(s.tmpFiles, tmpFile)

	return m
}

func (s *MessengerInstallationSuite) newMessenger(shh types.Whisper) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerInstallationSuite) TestReceiveInstallation() {
	theirMessenger := s.newMessengerWithKey(s.shh, s.privateKey)

	err := theirMessenger.SetInstallationMetadata(theirMessenger.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err := theirMessenger.SendPairInstallation(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().False(response.Chats[0].Active)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Installations) == 0 {
			err = errors.New("installation not received")
		}
		return err
	})
	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(theirMessenger.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("their-name", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("their-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(theirMessenger.installationID)
	s.Require().NoError(err)

	contactKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	contact, err := buildContact(&contactKey.PublicKey)
	s.Require().NoError(err)
	contact.SystemTags = append(contact.SystemTags, contactAdded)
	err = s.m.SaveContact(contact)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = theirMessenger.RetrieveAll()
		if err == nil && len(response.Contacts) == 0 {
			err = errors.New("contact not received")
		}
		if len(response.Contacts) != 0 && response.Contacts[0].ID != contact.ID {
			err = errors.New("contact not received")
		}
		return err
	})
	s.Require().NoError(err)

	actualContact := response.Contacts[0]
	s.Require().Equal(contact.ID, actualContact.ID)
	s.Require().True(actualContact.IsAdded())

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = theirMessenger.RetrieveAll()
		if err == nil && len(response.Chats) == 0 {
			err = errors.New("sync chat not received")
		}
		return err
	})
	s.Require().NoError(err)

	actualChat := response.Chats[0]
	s.Require().Equal("status", actualChat.ID)
	s.Require().True(actualChat.Active)
}

func (s *MessengerInstallationSuite) TestSyncInstallation() {

	// add contact
	contactKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	contact, err := buildContact(&contactKey.PublicKey)
	s.Require().NoError(err)
	contact.SystemTags = append(contact.SystemTags, contactAdded)
	err = s.m.SaveContact(contact)
	s.Require().NoError(err)

	// add chat
	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	// pair
	theirMessenger := s.newMessengerWithKey(s.shh, s.privateKey)

	err = theirMessenger.SetInstallationMetadata(theirMessenger.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err := theirMessenger.SendPairInstallation(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().False(response.Chats[0].Active)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Installations) == 0 {
			err = errors.New("installation not received")
		}
		return err
	})
	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(theirMessenger.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("their-name", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("their-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(theirMessenger.installationID)
	s.Require().NoError(err)

	// sync
	err = s.m.SyncDevices(context.Background(), "ens-name", "profile-image")
	s.Require().NoError(err)

	var allChats []*Chat
	var allContacts []*Contact
	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		allChats = append(allChats, response.Chats...)
		allContacts = append(allContacts, response.Contacts...)

		if len(allChats) >= 2 && len(allContacts) >= 3 {
			return nil
		}

		return errors.New("Not received all chats & contacts")

	})

	s.Require().NoError(err)

	var statusChat *Chat
	for _, c := range allChats {
		if c.ID == "status" {
			statusChat = c
		}
	}

	s.Require().NotNil(statusChat)

	var actualContact *Contact
	for _, c := range allContacts {
		if c.ID == contact.ID {
			actualContact = c
		}
	}

	s.Require().True(actualContact.IsAdded())

	var ourContact *Contact

	myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	for _, c := range allContacts {
		if c.ID == myID {
			if ourContact == nil || ourContact.LastUpdated < c.LastUpdated {
				ourContact = c
			}
		}
	}

	s.Require().NotNil(ourContact)
	s.Require().Equal("ens-name", ourContact.Name)
	s.Require().Equal("profile-image", ourContact.Photo)

}
