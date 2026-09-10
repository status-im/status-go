package accountsmanagement

import (
	"errors"
	"os"

	common "github.com/status-im/status-go/internal/accounts-management/common"
	customerrors "github.com/status-im/status-go/internal/accounts-management/errors"
	generator "github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/keystore"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

func (s *ManagerTestSuite) expectedWalletXPub() string {
	xpub, err := generator.DeriveExtendedPublicKeyAtPath(s.mnemonic, "", common.PathWalletXPub)
	s.Require().NoError(err)
	return xpub
}

// storeDistinctSeedKeypair creates a keypair from its own mnemonic and stores its
// master key, so each keypair in a test derives a different xpub. Sharing one
// master would let a backfill read the wrong keypair's file undetected.
func (s *ManagerTestSuite) storeDistinctSeedKeypair(keyUID string, clock uint64, password string) (*accsmanagementtypes.Keypair, string) {
	mnemonic, err := common.CreateRandomMnemonicWithDefaultLength()
	s.Require().NoError(err)
	master, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	s.Require().NoError(err)
	s.Require().NoError(s.accManager.storeToKeystore(master, password))

	xpub, err := generator.DeriveExtendedPublicKeyAtPath(mnemonic, "", common.PathWalletXPub)
	s.Require().NoError(err)

	return &accsmanagementtypes.Keypair{
		KeyUID:      keyUID,
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: master.Address().Hex(),
		Clock:       clock,
	}, xpub
}

func (s *ManagerTestSuite) deriveTestAccountAtPath(path string) *accsmanagementtypes.Account {
	_, derived, err := generator.CreateAndDeriveAccountsFromMnemonic(s.mnemonic, []string{path}, "")
	s.Require().NoError(err)
	return &accsmanagementtypes.Account{
		Address:  derived[path].Address(),
		KeyUID:   s.masterAccount.KeyUID(),
		Type:     accsmanagementtypes.AccountTypeGenerated,
		Path:     path,
		Operable: accsmanagementtypes.AccountFullyOperable,
	}
}

func (s *ManagerTestSuite) countKeystoreFiles() int {
	entries, err := os.ReadDir(s.getKeyDir())
	s.Require().NoError(err)
	return len(entries)
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordDerivesFromXPub() {
	keypair := s.createAndStoreProfileKeypair()
	s.Require().NotEmpty(keypair.XPub)

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().SaveOrUpdateAccounts([]*accsmanagementtypes.Account{acc}, true).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, "")
	s.Require().NoError(err)
	// no password -> the account has no keystore file and is only partially operable
	s.Require().Equal(accsmanagementtypes.AccountPartiallyOperable, acc.Operable)
	s.Require().Equal(filesBefore, s.countKeystoreFiles())
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordLedgerStylePath() {
	keypair := s.createAndStoreProfileKeypair()

	acc := s.deriveTestAccountAtPath(common.PathWalletXPub + "/5")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().SaveOrUpdateAccounts([]*accsmanagementtypes.Account{acc}, true).Return(nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, "")
	s.Require().NoError(err)
	s.Require().Equal(accsmanagementtypes.AccountPartiallyOperable, acc.Operable)
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordWrongAddressRejected() {
	keypair := s.createAndStoreProfileKeypair()

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")
	// address the claimed path doesn't derive to
	acc.Address = cryptotypes.HexToAddress("0x000000000000000000000000000000000000dead")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, "")
	s.Require().Error(err)
	s.Require().True(errors.Is(err, ErrAccountMismatch))
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordNonDerivablePathsRejected() {
	keypair := s.createAndStoreProfileKeypair()

	for _, path := range []string{
		common.PathWalletXPub + "/0'/1", // hardened below the wallet xpub
		"m/44'/61'/0'/0/1",              // out of the wallet xpub tree
		"m",                             // master
	} {
		acc := &accsmanagementtypes.Account{
			Address: cryptotypes.HexToAddress("0x000000000000000000000000000000000000dead"),
			KeyUID:  keypair.KeyUID,
			Type:    accsmanagementtypes.AccountTypeGenerated,
			Path:    path,
		}

		s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

		err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, "")
		s.Require().Error(err, "path %s", path)
		s.Require().True(errors.Is(err, ErrCannotDeriveAccountFromXPub), "path %s", path)
	}
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordAndWithoutXPubRejected() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.XPub = "" // keypair added before the xpub was tracked

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, "")
	s.Require().Error(err)
	s.Require().True(errors.Is(err, ErrNoPasswordProvidedAndNoXPubStored))
}

func (s *ManagerTestSuite) TestAddAccountsWithPasswordBackfillsXPub() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.XPub = "" // keypair added before the xpub was tracked

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().SaveOrUpdateAccounts([]*accsmanagementtypes.Account{acc}, true).Return(nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, s.expectedWalletXPub(), accsmanagementtypes.ColdWalletTypeNone, keypair.Clock).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, s.password)
	s.Require().NoError(err)
	// with a password the account gets its own keystore file, stays fully operable and the
	// keypair's missing xpub is backfilled on the way
	s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable)
	s.Require().Equal(filesBefore+1, s.countKeystoreFiles())
	s.Require().Equal(s.expectedWalletXPub(), keypair.XPub)
}

