package accountsmanagement

import (
	"go.uber.org/mock/gomock"

	common "github.com/status-im/status-go/internal/accounts-management/common"
	generator "github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/keystore"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// setupProfileKeystore initializes the keystore for the profile keypair's keyUID and
// optionally stores the profile master keystore file — the password-verification oracle
// for cold-wallet reverse migrations.
func (s *ManagerTestSuite) setupProfileKeystore(storeMasterFile bool) {
	ks, err := s.accManager.createKeystore(s.masterAccount.KeyUID())
	s.Require().NoError(err)
	s.accManager.setKeystore(ks)
	if storeMasterFile {
		s.Require().NoError(s.accManager.storeToKeystore(s.masterAccount, s.password))
	}
}

// requireKeypairKeystoreFilesOpen opens the master file and every account file of
// the keypair with the given password. A file count alone would pass on a wrong
// child key or a file encrypted under a different password.
func (s *ManagerTestSuite) requireKeypairKeystoreFilesOpen(mnemonic, password string, paths []string) {
	master, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	s.Require().NoError(err)
	restored, err := s.accManager.loadAccountInternally(master.Address(), password)
	s.Require().NoError(err, "the restored master keystore file must decrypt with the given password")
	s.Require().NotNil(restored)

	_, derived, err := generator.CreateAndDeriveAccountsFromMnemonic(mnemonic, paths, "")
	s.Require().NoError(err)
	for _, path := range paths {
		acc, err := s.accManager.loadAccountInternally(derived[path].Address(), password)
		s.Require().NoError(err, "the restored keystore file for %s must decrypt with the given password", path)
		s.Require().NotNil(acc)
	}
}

// createColdSeedKeypair builds a non-profile seed keypair already migrated to a cold wallet,
// derived from a fresh mnemonic so its KeyUID differs from the profile keypair's.
func (s *ManagerTestSuite) createColdSeedKeypair() (mnemonic string, keypair *accsmanagementtypes.Keypair) {
	mnemonic, err := common.CreateRandomMnemonicWithDefaultLength()
	s.Require().NoError(err)
	master, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	s.Require().NoError(err)

	keypair = &accsmanagementtypes.Keypair{
		KeyUID:      master.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		ColdWallet:  accsmanagementtypes.ColdWalletTypeStatusKeycard,
		DerivedFrom: master.Address().Hex(),
		Accounts: []*accsmanagementtypes.Account{
			{
				KeyUID:   master.KeyUID(),
				Path:     common.PathDefaultWalletAccount,
				Operable: accsmanagementtypes.AccountFullyOperable,
			},
		},
	}
	return mnemonic, keypair
}

func (s *ManagerTestSuite) TestMigrateKeypairToColdWalletRejectsWrongPassword() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.Type = accsmanagementtypes.KeypairTypeSeed

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet(keypair.KeyUID, "wrong-password", accsmanagementtypes.ColdWalletTypeStatusKeycard, 1)
	s.Require().Error(err, "migration with a wrong password must be rejected, else cold_wallet flips while decryptable key files stay on disk")
	s.Require().ErrorIs(err, keystore.ErrIncorrectPasswordProvided, "rejection must come from the keystore delete step refusing the wrong password")
	s.Require().Equal(3, s.countKeystoreFiles(), "all keystore files must remain on disk after a rejected migration")
}

func (s *ManagerTestSuite) TestMigrateProfileKeypairToColdWalletDeletesChatWalletAndMasterKeystoreFiles() {
	keypair := s.createAndStoreProfileKeypair()

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().MarkKeypairFullyOperable(keypair.KeyUID, uint64(4), true).Return(nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, "", accsmanagementtypes.ColdWalletTypeStatusKeycard, uint64(4)).Return(nil).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet(keypair.KeyUID, s.password, accsmanagementtypes.ColdWalletTypeStatusKeycard, 4)
	s.Require().NoError(err, "migrating the profile keypair to a keycard must succeed with the profile password")
	s.Require().Equal(0, s.countKeystoreFiles(),
		"the chat, wallet and master keystore files must all be deleted, else the profile's private keys stay decryptable on disk")

	_, err = s.accManager.loadAccountInternally(s.chatAddress, s.password)
	s.Require().ErrorIs(err, keystore.ErrKeystoreFileMissing, "the chat account file must be gone")

	chatAccount, err := s.accManager.SelectedChatAccount()
	s.Require().NoError(err, "the running session keeps its chat account, the migration only removes the files")
	s.Require().Equal(s.chatAddress, chatAccount.Address())
}

