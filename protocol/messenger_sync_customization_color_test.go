package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/multiaccounts/common"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerAccountCustomizationColor(t *testing.T) {
	suite.Run(t, new(MessengerSyncAccountCustomizationColorSuite))
}

type MessengerSyncAccountCustomizationColorSuite struct {
	suite.Suite
	alice  *Messenger
	alice2 *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerSyncAccountCustomizationColorSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	pk, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.alice, err = newMessengerWithKey(s.shh, pk, s.logger, nil)
	s.Require().NoError(err)

	s.alice2, err = newMessengerWithKey(s.shh, s.alice.identity, s.logger, nil)
	s.Require().NoError(err)

	prepareAliceMessengersForPairing(&s.Suite, s.alice, s.alice2)
}

func (s *MessengerSyncAccountCustomizationColorSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.alice)
	TearDownMessenger(&s.Suite, s.alice2)
	_ = s.logger.Sync()
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