func (s *ManagerTestSuite) TestAddAccountsToKeyKeypairRejected() {
	keypair := &accsmanagementtypes.Keypair{
		KeyUID: "key-keypair",
		Type:   accsmanagementtypes.KeypairTypeKey,
	}

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{{KeyUID: keypair.KeyUID}}, "")
	s.Require().Error(err)
	s.Require().True(errors.Is(err, ErrCannotAddAccountsToKeypairImportedViaPrivateKey))
}

func (s *ManagerTestSuite) TestGetVerifiedWalletAccountForPartiallyOperableAccountOfNonProfileKeypair() {
	// the profile keypair initializes the keystore
	_ = s.createAndStoreProfileKeypair()

	// a non-profile seed keypair with an account at a Ledger-style path; only the master keystore
	// file is stored — the account itself is partially operable
	mnemonic2, err := common.CreateRandomMnemonicWithDefaultLength()
	s.Require().NoError(err)
	accPath := common.PathWalletXPub + "/5"
	master2, derived2, err := generator.CreateAndDeriveAccountsFromMnemonic(mnemonic2, []string{accPath}, "")
	s.Require().NoError(err)
	s.Require().NoError(s.accManager.storeToKeystore(master2, s.password))

	address := derived2[accPath].Address()

	s.persistence.EXPECT().AddressExists(address).Return(true, nil).Times(1)
	s.persistence.EXPECT().GetAccountByAddress(address).Return(&accsmanagementtypes.Account{
		Address:  address,
		KeyUID:   master2.KeyUID(),
		Path:     accPath,
		Operable: accsmanagementtypes.AccountPartiallyOperable,
	}, nil).Times(1)
	s.persistence.EXPECT().GetKeypairByKeyUID(master2.KeyUID()).Return(&accsmanagementtypes.Keypair{
		KeyUID:      master2.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: master2.Address().Hex(),
	}, nil).Times(1)

	account, err := s.accManager.GetVerifiedWalletAccount(address, s.password)
	s.Require().NoError(err)
	s.Require().Equal(address, account.Address())
}

func (s *ManagerTestSuite) TestGetVerifiedWalletAccountFailsWhenMasterKeystoreFileMissing() {
	s.setupProfileKeystore(false)

	mnemonic2, err := common.CreateRandomMnemonicWithDefaultLength()
	s.Require().NoError(err)
	accPath := common.PathDefaultWalletAccount
	master2, derived2, err := generator.CreateAndDeriveAccountsFromMnemonic(mnemonic2, []string{accPath}, "")
	s.Require().NoError(err)
	address := derived2[accPath].Address()

	s.persistence.EXPECT().AddressExists(address).Return(true, nil).Times(1)
	s.persistence.EXPECT().GetAccountByAddress(address).Return(&accsmanagementtypes.Account{
		Address:  address,
		KeyUID:   master2.KeyUID(),
		Path:     accPath,
		Operable: accsmanagementtypes.AccountFullyOperable,
	}, nil).Times(1)
	s.persistence.EXPECT().GetKeypairByKeyUID(master2.KeyUID()).Return(&accsmanagementtypes.Keypair{
		KeyUID:      master2.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: master2.Address().Hex(),
	}, nil).Times(1)

	account, err := s.accManager.GetVerifiedWalletAccount(address, s.password)
	s.Require().Error(err,
		"with no keystore file for the account or its master key, the software-signing path must fail, not fabricate a key")
	s.Require().ErrorIs(err, keystore.ErrKeystoreFileMissing,
		"the keystore-file-missing error must surface")
	s.Require().Nil(account, "no signing account may be returned")
	var accountsErr *customerrors.AccountsError
	s.Require().ErrorAs(err, &accountsErr)
	s.Require().Equal(master2.Address().Hex(), accountsErr.Context["address"],
		"the missing file must be the master key's: the partial-account fallback was reached")
	s.Require().Equal(0, s.countKeystoreFiles(),
		"the partial-account fallback has no master file to derive from, so nothing may be written")
}

