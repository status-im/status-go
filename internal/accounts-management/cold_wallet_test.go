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
	s.Require().ErrorIs(err, ErrWrongPasswordProvided(nil), "the typed wrong-password error must be returned for app-side matching")
	s.Require().Equal(filesBefore, s.countKeystoreFiles(),
		"no keystore file may be written when the password was not verified, else accounts get encrypted under an unusable password")
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
	// the empty xpub argument takes the DB-layer preserve branch: the stored xpub must be retained
	s.persistence.EXPECT().UpdateKeypairXPub(coldKp.KeyUID, "", accsmanagementtypes.ColdWalletTypeNone, uint64(7)).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	keyUID, err := s.accManager.MigrateColdWalletKeypairToApp(mnemonic2, s.password, 7)
	s.Require().NoError(err, "migrating a cold keypair back to app with the correct profile password must succeed")
	s.Require().Equal(coldKp.KeyUID, keyUID, "returned keyUID must identify the migrated keypair")
	s.Require().Equal(filesBefore+2, s.countKeystoreFiles(),
		"keystore files must be recreated for the master key and every account path, else accounts stay unusable for signing")

	master2, err := generator.CreateAccountFromMnemonic(mnemonic2, "")
	s.Require().NoError(err)
	restored, err := s.accManager.loadAccountInternally(master2.Address(), s.password)
	s.Require().NoError(err, "restored master keystore file must decrypt with the provided password")
	s.Require().NotNil(restored)
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
	s.Require().NoError(err, "an already-migrated keypair has no keystore files, so switching cold-wallet type must not demand a password")
}

func (s *ManagerTestSuite) TestMigrateColdWalletKeypairToAppRejectsNonColdKeypair() {
	keypair := &accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: s.masterAccount.Address().Hex(),
	}

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	_, err := s.accManager.MigrateColdWalletKeypairToApp(s.mnemonic, s.password, 0)
	s.Require().Error(err, "reverse migration on a keypair that is not a cold wallet must be rejected, else healthy keystore state gets overwritten")
	s.Require().ErrorIs(err, ErrKeypairIsNotColdWallet, "the typed not-cold-wallet error must be returned for app-side matching")
}

func (s *ManagerTestSuite) TestMigrateColdWalletKeypairToAppUnknownKeypairErrors() {
	s.persistence.EXPECT().GetKeypairByKeyUID(s.masterAccount.KeyUID()).Return(nil, accsmanagementtypes.ErrDbKeypairNotFound).Times(1)

	_, err := s.accManager.MigrateColdWalletKeypairToApp(s.mnemonic, s.password, 0)
	s.Require().Error(err, "a mnemonic deriving to an unknown keyUID must surface the persistence error, not panic")
	s.Require().ErrorIs(err, accsmanagementtypes.ErrDbKeypairNotFound, "the persistence error must pass through unchanged for app-side matching")
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
	s.persistence.EXPECT().SaveOrUpdateKeypair(gomock.Any()).Return(nil).Times(1)

	keypair, err := s.accManager.AddKeypairStoredToColdWallet("ledger-kp-uid", "0xmaster-address", "ledger-kp",
		"ledger-wallet-xpub", accsmanagementtypes.ColdWalletTypeLedger, accounts, 9)
	s.Require().NoError(err, "importing a ledger keypair with valid wallet-path accounts must succeed")
	s.Require().Equal(accsmanagementtypes.KeypairTypeSeed, keypair.Type, "cold-wallet imports are persisted as seed keypairs")
	s.Require().Equal("0xmaster-address", keypair.DerivedFrom, "the caller-supplied master address must be stored as DerivedFrom")
	s.Require().Equal("ledger-wallet-xpub", keypair.XPub, "the caller-supplied wallet xpub must be stored for later no-password derivation")
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeLedger, keypair.ColdWallet, "the given cold-wallet type must be persisted")
	s.Require().Equal(uint64(9), keypair.Clock, "the given clock must be persisted")
	s.Require().Equal(int64(4), keypair.Accounts[0].Position, "positions must be assigned sequentially from GetPositionForNextNewAccount")
	s.Require().Equal(int64(5), keypair.Accounts[1].Position, "positions must be assigned sequentially from GetPositionForNextNewAccount")
	s.Require().Equal(accsmanagementtypes.AccountFullyOperable, keypair.Accounts[0].Operable, "cold-wallet accounts must be forced fully operable")
	s.Require().Equal(accsmanagementtypes.AccountFullyOperable, keypair.Accounts[1].Operable, "cold-wallet accounts must be forced fully operable")
	s.Require().Equal(0, s.countKeystoreFiles(), "no keystore file may be written for a cold-wallet keypair import")
}

func (s *ManagerTestSuite) TestAddKeypairStoredToColdWalletRejectsEmptyWalletAccounts() {
	_, err := s.accManager.AddKeypairStoredToColdWallet("some-kp-uid", "0xmaster-address", "kp-name",
		"some-xpub", accsmanagementtypes.ColdWalletTypeStatusKeycard, nil, 1)
	s.Require().Error(err, "a keypair import with zero wallet accounts must be rejected, else later xpub derivation or default-account resolution nil-derefs")
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
	s.Require().Error(err, "re-importing an existing keyUID must be rejected, else SaveOrUpdateKeypair silently clobbers the stored keypair")
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
		"a cold keypair must be skipped entirely, else its missing master keystore file aborts promotion for every keypair at login")
	s.Require().Equal([]cryptotypes.Address{partialAcc.Address}, addresses,
		"exactly the regular keypair's partially-operable account must be promoted — cold and fully-operable accounts are untouched")
	s.Require().Equal(filesBefore+1, s.countKeystoreFiles(),
		"the promoted account must gain its own keystore file, and nothing may be written for the cold keypair")
}

func (s *ManagerTestSuite) TestMigrateKeypairToColdWalletUnknownKeyUIDErrors() {
	s.persistence.EXPECT().GetKeypairByKeyUID("unknown-key-uid").Return(nil, accsmanagementtypes.ErrDbKeypairNotFound).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet("unknown-key-uid", testPassword, accsmanagementtypes.ColdWalletTypeStatusKeycard, 1)
	s.Require().Error(err, "a non-existent keyUID must surface the persistence error, not panic on the nil keypair")
	s.Require().ErrorIs(err, accsmanagementtypes.ErrDbKeypairNotFound, "the persistence error must pass through unchanged for app-side matching")
}