func (s *ManagerTestSuite) TestMigrateKeypairToColdWalletRetryAfterInterruptedDeleteSucceeds() {
	keypair := s.createAndStoreProfileKeypair()
	// an earlier attempt deleted the wallet account file, then died before flagging the keypair cold
	s.Require().NoError(s.accManager.deleteAccountFromKeystoreIfExists(s.walletAddress, s.password))
	s.Require().Equal(2, s.countKeystoreFiles())

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().MarkKeypairFullyOperable(keypair.KeyUID, uint64(6), true).Return(nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, "", accsmanagementtypes.ColdWalletTypeStatusKeycard, uint64(6)).Return(nil).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet(keypair.KeyUID, s.password, accsmanagementtypes.ColdWalletTypeStatusKeycard, 6)
	s.Require().NoError(err, "a keystore file that is already gone must not block the retry")
	s.Require().Equal(0, s.countKeystoreFiles())
}

func (s *ManagerTestSuite) TestMigrateColdWalletKeypairToAppRejectsWrongPassword() {
	s.setupProfileKeystore(true)
	mnemonic2, coldKp := s.createColdSeedKeypair()

	s.persistence.EXPECT().GetKeypairByKeyUID(coldKp.KeyUID).Return(coldKp, nil).Times(1)
	s.persistence.EXPECT().GetProfileKeypair().Return(&accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom: s.masterAccount.Address().Hex(),
	}, nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	_, err := s.accManager.MigrateColdWalletKeypairToApp(mnemonic2, "wrong-password", 2)
	s.Require().Error(err, "reverse migration must reject a password that fails against the profile master keystore file")
	s.requireAccountsErrorCode(err, ErrCodeWrongPasswordProvided)
	s.Require().Equal(filesBefore, s.countKeystoreFiles(),
		"no keystore file may be written when the password was not verified")
}

func (s *ManagerTestSuite) TestMigrateColdWalletKeypairToAppRestoresKeystoreFilesAndResetsColdState() {
	s.setupProfileKeystore(true)
	mnemonic2, coldKp := s.createColdSeedKeypair()

	s.persistence.EXPECT().GetKeypairByKeyUID(coldKp.KeyUID).Return(coldKp, nil).Times(1)
	s.persistence.EXPECT().GetProfileKeypair().Return(&accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom: s.masterAccount.Address().Hex(),
	}, nil).Times(1)
	// the manager passes an empty xpub; the DB layer treats that as keep-existing
	s.persistence.EXPECT().UpdateKeypairXPub(coldKp.KeyUID, "", accsmanagementtypes.ColdWalletTypeNone, uint64(7)).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	keyUID, err := s.accManager.MigrateColdWalletKeypairToApp(mnemonic2, s.password, 7)
	s.Require().NoError(err, "migrating a cold keypair back to app with the correct profile password must succeed")
	s.Require().Equal(coldKp.KeyUID, keyUID, "returned keyUID must identify the migrated keypair")
	s.Require().Equal(filesBefore+2, s.countKeystoreFiles(),
		"keystore files must be recreated for the master key and every account path")

	s.requireKeypairKeystoreFilesOpen(mnemonic2, s.password, []string{common.PathDefaultWalletAccount})
}