func (s *ManagerTestSuite) TestGetVerifiedWalletAccountWrongPasswordDoesNotFallBackToDerivation() {
	_ = s.createAndStoreProfileKeypair()

	s.persistence.EXPECT().AddressExists(s.walletAddress).Return(true, nil).Times(1)
	// no GetAccountByAddress expectation: a wrong password must never reach the partial-key fallback

	account, err := s.accManager.GetVerifiedWalletAccount(s.walletAddress, "wrong-password")
	s.Require().ErrorIs(err, keystore.ErrIncorrectPasswordProvided)
	s.Require().Nil(account)
	s.Require().Equal(3, s.countKeystoreFiles())
}

func (s *ManagerTestSuite) TestAddAccountsWithWrongPasswordRejectedBeforeSave() {
	keypair := s.createAndStoreProfileKeypair()
	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, "wrong-password")
	s.Require().ErrorIs(err, keystore.ErrIncorrectPasswordProvided)
	s.Require().Equal(3, s.countKeystoreFiles(), "nothing may be written when the password did not open the master file")
}

func (s *ManagerTestSuite) TestAddAccountsRejectsSecondDefaultChatAccountOnProfileKeypair() {
	keypair := s.createAndStoreProfileKeypair()

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")
	acc.Chat = true

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, s.password)
	s.Require().Error(err, "a second default chat account on the profile keypair must be rejected")
	s.Require().ErrorIs(err, ErrCannotAddDefaultChatAccount,
		"the typed cannot-add-default-chat-account error must be returned for app-side matching")
}

func (s *ManagerTestSuite) TestAddAccountsRejectsSecondDefaultWalletAccountOnProfileKeypair() {
	keypair := s.createAndStoreProfileKeypair()

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")
	acc.Wallet = true

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, s.password)
	s.Require().Error(err, "a second default wallet account on the profile keypair must be rejected")
	s.Require().ErrorIs(err, ErrCannotAddDefaultWalletAccount,
		"the typed cannot-add-default-wallet-account error must be returned for app-side matching")
}

func (s *ManagerTestSuite) TestAddAccountsRejectsDuplicateAddress() {
	keypair := s.createAndStoreProfileKeypair()

	dup := s.deriveTestAccountAtPath(common.PathDefaultWalletAccount)

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{dup}, s.password)
	s.Require().Error(err, "an account whose address already exists on the keypair must be rejected")
	s.Require().ErrorIs(err, ErrAccountAlreadyAdded,
		"the typed already-added error must be returned")
}

