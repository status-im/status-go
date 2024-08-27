package protocol

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/multiaccounts/accounts"
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

func (s *MessengerPairingSuite) TestMessengerPairAfterSeedPhrase() {
	// assuming alice2 want to sync with alice1
	// alice1 generated the connection string for bootstraping alice2
	// alice2 failed to connect to alice1 and restored from seed phrase
	// alice2 get the installationID1 from alice1 via parsing the connection string
	alice1 := s.m
	alice2, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, alice2)
	installationID1 := alice1.installationID
	installationID2 := alice2.installationID
	s.Require().NotEqual(installationID1, installationID2)
	_, err = alice2.EnableInstallationAndPair(&requests.EnableInstallationAndPair{InstallationID: installationID1})
	s.Require().NoError(err)

	// alice1 should get the installationID1 from alice2
	_, err = WaitOnMessengerResponse(
		alice2,
		func(r *MessengerResponse) bool {
			for _, i := range r.Installations() {
				if i.ID == installationID2 {
					return true
				}
			}
			return false
		},
		"no messages",
	)

	s.Require().NoError(err)

}

func (s *MessengerPairingSuite) TestMessengerPairEnableAndSync() {
	// assuming alice2 want to sync with alice1
	// alice1 generated the connection string for bootstraping alice2
	// alice2 failed to connect to alice1 and restored from seed phrase
	// alice2 get the installationID1 from alice1 via parsing the connection string
	alice1 := s.m
	alice2logger := s.logger.With(zap.Namespace("alice2"))
	alice2, err := newMessengerWithKey(s.shh, s.privateKey, alice2logger, nil)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, alice2)
	installationID1 := alice1.installationID
	installationID2 := alice2.installationID
	s.Require().NotEqual(installationID1, installationID2)
	_, err = alice2.EnableInstallationAndPair(&requests.EnableInstallationAndPair{InstallationID: installationID1})
	s.Require().NoError(err)

	profileKp := accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = alice2.account.KeyUID
	profileKp.Accounts[0].KeyUID = alice2.account.KeyUID

	err = alice2.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	profileKp = accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = alice1.account.KeyUID
	profileKp.Accounts[0].KeyUID = alice1.account.KeyUID

	err = alice1.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	// alice1 should get the installationID1 from alice2
	_, err = WaitOnMessengerResponse(
		alice1,
		func(r *MessengerResponse) bool {
			for _, i := range r.Installations() {
				if i.ID == installationID2 {
					return true
				}
			}
			return false
		},
		"no messages",
	)

	s.Require().NoError(err)

	displayName := "new-display-name"
	err = alice1.SetDisplayName(displayName)
	s.Require().NoError(err)

	// sync data from acc to acc2
	err = alice1.EnableAndSyncInstallation(&requests.EnableAndSyncInstallation{InstallationID: alice2.InstallationID()})
	s.Require().NoError(err)
	installations := alice1.Installations()
	s.Require().Len(installations, 2)

	if installations[0].ID == alice2.InstallationID() {
		s.Require().True(installations[0].Enabled)
	} else {
		s.Require().Equal(alice2.InstallationID(), installations[1].ID)
		s.Require().True(installations[1].Enabled)
	}
	// ensure acc2's display name get synced
	err = tt.RetryWithBackOff(func() error {
		r, err := alice2.RetrieveAll()
		s.Require().NoError(err)
		for _, ss := range r.Settings {
			if ss.GetDBName() == "display_name" && ss.Value == displayName {
				return nil
			}
		}
		return errors.New("display name setting not received yet")
	})
	s.Require().NoError(err)
}