func (s *ManagerTestSuite) TestMigrateColdWalletProfileKeypairToAppRestoresChatWalletAndMasterKeystoreFiles() {
	s.setupProfileKeystore(false)
	coldProfileKp := &accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeProfile,
		ColdWallet:  accsmanagementtypes.ColdWalletTypeStatusKeycard,
		DerivedFrom: s.masterAccount.Address().Hex(),
		Accounts: []*accsmanagementtypes.Account{
			{KeyUID: s.masterAccount.KeyUID(), Path: common.PathEIP1581Chat, Address: s.chatAddress,
				Chat: true, Operable: accsmanagementtypes.AccountFullyOperable},
			{KeyUID: s.masterAccount.KeyUID(), Path: common.PathDefaultWalletAccount, Address: s.walletAddress,
				Wallet: true, Operable: accsmanagementtypes.AccountFullyOperable},
		},
	}

	// no GetProfileKeypair expectation: the profile keypair is the one being migrated,
	// so there is no profile master file to verify the password against
	s.persistence.EXPECT().GetKeypairByKeyUID(coldProfileKp.KeyUID).Return(coldProfileKp, nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(coldProfileKp.KeyUID, "", accsmanagementtypes.ColdWalletTypeNone, uint64(8)).Return(nil).Times(1)

	keyUID, err := s.accManager.MigrateColdWalletKeypairToApp(s.mnemonic, s.password, 8)
	s.Require().NoError(err, "migrating a keycard profile back to the app must succeed without a profile-password check")
	s.Require().Equal(coldProfileKp.KeyUID, keyUID)
	s.Require().Equal(3, s.countKeystoreFiles(), "the master, chat and wallet keystore files must all be recreated")

	s.requireKeypairKeystoreFilesOpen(s.mnemonic, s.password, []string{common.PathEIP1581Chat, common.PathDefaultWalletAccount})
}

func (s *ManagerTestSuite) TestMigrateColdWalletKeypairToAppSkipsPasswordCheckWhenProfileKeypairIsCold() {
	s.setupProfileKeystore(false)
	mnemonic2, coldKp := s.createColdSeedKeypair()

	s.persistence.EXPECT().GetKeypairByKeyUID(coldKp.KeyUID).Return(coldKp, nil).Times(1)
	s.persistence.EXPECT().GetProfileKeypair().Return(&accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeProfile,
		ColdWallet:  accsmanagementtypes.ColdWalletTypeStatusKeycard,
		DerivedFrom: "0x000000000000000000000000000000000000dead",
	}, nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(coldKp.KeyUID, "", accsmanagementtypes.ColdWalletTypeNone, uint64(3)).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	keyUID, err := s.accManager.MigrateColdWalletKeypairToApp(mnemonic2, "never-verified-password", 3)
	s.Require().NoError(err, "migration must proceed without password verification when the profile keypair is itself on a cold wallet")
	s.Require().Equal(coldKp.KeyUID, keyUID)
	s.Require().Equal(filesBefore+2, s.countKeystoreFiles(),
		"keystore files must be recreated for the master key and the account path even though the password could not be verified")

	// the manager passes the password through unverified, so the files must open
	// with exactly that password; a file count alone does not show which secret was used
	s.requireKeypairKeystoreFilesOpen(mnemonic2, "never-verified-password", []string{common.PathDefaultWalletAccount})
}

func (s *ManagerTestSuite) TestMigrateAlreadyMigratedKeypairSwitchesColdWalletTypeWithoutPassword() {
	keypair := &accsmanagementtypes.Keypair{
		KeyUID:     "cold-kp",
		Type:       accsmanagementtypes.KeypairTypeSeed,
		ColdWallet: accsmanagementtypes.ColdWalletTypeStatusKeycard,
	}

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, "", accsmanagementtypes.ColdWalletTypeLedger, uint64(5)).Return(nil).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet(keypair.KeyUID, "", accsmanagementtypes.ColdWalletTypeLedger, 5)
	s.Require().NoError(err, "switching the cold-wallet type of an already-migrated keypair must not demand a password")
}

func (s *ManagerTestSuite) TestMigrateColdWalletKeypairToAppRejectsNonColdKeypair() {
	keypair := &accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: s.masterAccount.Address().Hex(),
	}

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	_, err := s.accManager.MigrateColdWalletKeypairToApp(s.mnemonic, s.password, 0)
	s.Require().Error(err, "reverse migration on a keypair that is not a cold wallet must be rejected")
	s.Require().ErrorIs(err, ErrKeypairIsNotColdWallet, "the typed not-cold-wallet error must be returned for app-side matching")
}