func (s *ManagerTestSuite) TestBackfillKeypairsXPub() {
	_ = s.createAndStoreProfileKeypair()
	xpub := s.expectedWalletXPub()

	kpRegular := &accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom: s.masterAccount.Address().Hex(),
		Clock:       42,
	}
	kpKey := &accsmanagementtypes.Keypair{KeyUID: "key-kp", Type: accsmanagementtypes.KeypairTypeKey}
	kpCold := &accsmanagementtypes.Keypair{KeyUID: "cold-kp", Type: accsmanagementtypes.KeypairTypeSeed, ColdWallet: accsmanagementtypes.ColdWalletTypeStatusKeycard}
	kpDone := &accsmanagementtypes.Keypair{KeyUID: "done-kp", Type: accsmanagementtypes.KeypairTypeSeed, XPub: "already-set"}
	// keypair with no master keystore file — skipped with a warning
	kpNoKeystoreFile := &accsmanagementtypes.Keypair{KeyUID: "no-file-kp", Type: accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: "0x000000000000000000000000000000000000dead"}

	s.persistence.EXPECT().GetActiveKeypairs().Return([]*accsmanagementtypes.Keypair{kpRegular, kpKey, kpCold, kpDone, kpNoKeystoreFile}, nil).Times(1)
	// only the regular keypair with a decryptable master keystore file gets backfilled
	s.persistence.EXPECT().UpdateKeypairXPub(kpRegular.KeyUID, xpub, accsmanagementtypes.ColdWalletTypeNone, uint64(42)).Return(nil).Times(1)

	err := s.accManager.BackfillKeypairsXPub(s.password)
	s.Require().NoError(err)
	s.Require().Equal(xpub, kpRegular.XPub)
	s.Require().Empty(kpNoKeystoreFile.XPub)
}

func (s *ManagerTestSuite) TestBackfillKeypairsXPubReturnsPersistenceError() {
	_ = s.createAndStoreProfileKeypair()

	kpRegular := &accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom: s.masterAccount.Address().Hex(),
		Clock:       42,
	}

	s.persistence.EXPECT().GetActiveKeypairs().Return([]*accsmanagementtypes.Keypair{kpRegular}, nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(kpRegular.KeyUID, s.expectedWalletXPub(), accsmanagementtypes.ColdWalletTypeNone, uint64(42)).
		Return(errors.New("db failure")).Times(1)

	err := s.accManager.BackfillKeypairsXPub(s.password)
	s.Require().Error(err)
	s.Require().ErrorContains(err, kpRegular.KeyUID)
	s.Require().Empty(kpRegular.XPub)
}

func (s *ManagerTestSuite) TestAddAccountsWithPasswordToColdKeypairIgnoresPasswordAndWritesNoKeystoreFiles() {
	s.setupProfileKeystore(false)
	keypair := &accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		ColdWallet:  accsmanagementtypes.ColdWalletTypeStatusKeycard,
		DerivedFrom: s.masterAccount.Address().Hex(),
		XPub:        s.expectedWalletXPub(),
	}
	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().SaveOrUpdateAccounts([]*accsmanagementtypes.Account{acc}, true).Return(nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, s.password)
	s.Require().NoError(err,
		"a password passed alongside a cold keypair must be ignored, not used to derive from a keystore that does not exist")
	s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable,
		"cold-keypair accounts must stay fully operable, they are never demoted to partially operable")
	s.Require().Equal(0, s.countKeystoreFiles(),
		"no keystore file may be written for a cold keypair, else key material lands on disk for a keycard keypair")
}

func (s *ManagerTestSuite) TestAddAccountsWithPasswordToColdKeypairStillValidatesAgainstXPub() {
	s.setupProfileKeystore(false)
	keypair := &accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		ColdWallet:  accsmanagementtypes.ColdWalletTypeStatusKeycard,
		DerivedFrom: s.masterAccount.Address().Hex(),
		XPub:        s.expectedWalletXPub(),
	}
	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")
	acc.Address = cryptotypes.HexToAddress("0x000000000000000000000000000000000000dead")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, s.password)
	s.Require().Error(err,
		"even with a password, a cold keypair must validate the account address against the stored xpub")
	s.Require().ErrorIs(err, ErrAccountMismatch,
		"the xpub-derived-address mismatch must surface as ErrAccountMismatch")
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordToColdKeypairWithoutXPubAccepted() {
	s.setupProfileKeystore(false)
	keypair := &accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeSeed,
		ColdWallet:  accsmanagementtypes.ColdWalletTypeStatusKeycard,
		DerivedFrom: s.masterAccount.Address().Hex(),
	}
	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().SaveOrUpdateAccounts([]*accsmanagementtypes.Account{acc}, true).Return(nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*accsmanagementtypes.Account{acc}, "")
	s.Require().NoError(err,
		"a cold keypair migrated before xpub tracking has nothing to validate against and must still accept accounts")
	s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable,
		"legacy cold-keypair accounts must be saved fully operable")
	s.Require().Equal(0, s.countKeystoreFiles(),
		"no keystore file may be written on the legacy-compat cold path")
}

