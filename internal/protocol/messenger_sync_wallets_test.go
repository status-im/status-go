package protocol

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	mock_protocol_accounts_manager "github.com/status-im/status-go/internal/protocol/mock"
	"github.com/status-im/status-go/internal/testutils"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func TestMessengerSyncWalletSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncWalletSuite))
}

type MessengerSyncWalletSuite struct {
	MessengerBaseTestSuite

	accountsManagerMock *mock_protocol_accounts_manager.MockAccountsManager
}

func (s *MessengerSyncWalletSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()

	ctrl := gomock.NewController(s.T())
	s.accountsManagerMock = mock_protocol_accounts_manager.NewMockAccountsManager(ctrl)
	s.m.accountsManager = s.accountsManagerMock
}

// user should not be able to change a keypair name directly, it follows display name
func (s *MessengerSyncWalletSuite) TestProfileKeypairNameChange() {
	profileKp, _, _, err := accounts.GetProfileKeypairForTest(true, false, false)
	s.Require().NoError(err)
	profileKp.KeyUID = s.m.account.KeyUID
	profileKp.Name = s.m.account.Name
	profileKp.Accounts[0].KeyUID = s.m.account.KeyUID

	// Create a main account on alice
	err = s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err, "profile keypair alice.settings.SaveOrUpdateKeypair")

	// Check account is present in the db
	dbProfileKp, err := s.m.settings.GetKeypairByKeyUID(profileKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairs(profileKp, dbProfileKp))

	// Try to change profile keypair name using `SaveOrUpdateKeypair` function
	profileKp1, _, _, err := accounts.GetProfileKeypairForTest(true, false, false)
	s.Require().NoError(err)
	profileKp1.Name = profileKp1.Name + "updated"
	profileKp1.KeyUID = s.m.account.KeyUID
	profileKp1.Accounts[0].KeyUID = s.m.account.KeyUID

	err = s.m.UpdateKeypair(profileKp1)
	s.Require().Error(err)
	s.Require().True(err == ErrCannotChangeKeypairName)

	// Check the db
	dbProfileKp, err = s.m.settings.GetKeypairByKeyUID(profileKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairs(profileKp, dbProfileKp))

	// Try to change profile keypair name using `UpdateKeypairName` function
	err = s.m.UpdateKeypairName(profileKp1.KeyUID, profileKp1.Name)
	s.Require().Error(err)
	s.Require().True(err == ErrCannotChangeKeypairName)

	// Check the db
	dbProfileKp, err = s.m.settings.GetKeypairByKeyUID(profileKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairs(profileKp, dbProfileKp))
}

