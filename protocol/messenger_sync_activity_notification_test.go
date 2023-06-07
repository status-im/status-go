package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
)

func TestMessengerSyncActivityCenterNotificationSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncActivityCenterNotificationSuite))
}

type MessengerSyncActivityCenterNotificationSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	theirMessenger *Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerSyncActivityCenterNotificationSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
	// We start the messenger in order to receive installations
	_, err := s.m.Start()
	s.Require().NoError(err)

	// pair
	s.theirMessenger, err = newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	err = s.theirMessenger.SetInstallationMetadata(s.theirMessenger.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	_, err = s.theirMessenger.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)
	s.Require().NoError(err)
	err = s.m.EnableInstallation(s.theirMessenger.installationID)
	s.Require().NoError(err)
}

func (s *MessengerSyncActivityCenterNotificationSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
	s.Require().NoError(s.theirMessenger.Shutdown())
}

func (s *MessengerSyncActivityCenterNotificationSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)

	return messenger
}

func (s *MessengerSyncActivityCenterNotificationSuite) TestSyncActivityCenterNotification() {
	// add notification
	now := currentMilliseconds()
	notificationID := types.HexBytes("123")
	expectedNotification := &ActivityCenterNotification{
		ID:	  notificationID,
		Author: "author",
		Type: ActivityCenterNotificationTypeContactRequest,
		Timestamp: now,
		UpdatedAt: now,
	}
	err := s.m.addActivityCenterNotification(&MessengerResponse{}, expectedNotification)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		response, err := s.theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.ActivityCenterNotifications()) == 1 {
			require.Equal(s.T(), expectedNotification, response.ActivityCenterNotifications()[0])
			return nil
		}
		return errors.New("not received notifications")
	})
	s.Require().NoError(err)

	notification, err := s.theirMessenger.ActivityCenterNotification(notificationID)
	s.Require().NoError(err)
	s.Require().Equal(expectedNotification, notification)

	// check state
	s1, err := s.m.GetActivityCenterState()
	s.Require().NoError(err)
	s.Require().NotNil(s1)
	s.Require().False(s1.HasSeen)
	s2, err := s.theirMessenger.GetActivityCenterState()
	s.Require().NoError(err)
	s.Require().NotNil(s2)
	s.Require().False(s2.HasSeen)
	s.Require().Equal(s1.UpdatedAt, s2.UpdatedAt)

	// mark as seen
	_, err = s.m.MarkAsSeenActivityCenterNotifications()
	s.Require().NoError(err)
	err = tt.RetryWithBackOff(func() error {
		response, err := s.theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}
		if response.ActivityCenterState() != nil {
			require.True(s.T(), response.ActivityCenterState().HasSeen)
			return nil
		}
		return errors.New("not received notification state")
	})
	s.Require().NoError(err)
	s3, err := s.theirMessenger.GetActivityCenterState()
	s.Require().NoError(err)
	s.Require().NotNil(s3)
	s.Require().True(s3.HasSeen)
}
