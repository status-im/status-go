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

func getWalletAccountsForTest() []*accounts.Account {
	defaultAccount := &accounts.Account{
		Address:                 types.Address{0x11},
		KeyUID:                  "0000000000000000000000000000000000000000000000000000000000000001",
		Wallet:                  true,
		Chat:                    false,
		Path:                    "m/44'/60'/0'/0/0",
		Name:                    "Default Account",
		Color:                   "blue",
		DerivedFrom:             "0x0001",
		KeypairName:             "Profile Keypair",
		LastUsedDerivationIndex: 0,
	}
	generatedFromDefaultAccount1 := &accounts.Account{
		Address:                 types.Address{0x12},
		Type:                    accounts.AccountTypeGenerated,
		KeyUID:                  defaultAccount.KeyUID,
		Path:                    "m/44'/60'/0'/0/1",
		Name:                    "Generated Acc 1",
		Color:                   "blue",
		DerivedFrom:             defaultAccount.DerivedFrom,
		KeypairName:             defaultAccount.KeypairName,
		LastUsedDerivationIndex: 1,
	}
	generatedFromDefaultAccount2 := &accounts.Account{
		Address:                 types.Address{0x13},
		Type:                    accounts.AccountTypeGenerated,
		KeyUID:                  defaultAccount.KeyUID,
		Path:                    "m/44'/60'/0'/0/2",
		Name:                    "Generated Acc 2",
		Color:                   "blue",
		DerivedFrom:             defaultAccount.DerivedFrom,
		KeypairName:             defaultAccount.KeypairName,
		LastUsedDerivationIndex: 2,
	}
	seedImportedAccount := &accounts.Account{
		Address:                 types.Address{0x14},
		Type:                    accounts.AccountTypeSeed,
		KeyUID:                  "0000000000000000000000000000000000000000000000000000000000000002",
		Path:                    "m/44'/60'/0'/0/0",
		Name:                    "Seed Imported Account",
		Color:                   "green",
		DerivedFrom:             "0x0002",
		KeypairName:             "Seed Keypair",
		LastUsedDerivationIndex: 0,
	}
	generatedFromSeedImportedAccount1 := &accounts.Account{
		Address:                 types.Address{0x15},
		Type:                    accounts.AccountTypeSeed,
		KeyUID:                  seedImportedAccount.KeyUID,
		Path:                    "m/44'/60'/0'/0/1",
		Name:                    "Generated Seed Account 1",
		Color:                   "green",
		DerivedFrom:             seedImportedAccount.DerivedFrom,
		KeypairName:             seedImportedAccount.KeypairName,
		LastUsedDerivationIndex: 1,
	}
	generatedFromSeedImportedAccount2 := &accounts.Account{
		Address:                 types.Address{0x16},
		Type:                    accounts.AccountTypeSeed,
		KeyUID:                  seedImportedAccount.KeyUID,
		Path:                    "m/44'/60'/0'/0/2",
		Name:                    "Generated Seed Account 2",
		Color:                   "green",
		DerivedFrom:             seedImportedAccount.DerivedFrom,
		KeypairName:             seedImportedAccount.KeypairName,
		LastUsedDerivationIndex: 2,
	}
	keyImportedAccount := &accounts.Account{
		Address:     types.Address{0x17},
		Type:        accounts.AccountTypeKey,
		KeyUID:      "0000000000000000000000000000000000000000000000000000000000000003",
		Path:        "m",
		Name:        "Key Imported Account",
		Color:       "blue",
		KeypairName: "Private Key Keypair",
	}
	watchOnlyAccount1 := &accounts.Account{
		Address: types.Address{0x18},
		Type:    accounts.AccountTypeWatch,
		Name:    "Watch Only Account 1",
		Color:   "green",
	}
	watchOnlyAccount2 := &accounts.Account{
		Address: types.Address{0x19},
		Type:    accounts.AccountTypeWatch,
		Name:    "Watch Only Account 1",
		Color:   "green",
	}

	return []*accounts.Account{
		defaultAccount,
		generatedFromDefaultAccount1,
		generatedFromDefaultAccount2,
		seedImportedAccount,
		generatedFromSeedImportedAccount1,
		generatedFromSeedImportedAccount2,
		keyImportedAccount,
		watchOnlyAccount1,
		watchOnlyAccount2,
	}
}