func (s *ManagerTestSuite) TestMigrateColdWalletKeypairToAppUnknownKeypairErrors() {
	s.persistence.EXPECT().GetKeypairByKeyUID(s.masterAccount.KeyUID()).Return(nil, accsmanagementtypes.ErrDbKeypairNotFound).Times(1)

	_, err := s.accManager.MigrateColdWalletKeypairToApp(s.mnemonic, s.password, 0)
	s.Require().Error(err, "a mnemonic deriving to an unknown keyUID must surface the persistence error, not panic")
	s.Require().ErrorIs(err, accsmanagementtypes.ErrDbKeypairNotFound, "the persistence error must be matchable app-side")
}

func (s *ManagerTestSuite) TestAddKeypairStoredToColdWalletPersistsLedgerKeypairWithoutKeystoreFiles() {
	s.setupProfileKeystore(false)

	accounts := []*accsmanagementtypes.Account{
		{KeyUID: "ledger-kp-uid", Path: common.PathWalletRoot + "/0",
			Address: cryptotypes.HexToAddress("0x1000000000000000000000000000000000000001"), Operable: accsmanagementtypes.AccountNonOperable},
		{KeyUID: "ledger-kp-uid", Path: common.PathWalletRoot + "/1",
			Address: cryptotypes.HexToAddress("0x1000000000000000000000000000000000000002"), Operable: accsmanagementtypes.AccountNonOperable},
	}

	s.persistence.EXPECT().GetKeypairByKeyUID("ledger-kp-uid").Return(nil, accsmanagementtypes.ErrDbKeypairNotFound).Times(1)
	s.persistence.EXPECT().GetPositionForNextNewAccount().Return(int64(4), nil).Times(1)
	// capture what is handed to persistence: asserting only on the returned keypair
	// would rely on it being the same object that was saved
	var saved *accsmanagementtypes.Keypair
	s.persistence.EXPECT().SaveOrUpdateKeypair(gomock.Any()).DoAndReturn(
		func(kp *accsmanagementtypes.Keypair) error {
			saved = kp
			return nil
		}).Times(1)

	keypair, err := s.accManager.AddKeypairStoredToColdWallet("ledger-kp-uid", "0xmaster-address", "ledger-kp",
		"ledger-wallet-xpub", accsmanagementtypes.ColdWalletTypeLedger, accounts, 9)
	s.Require().NoError(err, "importing a ledger keypair with valid wallet-path accounts must succeed")

	s.Require().NotNil(saved, "the keypair must reach persistence")
	s.Require().Equal("ledger-kp-uid", saved.KeyUID, "the saved keypair must carry the given keyUID")
	s.Require().Equal("ledger-kp", saved.Name, "the saved keypair must carry the given name")
	s.Require().Equal(accsmanagementtypes.KeypairTypeSeed, saved.Type, "cold-wallet imports are persisted as seed keypairs")
	s.Require().Equal("0xmaster-address", saved.DerivedFrom, "the saved keypair must carry the given master address")
	s.Require().Equal("ledger-wallet-xpub", saved.XPub, "the saved keypair must carry the given xpub")
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeLedger, saved.ColdWallet, "the saved keypair must carry the given cold-wallet type")
	s.Require().Equal(uint64(9), saved.Clock, "the saved keypair must carry the given clock")
	s.Require().Len(saved.Accounts, 2, "both wallet accounts must be saved")
	s.Require().Equal(int64(4), saved.Accounts[0].Position, "positions must be assigned sequentially from GetPositionForNextNewAccount")
	s.Require().Equal(int64(5), saved.Accounts[1].Position, "positions must be assigned sequentially from GetPositionForNextNewAccount")
	s.Require().Equal(accsmanagementtypes.AccountFullyOperable, saved.Accounts[0].Operable, "cold-wallet accounts must be forced fully operable")
	s.Require().Equal(accsmanagementtypes.AccountFullyOperable, saved.Accounts[1].Operable, "cold-wallet accounts must be forced fully operable")

	s.Require().Same(saved, keypair, "the returned keypair must be the one handed to persistence")
	s.Require().Equal(0, s.countKeystoreFiles(), "no keystore file may be written for a cold-wallet keypair import")
}

