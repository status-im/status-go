package protocol

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerPairingTest(t *testing.T) {
	suite.Run(t, new(MessengerPairingSuite))
}

type MessengerPairingSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerPairingSuite) TestEnableNonExistingInstallation() {
	installationID := uuid.New().String()
	_, err := s.m.FinishPairingThroughSeedPhraseProcess(&requests.FinishPairingThroughSeedPhraseProcess{InstallationID: installationID})
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
	_, err = alice2.FinishPairingThroughSeedPhraseProcess(&requests.FinishPairingThroughSeedPhraseProcess{InstallationID: installationID1})
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