func (s *MessengerSyncWalletSuite) TestSyncWallets() {
	mainAccount := &accounts.Account{
		Address: types.Address{0x01},
		Wallet:  false,
		Chat:    true,
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

	// Store wallet accounts on alice's device
	walletAccounts := getWalletAccountsForTest()
	expectedTotalNumOfAccounts := len(walletAccounts) + 1 // plus one for the Status profile account
	s.NoError(s.m.settings.SaveAccounts(walletAccounts))
	acc1, err = s.m.settings.GetAccounts()
	s.Require().NoError(err, "alice.settings.GetAccounts")
	s.Len(acc1, expectedTotalNumOfAccounts, "Must have all wallet accounts plus one for the Status profile account")

	// Trigger's a sync between devices
	err = s.m.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	err = tt.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.Accounts) != len(walletAccounts) {
			return errors.New("no sync wallet account received")
		}
		return nil
	})
	s.Require().NoError(err)

	acc2, err = alicesOtherDevice.settings.GetAccounts()
	s.Require().NoError(err, "alicesOtherDevice.settings.GetAccounts")
	s.Len(acc2, expectedTotalNumOfAccounts, "Must have all wallet accounts plus one for the Status profile account")

	for _, syncedAcc := range acc2 {
		if syncedAcc.Chat {
			continue
		}
		found := false
		for _, sentAcc := range walletAccounts {
			if syncedAcc.Address == sentAcc.Address {
				// Check account values match the expected values
				s.Require().Equal(sentAcc.Address, syncedAcc.Address)
				s.Require().Equal(sentAcc.Path, syncedAcc.Path)
				s.Require().Equal(sentAcc.KeyUID, syncedAcc.KeyUID)
				s.Require().Equal(sentAcc.Name, syncedAcc.Name)
				s.Require().Equal(sentAcc.Color, syncedAcc.Color)
				s.Require().Equal(sentAcc.Type, syncedAcc.Type)
				s.Require().Equal(sentAcc.KeypairName, syncedAcc.KeypairName)
				s.Require().Equal(sentAcc.DerivedFrom, syncedAcc.DerivedFrom)
				found = true
			}
		}
		s.Require().True(found)
	}

	// Updates alice's accounts attributes
	for _, acc := range walletAccounts {
		acc.Name = acc.Name + "New"
		acc.Color = "lightblue"
		s.Require().NoError(s.m.SaveAccount(acc))
	}

	// Sync between devices is triggered automatically
	// via watch account changes subscription
	// Retrieve community link & community
	err = tt.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.Accounts) != 1 {
			return errors.New("no sync wallet account received")
		}
		return nil
	})
	s.Require().NoError(err)

	acc2, err = alicesOtherDevice.settings.GetAccounts()
	s.Require().NoError(err, "alicesOtherDevice.settings.GetAccounts")
	s.Len(acc2, expectedTotalNumOfAccounts, "Must have all wallet accounts plus one for the Status profile account")

	for _, syncedAcc := range acc2 {
		if syncedAcc.Chat {
			continue
		}
		found := false
		for _, sentAcc := range walletAccounts {
			if syncedAcc.Address == sentAcc.Address {
				// Check account values match the expected values
				s.Require().Equal(sentAcc.Address, syncedAcc.Address)
				s.Require().Equal(sentAcc.Path, syncedAcc.Path)
				s.Require().Equal(sentAcc.KeyUID, syncedAcc.KeyUID)
				s.Require().Equal(sentAcc.Name, syncedAcc.Name)
				s.Require().Equal(sentAcc.Color, syncedAcc.Color)
				s.Require().Equal(sentAcc.Type, syncedAcc.Type)
				s.Require().Equal(sentAcc.KeypairName, syncedAcc.KeypairName)
				s.Require().Equal(sentAcc.DerivedFrom, syncedAcc.DerivedFrom)
				found = true
			}
		}
		s.Require().True(found)
	}
}