func (s *MessengerSyncWalletSuite) TestSyncWallets() {
	profileKp, _, _, err := accounts.GetProfileKeypairForTest(true, true, true)
	s.Require().NoError(err)
	// set clocks for accounts
	profileKp.Clock = uint64(len(profileKp.Accounts) - 1)
	for i, acc := range profileKp.Accounts {
		acc.Clock = uint64(i)
	}

	// Create a main account on alice
	err = s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err, "profile keypair alice.settings.SaveOrUpdateKeypair")

	// Check account is present in the db
	dbProfileKp1, err := s.m.settings.GetKeypairByKeyUID(profileKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairs(profileKp, dbProfileKp1))

	// Create new device and add main account to
	alicesOtherDevice := s.anotherMessenger()

	// Store only chat and default wallet account on other device
	profileKpOtherDevice, _, _, err := accounts.GetProfileKeypairForTest(true, true, false)
	s.Require().NoError(err)
	err = alicesOtherDevice.settings.SaveOrUpdateKeypair(profileKpOtherDevice)
	s.Require().NoError(err, "profile keypair alicesOtherDevice.settings.SaveOrUpdateKeypair")

	// Check account is present in the db
	dbProfileKp2, err := alicesOtherDevice.settings.GetKeypairByKeyUID(profileKpOtherDevice.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairs(profileKpOtherDevice, dbProfileKp2))

	// Pair devices
	im1 := &messagingtypes.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, im1)
	s.Require().NoError(err)
	response, err := alicesOtherDevice.SendPairInstallation(context.Background(), "", nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations()) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations()[0]
	s.Require().Equal(alicesOtherDevice.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("alice's-other-device", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("alice's-other-device-type", actualInstallation.InstallationMetadata.DeviceType)

	_, err = s.m.EnableInstallation(alicesOtherDevice.installationID)
	s.Require().NoError(err)

	// Store seed phrase keypair with accounts on alice's device
	seedPhraseKp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	err = s.m.settings.SaveOrUpdateKeypair(seedPhraseKp)
	s.Require().NoError(err, "seed phrase keypair alice.settings.SaveOrUpdateKeypair")

	dbSeedPhraseKp1, err := s.m.settings.GetKeypairByKeyUID(seedPhraseKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairs(seedPhraseKp, dbSeedPhraseKp1))

	// Store private key keypair with accounts on alice's device
	privKeyKp := accounts.GetPrivKeyImportedKeypairForTest()
	err = s.m.settings.SaveOrUpdateKeypair(privKeyKp)
	s.Require().NoError(err, "private key keypair alice.settings.SaveOrUpdateKeypair")

	dbPrivKeyKp1, err := s.m.settings.GetKeypairByKeyUID(privKeyKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairs(privKeyKp, dbPrivKeyKp1))

	// Store watch only accounts on alice's device
	woAccounts := accounts.GetWatchOnlyAccountsForTest()
	err = s.m.settings.SaveOrUpdateAccounts(woAccounts, false)
	s.Require().NoError(err)
	dbWoAccounts1, err := s.m.settings.GetActiveWatchOnlyAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbWoAccounts1))
	s.Require().True(haveSameElements(woAccounts, dbWoAccounts1, accounts.SameAccounts))

	dbAccounts1, err := s.m.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(profileKp.Accounts)+len(seedPhraseKp.Accounts)+len(privKeyKp.Accounts)+len(woAccounts), len(dbAccounts1))

	// Trigger's a sync between devices
	err = s.m.SyncDevices(context.Background(), "ens-name", "profile-image", false, nil)
	s.Require().NoError(err)

	err = testutils.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.Keypairs) != 3 || // 3 keypairs (profile, seed, priv key)
			len(response.WatchOnlyAccounts) != len(woAccounts) {
			return errors.New("no sync wallet account received")
		}
		return nil
	})
	s.Require().NoError(err)

	dbProfileKp2, err = alicesOtherDevice.settings.GetKeypairByKeyUID(profileKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(profileKp.KeyUID == dbProfileKp2.KeyUID &&
		profileKp.Name == dbProfileKp2.Name &&
		profileKp.Type == dbProfileKp2.Type &&
		profileKp.DerivedFrom == dbProfileKp2.DerivedFrom &&
		profileKp.LastUsedDerivationIndex == dbProfileKp2.LastUsedDerivationIndex &&
		profileKp.Clock == dbProfileKp2.Clock &&
		len(profileKp.Accounts) == len(dbProfileKp2.Accounts))
	// chat and default wallet account should be fully operable, other accounts partially operable
	for i := range profileKp.Accounts {
		match := false
		expectedOperableValue := accsmanagementtypes.AccountPartiallyOperable
		if profileKp.Accounts[i].Chat || profileKp.Accounts[i].Wallet {
			expectedOperableValue = accsmanagementtypes.AccountFullyOperable
		}
		for j := range dbProfileKp2.Accounts {
			if accounts.SameAccountsWithDifferentOperable(profileKp.Accounts[i], dbProfileKp2.Accounts[j], expectedOperableValue) {
				match = true
				break
			}
		}
		s.Require().True(match)
	}

	dbSeedPhraseKp2, err := alicesOtherDevice.settings.GetKeypairByKeyUID(seedPhraseKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairsWithDifferentSyncedFrom(seedPhraseKp, dbSeedPhraseKp2, true, "", accsmanagementtypes.AccountNonOperable))

	dbPrivKeyKp2, err := alicesOtherDevice.settings.GetKeypairByKeyUID(privKeyKp.KeyUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeypairsWithDifferentSyncedFrom(privKeyKp, dbPrivKeyKp2, true, "", accsmanagementtypes.AccountNonOperable))

	dbWoAccounts2, err := alicesOtherDevice.settings.GetActiveWatchOnlyAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbWoAccounts2))
	s.Require().True(haveSameElements(woAccounts, dbWoAccounts2, accounts.SameAccounts))

	dbAccounts2, err := alicesOtherDevice.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(profileKp.Accounts)+len(seedPhraseKp.Accounts)+len(privKeyKp.Accounts)+len(woAccounts), len(dbAccounts2))

	s.Require().True(haveSameElements(dbAccounts1, dbAccounts2, accounts.SameAccounts))

	// Update keypair name on alice's primary device
	profileKpUpdated, _, _, err := accounts.GetProfileKeypairForTest(true, true, false)
	s.Require().NoError(err)
	profileKpUpdated.Name = profileKp.Name + "Updated"
	profileKpUpdated.Accounts = profileKp.Accounts[:0]
	err = s.m.UpdateKeypair(profileKpUpdated)
	s.Require().NoError(err, "updated keypair name on alice primary device")

	// Sync between devices is triggered automatically
	// via watch account changes subscription
	// Retrieve community link & community
	err = testutils.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.Keypairs) != 1 {
			return errors.New("no sync keypairs received")
		}
		return nil
	})
	s.Require().NoError(err)

	// check on alice's other device
	dbProfileKp2, err = alicesOtherDevice.settings.GetKeypairByKeyUID(profileKp.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(profileKpUpdated.Name, dbProfileKp2.Name)

	// Update accounts on alice's primary device
	profileKpUpdated, _, _, err = accounts.GetProfileKeypairForTest(true, true, true)
	s.Require().NoError(err)
	accountsToUpdate := profileKpUpdated.Accounts[2:]
	for _, acc := range accountsToUpdate {
		acc.Name = acc.Name + "Updated"
		acc.ColorID = acc.ColorID + "Updated"
		acc.Emoji = acc.Emoji + "Updated"
		err = s.m.UpdateAccount(acc)
		s.Require().NoError(err, "updated account on alice primary device")
	}

	err = testutils.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.Keypairs) != 2 {
			return errors.New("no sync keypairs received")
		}
		return nil
	})
	s.Require().NoError(err)

	// check on alice's other device
	dbProfileKp2, err = alicesOtherDevice.settings.GetKeypairByKeyUID(profileKp.KeyUID)
	s.Require().NoError(err)
	for _, acc := range accountsToUpdate {
		s.Require().True(contains(dbProfileKp2.Accounts, acc, accounts.SameAccounts))
	}
}

