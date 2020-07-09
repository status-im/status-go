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
	var bob1AccessTokens, bob2AccessTokens []string

	bob1 := s.m
	bob2 := s.newMessengerWithKey(s.shh, s.m.identity)
	server := s.newPushNotificationServer(s.shh)
	client2 := s.newMessenger(s.shh)

	// Register bob1
	err := bob1.AddPushNotificationServer(context.Background(), &server.identity.PublicKey)
	s.Require().NoError(err)

	go func() {
		bob1AccessTokens, err = bob1.RegisterForPushNotifications(context.Background(), bob1DeviceToken)
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
	s.Require().NotNil(bob1AccessTokens)

	// Register bob2
	err = bob2.AddPushNotificationServer(context.Background(), &server.identity.PublicKey)
	s.Require().NoError(err)

	go func() {
		bob2AccessTokens, err = bob2.RegisterForPushNotifications(context.Background(), bob2DeviceToken)
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
	s.Require().NotNil(bob2AccessTokens)

	var info []*push_notification_client.PushNotificationInfo
	go func() {
		info, err = client2.pushNotificationClient.RetrievePushNotificationInfo(&bob2.identity.PublicKey)
		errChan <- err
	}()

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
	_, err = client2.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = client2.RetrieveAll()
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
	_, err = client2.RetrieveAll()
	s.Require().NoError(err)

	err = <-errChan
	s.Require().NoError(err)
	s.Require().NotNil(info)
	// Check we have replies for both bob1 and bob2
	s.Require().Len(info, 2)

	var bob1Info, bob2Info *push_notification_client.PushNotificationInfo

	if info[0].AccessToken == bob1AccessTokens[0] {
		bob1Info = info[0]
		bob2Info = info[1]
	} else {
		bob2Info = info[0]
		bob1Info = info[1]
	}

	s.Require().NotNil(bob1Info)
	s.Require().Equal(bob1Info, &push_notification_client.PushNotificationInfo{
		InstallationID: bob1.installationID,
		AccessToken:    bob1DeviceToken,
		PublicKey:      &bob1.identity.PublicKey,
	})

	s.Require().NotNil(bob2Info)
	s.Require().Equal(bob2Info, &push_notification_client.PushNotificationInfo{
		InstallationID: bob2.installationID,
		AccessToken:    bob2DeviceToken,
		PublicKey:      &bob1.identity.PublicKey,
	})

}
