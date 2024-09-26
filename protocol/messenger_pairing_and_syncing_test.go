package protocol

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

func TestMessengerPairingTest(t *testing.T) {
	suite.Run(t, new(MessengerPairingSuite))
}

type MessengerPairingSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerPairingSuite) TestEnableNonExistingInstallation() {
	installationID := uuid.New().String()
	_, err := s.m.EnableInstallationAndPair(&requests.EnableInstallationAndPair{InstallationID: installationID})
	s.Require().NoError(err)

	installations := s.m.Installations()
	s.Require().NoError(err)

	s.Require().Len(installations, 2)
	var theirInstallation *multidevice.Installation
	for _, i := range installations {
		if i.ID == installationID {
			theirInstallation = i
			break
		} else {
			s.Require().NotNil(i.InstallationMetadata)
		}
	}
	s.Require().NotNil(theirInstallation)
	s.Require().True(theirInstallation.Enabled)

	installationsFromDB, err := s.m.encryptor.GetOurActiveInstallations(&s.m.identity.PublicKey)
	s.Require().NoError(err)
	s.Require().Len(installationsFromDB, 2)
	for _, i := range installationsFromDB {
		s.Require().True(i.Enabled)
		if i.ID == installationID {
			theirInstallation = i
			break
		}
	}
	s.Require().NotNil(theirInstallation)
	s.Require().True(theirInstallation.Enabled)

}

func (s *MessengerPairingSuite) TestWrongTargetInstallationID() {
	alice1 := s.m
	alice2, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, alice2)

	wrongTargetInstallationID := uuid.New().String()
	mockRequest := requests.NewMockEnableInstallationAndPair(alice1.installationID, func() string {
		return wrongTargetInstallationID
	})
	_, err = alice2.EnableInstallationAndPair(mockRequest)
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(
		alice1,
		func(r *MessengerResponse) bool {
			for _, i := range r.Installations() {
				// We expect the installation to be added but no activity center notification
				if i.ID == alice2.installationID && len(r.ActivityCenterNotifications()) == 0 {
					return true
				}
			}
			return false
		},
		"no messages",
	)
	s.Require().NoError(err)
}

// TestMessengerSyncFallback tests the scenario where alice2 wants to sync with alice1
// alice1 generated the connection string for bootstraping alice2
// alice2 failed to connect to alice1 and restored from seed phrase
// alice2 get the installationID1 from alice1 via parsing the connection string
// alice2 should get the display name from alice1 after pairing
func (s *MessengerPairingSuite) TestMessengerSyncFallback() {
	alice1 := s.m
	alice2, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, alice2)

	alice1ProfileKp := accounts.GetProfileKeypairForTest(true, false, false)
	alice1ProfileKp.KeyUID = alice1.account.KeyUID
	alice1ProfileKp.Accounts[0].KeyUID = alice1.account.KeyUID
	err = alice1.settings.SaveOrUpdateKeypair(alice1ProfileKp)
	s.Require().NoError(err)

	expectedDisplayName := "alice1"
	s.Require().NoError(alice1.SetDisplayName(expectedDisplayName))
	ss, err := alice2.getSettings()
	s.Require().NoError(err)
	s.Require().NotEqual(expectedDisplayName, ss.DisplayName)
	installationID1 := alice1.installationID
	installationID2 := alice2.installationID
	s.Require().NotEqual(installationID1, installationID2)

	_, err = alice2.EnableInstallationAndPair(&requests.EnableInstallationAndPair{InstallationID: installationID1})
	s.Require().NoError(err)

	// alice1 should get the installationID1 from alice2
	_, err = WaitOnMessengerResponse(
		alice1,
		func(r *MessengerResponse) bool {
			for _, i := range r.Installations() {
				if i.ID == installationID2 && len(r.ActivityCenterNotifications()) == 1 && r.ActivityCenterNotifications()[0].Type == ActivityCenterNotificationTypeNewInstallationReceived {
					return true
				}
			}
			return false
		},
		"no messages",
	)
	s.Require().NoError(err)

	// check response from alice1
	resp, err := alice1.EnableInstallationAndSync(&requests.EnableInstallationAndSync{InstallationID: installationID2})
	s.Require().NoError(err)
	installationID2Exist := false
	for _, i := range resp.Installations() {
		if i.ID == installationID2 {
			installationID2Exist = true
			break
		}
	}
	s.Require().True(installationID2Exist)

	// check if the display name is synced
	err = tt.RetryWithBackOff(func() error {
		r, err := alice2.RetrieveAll()
		s.Require().NoError(err)
		for _, ss := range r.Settings {
			if ss.GetDBName() == settings.DisplayName.GetDBName() {
				return nil
			}
		}
		return errors.New("display name setting not received yet")
	})
	s.Require().NoError(err)
	ss, err = alice2.getSettings()
	s.Require().NoError(err)
	s.Require().Equal(expectedDisplayName, ss.DisplayName)
}