func (s *MessengerSyncWalletSuite) TestSyncWalletAccountsReorder() {
	profileKp, _, _, err := accounts.GetProfileKeypairForTest(true, false, false)
	s.Require().NoError(err)
	profileKp.Accounts[0].Position = -1 // Chat account must be at position -1 always

	woAccounts := []*accsmanagementtypes.Account{
		{Address: types.Address{0x11}, Type: accsmanagementtypes.AccountTypeWatch, Position: 0},
		{Address: types.Address{0x12}, Type: accsmanagementtypes.AccountTypeWatch, Position: 1},
		{Address: types.Address{0x13}, Type: accsmanagementtypes.AccountTypeWatch, Position: 2},
		{Address: types.Address{0x14}, Type: accsmanagementtypes.AccountTypeWatch, Position: 3},
		{Address: types.Address{0x15}, Type: accsmanagementtypes.AccountTypeWatch, Position: 4},
		{Address: types.Address{0x16}, Type: accsmanagementtypes.AccountTypeWatch, Position: 5},
	}

	// Create a main account on alice
	err = s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err, "profile keypair alice.settings.SaveOrUpdateKeypair")
	// Store watch only accounts on alice's device
	err = s.m.settings.SaveOrUpdateAccounts(woAccounts, false)
	s.Require().NoError(err, "wo accounts alice.settings.SaveOrUpdateKeypair")

	dbAccounts, err := s.m.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbAccounts)-1)

	// Create a main account on alice's other device
	alicesOtherDevice := s.anotherMessenger()

	err = alicesOtherDevice.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err, "profile keypair alice.settings.SaveOrUpdateKeypair")
	// Store watch only accounts on alice's other device
	err = alicesOtherDevice.settings.SaveOrUpdateAccounts(woAccounts, false)
	s.Require().NoError(err, "wo accounts alice.settings.SaveOrUpdateKeypair")

	dbAccounts, err = alicesOtherDevice.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbAccounts)-1)

	// Pair devices
	im1 := &messagingtypes.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, im1)
	s.Require().NoError(err)
	response, err := alicesOtherDevice.SendPairInstallation(context.Background(), "", nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations()) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations()[0]
	s.Require().Equal(alicesOtherDevice.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("alice's-other-device", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("alice's-other-device-type", actualInstallation.InstallationMetadata.DeviceType)

	_, err = s.m.EnableInstallation(alicesOtherDevice.installationID)
	s.Require().NoError(err)

	// Move down account from position 1 to position 4
	err = s.m.MoveWalletAccount(1, 4)
	s.Require().NoError(err)

	// Expected after moving down
	woAccounts = []*accsmanagementtypes.Account{
		{Address: types.Address{0x11}, Type: accsmanagementtypes.AccountTypeWatch, Position: 0},
		{Address: types.Address{0x13}, Type: accsmanagementtypes.AccountTypeWatch, Position: 1},
		{Address: types.Address{0x14}, Type: accsmanagementtypes.AccountTypeWatch, Position: 2},
		{Address: types.Address{0x15}, Type: accsmanagementtypes.AccountTypeWatch, Position: 3},
		{Address: types.Address{0x12}, Type: accsmanagementtypes.AccountTypeWatch, Position: 4}, // acc with addr 0x12 is at position 4 (moved from position 1)
		{Address: types.Address{0x16}, Type: accsmanagementtypes.AccountTypeWatch, Position: 5},
	}

	dbAccounts, err = s.m.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbAccounts)-1)
	for i := 0; i < len(woAccounts); i++ {
		s.Require().True(accounts.SameAccountsIncludingPosition(woAccounts[i], dbAccounts[i+1]))
	}

	// Sync between devices is triggered automatically
	err = testutils.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.AccountsPositions) != len(woAccounts) {
			return errors.New("no sync message received for accounts reordering")
		}
		return nil
	})
	s.Require().NoError(err)

	// check on alice's other device
	dbAccounts, err = alicesOtherDevice.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbAccounts)-1)
	for i := 0; i < len(woAccounts); i++ {
		s.Require().True(accounts.SameAccountsIncludingPosition(woAccounts[i], dbAccounts[i+1]))
	}

	// compare times
	dbClock, err := s.m.settings.GetClockOfLastAccountsPositionChange()
	s.Require().NoError(err)
	dbClockOtherDevice, err := s.m.settings.GetClockOfLastAccountsPositionChange()
	s.Require().NoError(err)
	s.Require().Equal(dbClock, dbClockOtherDevice)

	// Move up account from position 5 to position 0
	err = s.m.MoveWalletAccount(5, 0)
	s.Require().NoError(err)

	// Expected after moving down
	woAccounts = []*accsmanagementtypes.Account{
		{Address: types.Address{0x16}, Type: accsmanagementtypes.AccountTypeWatch, Position: 0}, // acc with addr 0x16 is at position 0 (moved from position 5)
		{Address: types.Address{0x11}, Type: accsmanagementtypes.AccountTypeWatch, Position: 1},
		{Address: types.Address{0x13}, Type: accsmanagementtypes.AccountTypeWatch, Position: 2},
		{Address: types.Address{0x14}, Type: accsmanagementtypes.AccountTypeWatch, Position: 3},
		{Address: types.Address{0x15}, Type: accsmanagementtypes.AccountTypeWatch, Position: 4},
		{Address: types.Address{0x12}, Type: accsmanagementtypes.AccountTypeWatch, Position: 5},
	}

	dbAccounts, err = s.m.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbAccounts)-1)
	for i := 0; i < len(woAccounts); i++ {
		s.Require().True(accounts.SameAccountsIncludingPosition(woAccounts[i], dbAccounts[i+1]))
	}

	// Sync between devices is triggered automatically
	err = testutils.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.AccountsPositions) != len(woAccounts) {
			return errors.New("no sync message received for accounts reordering")
		}
		return nil
	})
	s.Require().NoError(err)

	// check on alice's other device
	dbAccounts, err = alicesOtherDevice.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbAccounts)-1)
	for i := 0; i < len(woAccounts); i++ {
		s.Require().True(accounts.SameAccountsIncludingPosition(woAccounts[i], dbAccounts[i+1]))
	}

	// compare times
	dbClock, err = s.m.settings.GetClockOfLastAccountsPositionChange()
	s.Require().NoError(err)
	dbClockOtherDevice, err = s.m.settings.GetClockOfLastAccountsPositionChange()
	s.Require().NoError(err)
	s.Require().Equal(dbClock, dbClockOtherDevice)
}

