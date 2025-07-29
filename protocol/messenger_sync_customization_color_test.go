package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/multiaccounts/common"

	"github.com/status-im/status-go/protocol/encryption/multidevice"
)

func TestMessengerAccountCustomizationColor(t *testing.T) {
	suite.Run(t, new(MessengerSyncAccountCustomizationColorSuite))
}

type MessengerSyncAccountCustomizationColorSuite struct {
	MessengerBaseTestSuite
	alice  *Messenger
	alice2 *Messenger
}

func (s *MessengerSyncAccountCustomizationColorSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()

	s.alice = s.m
	s.alice2 = s.anotherMessenger()

	prepareAliceMessengersForPairing(&s.Suite, s.alice, s.alice2)
}

func (s *MessengerSyncAccountCustomizationColorSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.alice2)
	s.MessengerBaseTestSuite.TearDownTest()
}

func prepareAliceMessengersForPairing(s *suite.Suite, alice1, alice2 *Messenger) {
	// Set Alice's installation metadata
	aim := &multidevice.InstallationMetadata{
		Name:       "alice's-device",
		DeviceType: "alice's-device-type",
	}
	err := alice1.SetInstallationMetadata(alice1.installationID, aim)
	s.Require().NoError(err)

	// Set Alice 2's installation metadata
	a2im := &multidevice.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alice2.SetInstallationMetadata(alice2.installationID, a2im)
	s.Require().NoError(err)
}

func (s *MessengerSyncAccountCustomizationColorSuite) TestSyncCustomizationColor() {
	PairDevices(&s.Suite, s.alice2, s.alice)
	PairDevices(&s.Suite, s.alice, s.alice2)

	s.Require().Equal(s.alice.account.KeyUID, s.alice2.account.KeyUID)

	err := s.alice.multiAccounts.SaveAccount(*s.alice.account)
	s.Require().NoError(err)
	err = s.alice2.multiAccounts.SaveAccount(*s.alice2.account)
	s.Require().NoError(err)

	// check that accounts have no customization color
	acc, err := s.alice.multiAccounts.GetAccount(s.alice.account.KeyUID)
	s.Require().NoError(err)
	acc2, err := s.alice2.multiAccounts.GetAccount(s.alice2.account.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(acc.CustomizationColor, common.CustomizationColor(""))
	s.Require().Equal(acc.CustomizationColorClock, uint64(0))
	s.Require().Equal(acc2.CustomizationColor, common.CustomizationColor(""))
	s.Require().Equal(acc2.CustomizationColorClock, uint64(0))

	acc.CustomizationColor = common.CustomizationColorBlue
	acc.CustomizationColorClock = 1
	err = s.alice.syncAccountCustomizationColor(context.TODO(), acc)
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.alice2, func(r *MessengerResponse) bool {
		return len(r.CustomizationColor) > 0
	}, "message syncAccountCustomizationColor not received")
	s.Require().NoError(err)
	acc2, err = s.alice2.multiAccounts.GetAccount(s.alice2.account.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(acc.CustomizationColor, acc2.CustomizationColor)
}
