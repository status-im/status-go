package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"testing"
	"time"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/protocol/verification"
	"github.com/status-im/status-go/waku"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
)

func TestMessengerSyncVerificationRequests(t *testing.T) {
	suite.Run(t, new(MessengerSyncVerificationRequests))
}

type MessengerSyncVerificationRequests struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerSyncVerificationRequests) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())
	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	// We start the messenger in order to receive installations
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerSyncVerificationRequests) TestSyncVerificationRequests() {
	request := &verification.Request{
		From:          "0x01",
		To:            "0x02",
		Challenge:     "ABC",
		Response:      "ABC",
		RequestedAt:   uint64(time.Now().Unix()),
		RepliedAt:     uint64(time.Now().Unix()),
		RequestStatus: verification.RequestStatusACCEPTED,
	}
	err := s.m.verificationDatabase.SaveVerificationRequest(request)
	s.Require().NoError(err)

	// pair
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
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

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

	// sync
	err = s.m.SyncVerificationRequest(context.Background(), request, s.m.dispatchMessage)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.VerificationRequests()) == 1 {
			return nil
		}
		return errors.New("Not received all verification requests")
	})

	s.Require().NoError(err)

	time.Sleep(4 * time.Second)

	requests, err := theirMessenger.verificationDatabase.GetVerificationRequests()
	s.Require().NoError(err)
	s.Require().Len(requests, 1)

	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSyncVerificationRequests) TestSyncTrust() {
	err := s.m.verificationDatabase.SetTrustStatus("0x01", verification.TrustStatusTRUSTED, 123)
	s.Require().NoError(err)

	// pair
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
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

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

	// sync
	err = s.m.SyncTrustedUser(context.Background(), "0x01", verification.TrustStatusTRUSTED, s.m.dispatchMessage)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		if response.TrustStatus() != nil {
			return nil
		}

		return errors.New("Not received all user trust levels")
	})

	s.Require().NoError(err)

	trustLevel, err := theirMessenger.verificationDatabase.GetTrustStatus("0x01")
	s.Require().NoError(err)
	s.Require().Equal(verification.TrustStatusTRUSTED, trustLevel)

	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSyncVerificationRequests) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerSyncVerificationRequests) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}