func (s *MessengerSyncWalletSuite) TestSyncWalletAccountOrderAfterDeletion() {
	profileKp, _, _, err := accounts.GetProfileKeypairForTest(true, true, true)
	s.Require().NoError(err)
	// set clocks for accounts
	profileKp.Clock = uint64(len(profileKp.Accounts) - 1)
	i := -1
	for _, acc := range profileKp.Accounts {
		acc.Clock = uint64(i + 1)
		acc.Position = int64(i)
		acc.Operable = accsmanagementtypes.AccountNonOperable
		i++
	}

	// Create a main account on alice
	err = s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err, "profile keypair alice.settings.SaveOrUpdateKeypair")
	// Store seed phrase keypair with accounts on alice's device
	seedPhraseKp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	for _, acc := range seedPhraseKp.Accounts {
		acc.Clock = uint64(i + 1)
		acc.Position = int64(i)
		acc.Operable = accsmanagementtypes.AccountNonOperable
		i++
	}
	err = s.m.settings.SaveOrUpdateKeypair(seedPhraseKp)
	s.Require().NoError(err, "seed phrase keypair alice.settings.SaveOrUpdateKeypair")
	// Store private key keypair with accounts on alice's device
	privKeyKp := accounts.GetPrivKeyImportedKeypairForTest()
	for _, acc := range privKeyKp.Accounts {
		acc.Clock = uint64(i + 1)
		acc.Position = int64(i)
		acc.Operable = accsmanagementtypes.AccountNonOperable
		i++
	}
	err = s.m.settings.SaveOrUpdateKeypair(privKeyKp)
	s.Require().NoError(err, "private key keypair alice.settings.SaveOrUpdateKeypair")
	// Store watch only accounts on alice's device
	woAccounts := accounts.GetWatchOnlyAccountsForTest()
	for _, acc := range woAccounts {
		acc.Clock = uint64(i + 1)
		acc.Position = int64(i)
		acc.Operable = accsmanagementtypes.AccountFullyOperable
		i++
	}
	err = s.m.settings.SaveOrUpdateAccounts(woAccounts, false)
	s.Require().NoError(err)
	// Check accounts
	dbAccounts1, err := s.m.settings.GetActiveAccounts()
	s.Require().NoError(err)
	totalNumOfAccounts := len(profileKp.Accounts) + len(seedPhraseKp.Accounts) + len(privKeyKp.Accounts) + len(woAccounts)
	s.Require().Equal(totalNumOfAccounts, len(dbAccounts1))

	// Create new device and add main account to
	alicesOtherDevice := s.anotherMessenger()

	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	accountsManagerAnotherMock := mock_protocol_accounts_manager.NewMockAccountsManager(ctrl)
	alicesOtherDevice.accountsManager = accountsManagerAnotherMock

	// Store only chat and default wallet account on other device
	profileKpOtherDevice, _, _, err := accounts.GetProfileKeypairForTest(true, true, false)
	s.Require().NoError(err)
	err = alicesOtherDevice.settings.SaveOrUpdateKeypair(profileKpOtherDevice)
	s.Require().NoError(err, "profile keypair alicesOtherDevice.settings.SaveOrUpdateKeypair")

	// Pair devices
	im1 := &messagingtypes.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, im1)
	s.Require().NoError(err)
	response, err := alicesOtherDevice.SendPairInstallation(context.Background(), "", nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations()) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations()[0]
	s.Require().Equal(alicesOtherDevice.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("alice's-other-device", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("alice's-other-device-type", actualInstallation.InstallationMetadata.DeviceType)

	_, err = s.m.EnableInstallation(alicesOtherDevice.installationID)
	s.Require().NoError(err)

	// Trigger's a sync between devices
	err = s.m.SyncDevices(context.Background(), "ens-name", "profile-image", false, nil)
	s.Require().NoError(err)

	err = testutils.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.Keypairs) != 3 || // 3 keypairs (profile, seed, priv key)
			len(response.WatchOnlyAccounts) != len(woAccounts) ||
			len(response.AccountsPositions) != totalNumOfAccounts-1 /* we don't include chat account in position ordering*/ {
			return errors.New("no sync wallet account received")
		}
		return nil
	})
	s.Require().NoError(err)

	dbAccounts2, err := alicesOtherDevice.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(totalNumOfAccounts, len(dbAccounts2))

	s.Require().True(haveSameElements(dbAccounts1, dbAccounts2, accounts.SameAccountsIncludingPosition))

	// Delete keypair related account on alice's primary device
	accToDelete := seedPhraseKp.Accounts[1]

	s.accountsManagerMock.EXPECT().DeleteAccount(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(address types.Address, password string, clock uint64) (*accsmanagementtypes.Account, error) {
			err := s.m.settings.RemoveAccount(address, clock)
			if err != nil {
				return nil, err
			}
			return accToDelete, nil
		}).Times(1)

	err = s.m.DeleteAccount(accToDelete.Address, "")
	s.Require().NoError(err, "delete account on alice primary device")

	totalNumOfAccounts-- //one acc less

	dbAccounts1, err = s.m.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(totalNumOfAccounts, len(dbAccounts1))

	err = testutils.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.Keypairs) != 1 {
			return errors.New("no sync keypairs received")
		}
		return nil
	})
	s.Require().NoError(err)

	dbAccounts2, err = alicesOtherDevice.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(totalNumOfAccounts, len(dbAccounts2))

	s.Require().True(haveSameElements(dbAccounts1, dbAccounts2, accounts.SameAccountsIncludingPosition))

	// Delete watch only account on alice's primary device
	accToDelete = woAccounts[1]

	s.accountsManagerMock.EXPECT().DeleteAccount(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(address types.Address, password string, clock uint64) (*accsmanagementtypes.Account, error) {
			err := s.m.settings.RemoveAccount(address, clock)
			if err != nil {
				return nil, err
			}
			return accToDelete, nil
		}).Times(1)

	err = s.m.DeleteAccount(accToDelete.Address, "")
	s.Require().NoError(err, "delete account on alice primary device")

	totalNumOfAccounts-- //one acc less

	err = testutils.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.WatchOnlyAccounts) != 1 {
			return errors.New("no sync keypairs received")
		}
		return nil
	})
	s.Require().NoError(err)

	dbAccounts1, err = s.m.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(totalNumOfAccounts, len(dbAccounts1))

	dbAccounts2, err = alicesOtherDevice.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Equal(totalNumOfAccounts, len(dbAccounts2))

	s.Require().True(haveSameElements(dbAccounts1, dbAccounts2, accounts.SameAccountsIncludingPosition))
}