func (s *ManagerTestSuite) TestAddKeypairStoredToColdWalletRejectsEmptyWalletAccounts() {
	_, err := s.accManager.AddKeypairStoredToColdWallet("some-kp-uid", "0xmaster-address", "kp-name",
		"some-xpub", accsmanagementtypes.ColdWalletTypeStatusKeycard, nil, 1)
	s.Require().Error(err, "a keypair import with zero wallet accounts must be rejected")
	s.Require().ErrorIs(err, ErrKeypairMustHaveAtLeastOneWalletAccount, "the typed at-least-one-account error must be returned for app-side matching")
}

func (s *ManagerTestSuite) TestAddKeypairStoredToColdWalletRejectsNonWalletPath() {
	accounts := []*accsmanagementtypes.Account{
		{KeyUID: "some-kp-uid", Path: "m/45'/60'/0'/0/0",
			Address: cryptotypes.HexToAddress("0x1000000000000000000000000000000000000003")},
	}

	_, err := s.accManager.AddKeypairStoredToColdWallet("some-kp-uid", "0xmaster-address", "kp-name",
		"some-xpub", accsmanagementtypes.ColdWalletTypeStatusKeycard, accounts, 1)
	s.Require().Error(err, "an account path outside the wallet tree must be rejected at import")
	s.Require().ErrorIs(err, ErrUnsupportedWalletAccountPath, "the typed unsupported-path error must be returned for app-side matching")
}

func (s *ManagerTestSuite) TestAddKeypairStoredToColdWalletRejectsAlreadyAddedKeypair() {
	accounts := []*accsmanagementtypes.Account{
		{KeyUID: "existing-kp-uid", Path: common.PathWalletRoot + "/0",
			Address: cryptotypes.HexToAddress("0x1000000000000000000000000000000000000004")},
	}

	s.persistence.EXPECT().GetKeypairByKeyUID("existing-kp-uid").Return(
		&accsmanagementtypes.Keypair{KeyUID: "existing-kp-uid", Type: accsmanagementtypes.KeypairTypeSeed}, nil).Times(1)

	_, err := s.accManager.AddKeypairStoredToColdWallet("existing-kp-uid", "0xmaster-address", "kp-name",
		"some-xpub", accsmanagementtypes.ColdWalletTypeStatusKeycard, accounts, 1)
	s.Require().Error(err, "re-importing an existing keyUID must be rejected")
	s.Require().ErrorIs(err, ErrKeypairAlreadyAdded, "the typed already-added error must be returned for app-side matching")
}

func (s *ManagerTestSuite) TestMigrateKeypairToColdWalletWithOnlyPartialAndRemovedAccountsDeletesMasterKeystore() {
	s.setupProfileKeystore(false)

	mnemonic2, err := common.CreateRandomMnemonicWithDefaultLength()
	s.Require().NoError(err)
	master2, err := generator.CreateAccountFromMnemonic(mnemonic2, "")
	s.Require().NoError(err)
	s.Require().NoError(s.accManager.storeToKeystore(master2, s.password))

	keypair := &accsmanagementtypes.Keypair{
		KeyUID:      master2.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: master2.Address().Hex(),
		XPub:        "already-set",
		Accounts: []*accsmanagementtypes.Account{
			{KeyUID: master2.KeyUID(), Path: common.PathWalletXPub + "/5",
				Address:  cryptotypes.HexToAddress("0x2000000000000000000000000000000000000001"),
				Operable: accsmanagementtypes.AccountPartiallyOperable},
			{KeyUID: master2.KeyUID(), Path: common.PathWalletRoot + "/1",
				Address:  cryptotypes.HexToAddress("0x2000000000000000000000000000000000000002"),
				Operable: accsmanagementtypes.AccountFullyOperable, Removed: true},
			{KeyUID: master2.KeyUID(), Path: common.PathWalletRoot + "/2",
				Address:  cryptotypes.HexToAddress("0x2000000000000000000000000000000000000003"),
				Operable: accsmanagementtypes.AccountNonOperable},
		},
	}

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().MarkKeypairFullyOperable(keypair.KeyUID, uint64(11), true).Return(nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, "", accsmanagementtypes.ColdWalletTypeStatusKeycard, uint64(11)).Return(nil).Times(1)

	err = s.accManager.MigrateKeypairToColdWallet(keypair.KeyUID, s.password, accsmanagementtypes.ColdWalletTypeStatusKeycard, 11)
	s.Require().NoError(err,
		"a keypair holding only a partially-operable account (no per-account keystore file) plus removed/non-operable accounts must still migrate")
	s.Require().Equal(0, s.countKeystoreFiles(),
		"the master keystore file must be deleted on migration, else the master private key stays decryptable on disk for a cold keypair")
}

