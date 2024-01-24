package protocol

import (
	"context"
	"testing"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"

	"github.com/stretchr/testify/suite"
)

func TestMessengerSettings(t *testing.T) {
	suite.Run(t, new(MessengerSettingsSuite))
}

type MessengerSettingsSuite struct {
	MessengerBaseTestSuite
	m2 *Messenger
}

func (s *MessengerSettingsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	pk, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.m, err = newMessengerWithKey(s.shh, pk, s.logger, nil)
	s.Require().NoError(err)

	s.m2, err = newMessengerWithKey(s.shh, s.m.identity, s.logger, nil)
	s.Require().NoError(err)

	prepareMessengersForPairing(&s.Suite, s.m, s.m2)
}

func (s *MessengerSettingsSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.m)
	TearDownMessenger(&s.Suite, s.m2)
	_ = s.logger.Sync()
}

func prepareMessengersForPairing(s *suite.Suite, m1, m2 *Messenger) {
	// Set m's installation metadata
	aim := &multidevice.InstallationMetadata{
		Name:       "m's-device",
		DeviceType: "m's-device-type",
	}
	err := m1.SetInstallationMetadata(m1.installationID, aim)
	s.Require().NoError(err)

	// Set m 2's installation metadata
	a2im := &multidevice.InstallationMetadata{
		Name:       "m's-other-device",
		DeviceType: "m's-other-device-type",
	}
	err = m2.SetInstallationMetadata(m2.installationID, a2im)
	s.Require().NoError(err)
}

func (s *MessengerSettingsSuite) TestSetCustomizationColor() {
	PairDevices(&s.Suite, s.m2, s.m)
	PairDevices(&s.Suite, s.m, s.m2)

	s.Require().Equal(s.m.account.KeyUID, s.m2.account.KeyUID)

	err := s.m.multiAccounts.SaveAccount(*s.m.account)
	s.Require().NoError(err)
	err = s.m2.multiAccounts.SaveAccount(*s.m2.account)
	s.Require().NoError(err)

	// check that accounts have no customization color
	acc, err := s.m.multiAccounts.GetAccount(s.m.account.KeyUID)
	s.Require().NoError(err)
	acc2, err := s.m2.multiAccounts.GetAccount(s.m2.account.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(acc.CustomizationColor, common.CustomizationColor(""))
	s.Require().Equal(acc.CustomizationColorClock, uint64(0))
	s.Require().Equal(acc2.CustomizationColor, common.CustomizationColor(""))
	s.Require().Equal(acc2.CustomizationColorClock, uint64(0))

	err = s.m.SetCustomizationColor(context.TODO(), &requests.SetCustomizationColor{KeyUID: s.m.account.KeyUID, CustomizationColor: common.CustomizationColorBlue})
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		return len(r.CustomizationColor) > 0
	}, "message syncAccountCustomizationColor not received")
	s.Require().NoError(err)
	acc, err = s.m2.multiAccounts.GetAccount(s.m.account.KeyUID)
	s.Require().NoError(err)
	acc2, err = s.m2.multiAccounts.GetAccount(s.m2.account.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(common.CustomizationColorBlue, acc.CustomizationColor)
	s.Require().Equal(acc.CustomizationColor, acc2.CustomizationColor)
}