func (s *MessengerSyncWalletSuite) pairOtherDevice() *Messenger {
	otherDevice := s.anotherMessenger()

	im := &messagingtypes.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err := otherDevice.SetInstallationMetadata(otherDevice.installationID, im)
	s.Require().NoError(err)

	PairDevices(&s.Suite, otherDevice, s.m)

	return otherDevice
}

func (s *MessengerSyncWalletSuite) saveSeedKeypairOn(m *Messenger, coldWallet accsmanagementtypes.ColdWalletType) *accsmanagementtypes.Keypair {
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.Clock = 1
	kp.ColdWallet = coldWallet
	for _, acc := range kp.Accounts {
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(m.settings.SaveOrUpdateKeypair(kp))
	return kp
}

func (s *MessengerSyncWalletSuite) waitForKeypairOn(otherDevice *Messenger) {
	err := testutils.RetryWithBackOff(func() error {
		response, err := otherDevice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Keypairs) != 1 {
			return errors.New("no sync keypair received")
		}
		return nil
	})
	s.Require().NoError(err)
}

func (s *MessengerSyncWalletSuite) TestMigrateKeypairToColdWalletSyncsColdStateToPairedDevice() {
	kp := s.saveSeedKeypairOn(s.m, accsmanagementtypes.ColdWalletTypeNone)
	otherDevice := s.pairOtherDevice()
	s.saveSeedKeypairOn(otherDevice, accsmanagementtypes.ColdWalletTypeNone)

	s.accountsManagerMock.EXPECT().
		MigrateKeypairToColdWallet(kp.KeyUID, "password", accsmanagementtypes.ColdWalletTypeStatusKeycard, gomock.Any()).
		DoAndReturn(func(keyUID string, password string, coldWallet accsmanagementtypes.ColdWalletType, clock uint64) error {
			return s.m.settings.UpdateKeypairXPub(keyUID, "", coldWallet, clock)
		}).Times(1)

	err := s.m.MigrateKeypairToColdWallet(context.Background(), kp.KeyUID, "password", accsmanagementtypes.ColdWalletTypeStatusKeycard)
	s.Require().NoError(err)

	s.waitForKeypairOn(otherDevice)

	dbKp, err := otherDevice.settings.GetKeypairByKeyUID(kp.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet,
		"the paired device must learn the keypair signs via keycard now, or it will keep trying deleted keystore files")
}