func (s *ManagerTestSuite) TestBackfillKeypairsXPubSkipsKeypairOnWrongPassword() {
	s.setupProfileKeystore(false)
	kpOtherPassword, _ := s.storeDistinctSeedKeypair("other-password-kp", 1, "other-password")
	kpHealthy, xpubHealthy := s.storeDistinctSeedKeypair("healthy-kp", 2, s.password)

	s.persistence.EXPECT().GetActiveKeypairs().Return([]*accsmanagementtypes.Keypair{kpOtherPassword, kpHealthy}, nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(kpHealthy.KeyUID, xpubHealthy, accsmanagementtypes.ColdWalletTypeNone, uint64(2)).Return(nil).Times(1)

	err := s.accManager.BackfillKeypairsXPub(s.password)
	s.Require().NoError(err, "a master file that does not open with the login password is skipped, not an error")
	s.Require().Empty(kpOtherPassword.XPub, "no xpub may be recorded for the keypair whose file could not be decrypted")
	s.Require().Equal(xpubHealthy, kpHealthy.XPub, "the keypair listed after the skipped one must still be backfilled")
}

func (s *ManagerTestSuite) TestBackfillKeypairsXPubJoinsErrorsAndContinuesPastFailures() {
	_ = s.createAndStoreProfileKeypair()

	kpFail1, xpub1 := s.storeDistinctSeedKeypair("fail-1", 1, s.password)
	kpFail2, xpub2 := s.storeDistinctSeedKeypair("fail-2", 2, s.password)
	kpHealthy, xpubHealthy := s.storeDistinctSeedKeypair("healthy-kp", 3, s.password)
	s.Require().NotEqual(xpub1, xpub2)
	s.Require().NotEqual(xpub2, xpubHealthy)

	s.persistence.EXPECT().GetActiveKeypairs().Return([]*accsmanagementtypes.Keypair{kpFail1, kpFail2, kpHealthy}, nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(kpFail1.KeyUID, xpub1, accsmanagementtypes.ColdWalletTypeNone, uint64(1)).
		Return(errors.New("db failure 1")).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(kpFail2.KeyUID, xpub2, accsmanagementtypes.ColdWalletTypeNone, uint64(2)).
		Return(errors.New("db failure 2")).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(kpHealthy.KeyUID, xpubHealthy, accsmanagementtypes.ColdWalletTypeNone, uint64(3)).
		Return(nil).Times(1)

	err := s.accManager.BackfillKeypairsXPub(s.password)
	s.Require().Error(err, "failed keypairs must surface in the returned error")
	s.Require().ErrorContains(err, kpFail1.KeyUID, "the joined error must name the first failing keypair")
	s.Require().ErrorContains(err, kpFail2.KeyUID, "the joined error must name the second failing keypair")
	s.Require().Equal(xpubHealthy, kpHealthy.XPub,
		"a healthy keypair listed after failures must still be backfilled")
}

func (s *ManagerTestSuite) TestMigrateKeypairToColdWalletBackfillsXPub() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.XPub = "" // keypair added before the xpub was tracked

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	// the xpub is captured from the master keystore file before the keystore files are deleted
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, s.expectedWalletXPub(), accsmanagementtypes.ColdWalletTypeNone, keypair.Clock).Return(nil).Times(1)
	s.persistence.EXPECT().MarkKeypairFullyOperable(keypair.KeyUID, uint64(99), true).Return(nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, "", accsmanagementtypes.ColdWalletTypeStatusKeycard, uint64(99)).Return(nil).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet(keypair.KeyUID, s.password, accsmanagementtypes.ColdWalletTypeStatusKeycard, 99)
	s.Require().NoError(err)
	s.Require().Equal(s.expectedWalletXPub(), keypair.XPub)
	// all keystore files of the keypair are deleted
	s.Require().Equal(0, s.countKeystoreFiles())
}
