package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"testing"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
)

func TestMessengerSyncWalletSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncWalletSuite))
}

type MessengerSyncWalletSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerSyncWalletSuite) SetupTest() {
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

func (s *MessengerSyncWalletSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerSyncWalletSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)

	return messenger
}

func (s *MessengerSyncWalletSuite) TestSyncWallets() {

	mainAccount := &accounts.Account{
		Address: types.Address{0x01},
		Wallet:  true,
		Chat:    true,
	}
	watchOnly1 := &accounts.Account{
		Address: types.Address{0x02},
		Name:    "Alice watch only",
		Color:   "green",
		Type:    accounts.AccountTypeWatch,
	}

	// Create a main account on alice
	s.NoError(s.m.settings.SaveAccounts([]*accounts.Account{mainAccount}))

	// Check account is present in the db
	acc1, err := s.m.settings.GetAccounts()
	s.Require().NoError(err, "alice.settings.GetAccounts")
	s.Len(acc1, 1, "Must have 1 main account")

	// Check account values match the expected values
	s.Require().Equal(mainAccount.Address, acc1[0].Address)
	s.Require().Equal(mainAccount.Name, acc1[0].Name)
	s.Require().Equal(mainAccount.Color, acc1[0].Color)
	s.Require().Equal(mainAccount.Type, acc1[0].Type)

	// Create new device and add main account to
	alicesOtherDevice, err := newMessengerWithKey(s.shh, s.m.identity, s.logger, nil)
	s.Require().NoError(err)

	s.NoError(alicesOtherDevice.settings.SaveAccounts([]*accounts.Account{mainAccount}), true)

	acc2, err := alicesOtherDevice.settings.GetAccounts()
	s.Require().NoError(err, "alicesOtherDevice.settings.GetAccounts")
	s.Len(acc2, 1, "Must have 1 main account")

	// Check account values match the expected values
	s.Require().Equal(mainAccount.Address, acc2[0].Address)
	s.Require().Equal(mainAccount.Name, acc2[0].Name)
	s.Require().Equal(mainAccount.Color, acc2[0].Color)
	s.Require().Equal(mainAccount.Type, acc2[0].Type)

	// Pair devices
	im1 := &multidevice.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, im1)
	s.Require().NoError(err)
	response, err := alicesOtherDevice.SendPairInstallation(context.Background(), nil)
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
	s.Require().Equal(alicesOtherDevice.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("alice's-other-device", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("alice's-other-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(alicesOtherDevice.installationID)
	s.Require().NoError(err)

	// Create a watch-only acount on alice
	s.NoError(s.m.settings.SaveAccounts([]*accounts.Account{watchOnly1}))
	acc1, err = s.m.settings.GetAccounts()
	s.Require().NoError(err, "alice.settings.GetAccounts")
	s.Len(acc1, 2, "Must have 2 accounts")

	// Trigger's a sync between devices
	err = s.m.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	err = tt.RetryWithBackOff(func() error {
		_, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		accs, err := alicesOtherDevice.settings.GetAccounts()
		if err != nil {
			return err
		}

		if len(accs) != 2 {
			return errors.New("no sync wallet account received")
		}
		return nil
	})
	s.Require().NoError(err)

	acc2, err = alicesOtherDevice.settings.GetAccounts()
	s.Require().NoError(err, "alicesOtherDevice.settings.GetAccounts")
	s.Len(acc2, 2, "Must have 2 accounts")

	found := false
	for _, acc := range acc2 {
		if acc.Address == watchOnly1.Address {
			// Check account values match the expected values
			s.Require().Equal(watchOnly1.Address, acc.Address)
			s.Require().Equal(watchOnly1.Name, acc.Name)
			s.Require().Equal(watchOnly1.Color, acc.Color)
			s.Require().Equal(watchOnly1.Type, acc.Type)
			found = true
		}
	}
	s.Require().True(found)

	// Updates alice's watch only account attributes

	watchOnly2 := &accounts.Account{
		Address: types.Address{0x03},
		Name:    "Alice watch only 2",
		Color:   "blue",
		Type:    accounts.AccountTypeWatch,
	}
	s.Require().NoError(s.m.SaveAccounts([]*accounts.Account{watchOnly2}))

	// Sync between devices is triggered automatically
	// via watch account changes subscription
	// Retrieve community link & community
	err = tt.RetryWithBackOff(func() error {
		_, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		accs, err := alicesOtherDevice.settings.GetAccounts()
		if err != nil {
			return err
		}

		if len(accs) != 3 {
			return errors.New("no sync wallet account received")
		}
		return nil
	})
	s.Require().NoError(err)

	acc2, err = alicesOtherDevice.settings.GetAccounts()
	s.Require().NoError(err, "alicesOtherDevice.settings.GetAccounts")
	s.Len(acc2, 3, "Must have 2 accounts")

	found = false
	for _, acc := range acc2 {
		if acc.Address == watchOnly2.Address {
			// Check account values match the expected values
			s.Require().Equal(watchOnly2.Address, acc.Address)
			s.Require().Equal(watchOnly2.Name, acc.Name)
			s.Require().Equal(watchOnly2.Color, acc.Color)
			s.Require().Equal(watchOnly2.Type, acc.Type)
			found = true
		}
	}
	s.Require().True(found)
}