func (s *MessengerSyncWalletSuite) TestMigrateColdWalletKeypairToAppSyncsRevertToPairedDevice() {
	kp := s.saveSeedKeypairOn(s.m, accsmanagementtypes.ColdWalletTypeStatusKeycard)
	otherDevice := s.pairOtherDevice()
	s.saveSeedKeypairOn(otherDevice, accsmanagementtypes.ColdWalletTypeStatusKeycard)

	s.accountsManagerMock.EXPECT().
		MigrateColdWalletKeypairToApp("some mnemonic", "password", gomock.Any()).
		DoAndReturn(func(mnemonic string, password string, clock uint64) (string, error) {
			return kp.KeyUID, s.m.settings.UpdateKeypairXPub(kp.KeyUID, "", accsmanagementtypes.ColdWalletTypeNone, clock)
		}).Times(1)

	err := s.m.MigrateColdWalletKeypairToApp(context.Background(), "some mnemonic", "password")
	s.Require().NoError(err)

	s.waitForKeypairOn(otherDevice)

	dbKp, err := otherDevice.settings.GetKeypairByKeyUID(kp.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeNone, dbKp.ColdWallet,
		"the paired device must learn the keypair signs via keystore again, or its signing method stays keycard forever")
}

func (s *MessengerSyncWalletSuite) TestAddKeypairStoredToColdWalletSyncsKeypairToPairedDevice() {
	otherDevice := s.pairOtherDevice()

	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.XPub = "xpub6LedgerColdWalletKeypair"
	kp.ColdWallet = accsmanagementtypes.ColdWalletTypeLedger

	s.accountsManagerMock.EXPECT().
		AddKeypairStoredToColdWallet(kp.KeyUID, kp.DerivedFrom, kp.Name, kp.XPub, accsmanagementtypes.ColdWalletTypeLedger, gomock.Any(), gomock.Any()).
		DoAndReturn(func(keyUID string, masterAddress string, name string, walletXPub string, coldWallet accsmanagementtypes.ColdWalletType,
			walletAccounts []*accsmanagementtypes.Account, clock uint64) (*accsmanagementtypes.Keypair, error) {
			// the accounts the messenger forwarded, not the ones captured above:
			// otherwise a nil or wrong slice would still sync two accounts
			s.Require().Equal(kp.Accounts, walletAccounts, "the messenger must forward the wallet accounts it was given")
			kp.Clock = clock
			for i, acc := range kp.Accounts {
				acc.Position = int64(i)
				acc.Operable = accsmanagementtypes.AccountFullyOperable
			}
			return kp, s.m.settings.SaveOrUpdateKeypair(kp)
		}).Times(1)

	returnedKp, err := s.m.AddKeypairStoredToColdWallet(kp.KeyUID, kp.DerivedFrom, kp.Name, kp.XPub, accsmanagementtypes.ColdWalletTypeLedger, kp.Accounts)
	s.Require().NoError(err)

	s.waitForKeypairOn(otherDevice)

	dbKp, err := otherDevice.settings.GetKeypairByKeyUID(kp.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeLedger, dbKp.ColdWallet,
		"the paired device must know the keypair lives on a ledger, there are no keystore files to fall back to")
	s.Require().Equal(kp.XPub, dbKp.XPub,
		"the xpub must survive the wire trip, password-less account derivation on the paired device depends on it")
	s.Require().Equal(returnedKp.Name, dbKp.Name, "the keypair must arrive under the name it was created with")
	s.Require().Equal(returnedKp.DerivedFrom, dbKp.DerivedFrom, "the master address must survive the wire trip")
	s.Require().Equal(len(returnedKp.Accounts), len(dbKp.Accounts), "every ledger wallet account must reach the paired device")
	for _, expected := range returnedKp.Accounts {
		found := false
		for _, got := range dbKp.Accounts {
			if got.Address == expected.Address {
				found = true
				break
			}
		}
		s.Require().True(found, "wallet account %s must exist on the paired device", expected.Address.Hex())
	}
}