func (s *ManagerTestSuite) TestMakePartiallyOperableAccountsFullyOperableRequiresPassword() {
	addresses, err := s.accManager.MakePartiallyOperableAccoutsFullyOperable("")
	s.Require().Error(err, "an empty password must be rejected before any keypair is touched")
	s.Require().ErrorIs(err, ErrNoPasswordProvided,
		"the typed no-password error must be returned for app-side matching")
	s.Require().Nil(addresses, "no address may be reported as promoted when the call was rejected")
}

func (s *ManagerTestSuite) TestMakePartiallyOperableAccountsFullyOperableSkipsColdKeypairAndPromotesRegular() {
	keypair := s.createAndStoreProfileKeypair()

	partialAcc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")
	partialAcc.Operable = accsmanagementtypes.AccountPartiallyOperable
	keypair.Accounts = append(keypair.Accounts, partialAcc)

	// the cold keypair is listed FIRST: if the skip-cold guard breaks, its missing master
	// keystore file errors the loop before the regular keypair is ever reached
	coldKp := &accsmanagementtypes.Keypair{
		KeyUID:      "cold-kp",
		Type:        accsmanagementtypes.KeypairTypeSeed,
		ColdWallet:  accsmanagementtypes.ColdWalletTypeStatusKeycard,
		DerivedFrom: "0x000000000000000000000000000000000000dead",
		Accounts: []*accsmanagementtypes.Account{
			{KeyUID: "cold-kp", Path: common.PathWalletRoot + "/0",
				Address:  cryptotypes.HexToAddress("0x3000000000000000000000000000000000000001"),
				Operable: accsmanagementtypes.AccountPartiallyOperable},
		},
	}

	s.persistence.EXPECT().GetActiveKeypairs().Return([]*accsmanagementtypes.Keypair{coldKp, keypair}, nil).Times(1)
	s.persistence.EXPECT().MarkAccountFullyOperable(partialAcc.Address).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	addresses, err := s.accManager.MakePartiallyOperableAccoutsFullyOperable(s.password)
	s.Require().NoError(err,
		"a cold keypair must be skipped entirely, else its missing master keystore file aborts promotion for every keypair")
	s.Require().Equal([]cryptotypes.Address{partialAcc.Address}, addresses,
		"exactly the regular keypair's partially-operable account must be promoted — cold and fully-operable accounts are untouched")
	s.Require().Equal(filesBefore+1, s.countKeystoreFiles(),
		"the promoted account must gain its own keystore file, and nothing may be written for the cold keypair")
	_, err = s.accManager.loadAccountInternally(partialAcc.Address, s.password)
	s.Require().NoError(err, "the new file must hold the promoted account's own key")
}

func (s *ManagerTestSuite) TestMigrateKeypairToColdWalletUnknownKeyUIDErrors() {
	s.persistence.EXPECT().GetKeypairByKeyUID("unknown-key-uid").Return(nil, accsmanagementtypes.ErrDbKeypairNotFound).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet("unknown-key-uid", s.password, accsmanagementtypes.ColdWalletTypeStatusKeycard, 1)
	s.Require().Error(err, "a non-existent keyUID must surface the persistence error, not panic on the nil keypair")
	s.Require().ErrorIs(err, accsmanagementtypes.ErrDbKeypairNotFound, "the persistence error must be matchable app-side")
}
