package protocol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/protocol/verification"

	"github.com/stretchr/testify/suite"
)

func TestMessengerSyncVerificationRequests(t *testing.T) {
	suite.Run(t, new(MessengerSyncVerificationRequests))
}

type MessengerSyncVerificationRequests struct {
	MessengerBaseTestSuite
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
