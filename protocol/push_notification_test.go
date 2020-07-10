package protocol

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/push_notification_client"
	"github.com/status-im/status-go/protocol/push_notification_server"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/whisper/v6"
)

func TestMessengerPushNotificationSuite(t *testing.T) {
	suite.Run(t, new(MessengerPushNotificationSuite))
}

// TODO: to test. Register -> stop server -> re-start -> is it loading the topics?

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

func (s *MessengerPushNotificationSuite) newMessengerWithOptions(shh types.Whisper, privateKey *ecdsa.PrivateKey, options []Option) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

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

func (s *MessengerPushNotificationSuite) newMessengerWithKey(shh types.Whisper, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	options := []Option{
		WithCustomLogger(s.logger),
		WithMessagesPersistenceEnabled(),
		WithDatabaseConfig(tmpFile.Name(), "some-key"),
		WithDatasync(),
	}
	return s.newMessengerWithOptions(shh, privateKey, options)
}

func (s *MessengerPushNotificationSuite) newMessenger(shh types.Whisper) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerPushNotificationSuite) newPushNotificationServer(shh types.Whisper) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	serverConfig := &push_notification_server.Config{
		Logger:   s.logger,
		Identity: privateKey,
	}

	options := []Option{
		WithCustomLogger(s.logger),
		WithMessagesPersistenceEnabled(),
		WithDatabaseConfig(tmpFile.Name(), "some-key"),
		WithPushNotificationServerConfig(serverConfig),
		WithDatasync(),
	}
	return s.newMessengerWithOptions(shh, privateKey, options)
}

func (s *MessengerPushNotificationSuite) TestReceivePushNotification() {
	errChan := make(chan error)

	bob1DeviceToken := "token-1"
	bob2DeviceToken := "token-2"
	var bob1Servers, bob2Servers []*push_notification_client.PushNotificationServer

	bob1 := s.m
	bob2 := s.newMessengerWithKey(s.shh, s.m.identity)
	server := s.newPushNotificationServer(s.shh)
	alice := s.newMessenger(s.shh)
	bobInstallationIDs := []string{bob1.installationID, bob2.installationID}

	// Register bob1
	err := bob1.AddPushNotificationServer(context.Background(), &server.identity.PublicKey)
	s.Require().NoError(err)

	go func() {
		bob1Servers, err = bob1.RegisterForPushNotifications(context.Background(), bob1DeviceToken)
		errChan <- err
	}()

	// Receive message, reply
	// TODO: find a better way to handle this waiting
	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	// Check reply
	// TODO: find a better way to handle this waiting
	time.Sleep(500 * time.Millisecond)
	_, err = bob1.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = bob1.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = bob1.RetrieveAll()
	s.Require().NoError(err)

	// Make sure we receive it
	err = <-errChan
	s.Require().NoError(err)
	s.Require().NotNil(bob1Servers)
	s.Require().Len(bob1Servers, 1)
	s.Require().True(bob1Servers[0].Registered)

	// Register bob2
	err = bob2.AddPushNotificationServer(context.Background(), &server.identity.PublicKey)
	s.Require().NoError(err)

	go func() {
		bob2Servers, err = bob2.RegisterForPushNotifications(context.Background(), bob2DeviceToken)
		errChan <- err
	}()

	// Receive message, reply
	// TODO: find a better way to handle this waiting
	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	// Check reply
	// TODO: find a better way to handle this waiting
	time.Sleep(500 * time.Millisecond)
	_, err = bob2.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = bob2.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = bob2.RetrieveAll()
	s.Require().NoError(err)

	// Make sure we receive it
	err = <-errChan
	s.Require().NoError(err)
	s.Require().NotNil(bob2Servers)
	s.Require().Len(bob2Servers, 1)
	s.Require().True(bob2Servers[0].Registered)

	err = alice.pushNotificationClient.QueryPushNotificationInfo(&bob2.identity.PublicKey)
	s.Require().NoError(err)

	// Receive push notification query
	// TODO: find a better way to handle this waiting
	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = server.RetrieveAll()
	s.Require().NoError(err)

	// Receive push notification query response
	// TODO: find a better way to handle this waiting
	time.Sleep(500 * time.Millisecond)
	_, err = alice.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = alice.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = alice.RetrieveAll()
	s.Require().NoError(err)

	// Here we should poll, as we don't know whether they are already there

	info, err := alice.pushNotificationClient.GetPushNotificationInfo(&bob1.identity.PublicKey, bobInstallationIDs)
	s.Require().NoError(err)
	// Check we have replies for both bob1 and bob2
	s.Require().NotNil(info)
	s.Require().Len(info, 2)

	var bob1Info, bob2Info *push_notification_client.PushNotificationInfo

	if info[0].AccessToken == bob1Servers[0].AccessToken {
		bob1Info = info[0]
		bob2Info = info[1]
	} else {
		bob2Info = info[0]
		bob1Info = info[1]
	}

	s.Require().NotNil(bob1Info)
	s.Require().Equal(bob1.installationID, bob1Info.InstallationID)
	s.Require().Equal(bob1Info.AccessToken, bob1Servers[0].AccessToken, bob1Info.AccessToken)
	s.Require().Equal(&bob1.identity.PublicKey, bob1Info.PublicKey)

	s.Require().NotNil(bob2Info)
	s.Require().Equal(bob2.installationID, bob2Info.InstallationID)
	s.Require().Equal(bob2Servers[0].AccessToken, bob2Info.AccessToken)
	s.Require().Equal(&bob2.identity.PublicKey, bob2Info.PublicKey)

	retrievedNotificationInfo, err := alice.pushNotificationClient.GetPushNotificationInfo(&bob1.identity.PublicKey, bobInstallationIDs)
	alice.logger.Info("BOB KEY", zap.Any("key", bob1.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().NotNil(retrievedNotificationInfo)
	s.Require().Len(retrievedNotificationInfo, 2)
}
