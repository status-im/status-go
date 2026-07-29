package accountsmanagement

import (
	"errors"
	"os"

	common "github.com/status-im/status-go/internal/accounts-management/common"
	generator "github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/types"
	types2 "github.com/status-im/status-go/internal/crypto/types"
)

func (s *ManagerTestSuite) expectedWalletXPub() string {
	xpub, err := generator.DeriveExtendedPublicKeyAtPath(s.mnemonic, "", common.PathWalletXPub)
	s.Require().NoError(err)
	return xpub
}

func (s *ManagerTestSuite) deriveTestAccountAtPath(path string) *types.Account {
	_, derived, err := generator.CreateAndDeriveAccountsFromMnemonic(s.mnemonic, []string{path}, "")
	s.Require().NoError(err)
	return &types.Account{
		Address:  derived[path].Address(),
		KeyUID:   s.masterAccount.KeyUID(),
		Type:     types.AccountTypeGenerated,
		Path:     path,
		Operable: types.AccountFullyOperable,
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
	s.persistence.EXPECT().SaveOrUpdateAccounts([]*types.Account{acc}, true).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	err := s.accManager.AddAccounts(keypair.KeyUID, []*types.Account{acc}, "")
	s.Require().NoError(err)
	// no password -> the account has no keystore file and is only partially operable
	s.Require().Equal(types.AccountPartiallyOperable, acc.Operable)
	s.Require().Equal(filesBefore, s.countKeystoreFiles())
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordLedgerStylePath() {
	keypair := s.createAndStoreProfileKeypair()

	acc := s.deriveTestAccountAtPath(common.PathWalletXPub + "/5")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().SaveOrUpdateAccounts([]*types.Account{acc}, true).Return(nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*types.Account{acc}, "")
	s.Require().NoError(err)
	s.Require().Equal(types.AccountPartiallyOperable, acc.Operable)
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordWrongAddressRejected() {
	keypair := s.createAndStoreProfileKeypair()

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")
	// address the claimed path doesn't derive to
	acc.Address = types2.HexToAddress("0x000000000000000000000000000000000000dead")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*types.Account{acc}, "")
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
		acc := &types.Account{
			Address: types2.HexToAddress("0x000000000000000000000000000000000000dead"),
			KeyUID:  keypair.KeyUID,
			Type:    types.AccountTypeGenerated,
			Path:    path,
		}

		s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

		err := s.accManager.AddAccounts(keypair.KeyUID, []*types.Account{acc}, "")
		s.Require().Error(err, "path %s", path)
		s.Require().True(errors.Is(err, ErrCannotDeriveAccountFromXPub), "path %s", path)
	}
}

func (s *ManagerTestSuite) TestAddAccountsWithoutPasswordAndWithoutXPubRejected() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.XPub = "" // keypair added before the xpub was tracked

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*types.Account{acc}, "")
	s.Require().Error(err)
	s.Require().True(errors.Is(err, ErrNoPasswordProvidedAndNoXPubStored))
}

func (s *ManagerTestSuite) TestAddAccountsWithPasswordBackfillsXPub() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.XPub = "" // keypair added before the xpub was tracked

	acc := s.deriveTestAccountAtPath(common.PathWalletRoot + "/1")

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().SaveOrUpdateAccounts([]*types.Account{acc}, true).Return(nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, s.expectedWalletXPub(), types.ColdWalletTypeNone, keypair.Clock).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	err := s.accManager.AddAccounts(keypair.KeyUID, []*types.Account{acc}, s.password)
	s.Require().NoError(err)
	// with a password the account gets its own keystore file, stays fully operable and the
	// keypair's missing xpub is backfilled on the way
	s.Require().Equal(types.AccountFullyOperable, acc.Operable)
	s.Require().Equal(filesBefore+1, s.countKeystoreFiles())
	s.Require().Equal(s.expectedWalletXPub(), keypair.XPub)
}

func (s *ManagerTestSuite) TestAddAccountsToKeyKeypairRejected() {
	keypair := &types.Keypair{
		KeyUID: "key-keypair",
		Type:   types.KeypairTypeKey,
	}

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.AddAccounts(keypair.KeyUID, []*types.Account{{KeyUID: keypair.KeyUID}}, "")
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
	s.persistence.EXPECT().GetAccountByAddress(address).Return(&types.Account{
		Address:  address,
		KeyUID:   master2.KeyUID(),
		Path:     accPath,
		Operable: types.AccountPartiallyOperable,
	}, nil).Times(1)
	s.persistence.EXPECT().GetKeypairByKeyUID(master2.KeyUID()).Return(&types.Keypair{
		KeyUID:      master2.KeyUID(),
		Type:        types.KeypairTypeSeed,
		DerivedFrom: master2.Address().Hex(),
	}, nil).Times(1)

	account, err := s.accManager.GetVerifiedWalletAccount(address, s.password)
	s.Require().NoError(err)
	s.Require().Equal(address, account.Address())
}

func (s *ManagerTestSuite) TestBackfillKeypairsXPub() {
	_ = s.createAndStoreProfileKeypair()
	xpub := s.expectedWalletXPub()

	kpRegular := &types.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        types.KeypairTypeProfile,
		DerivedFrom: s.masterAccount.Address().Hex(),
		Clock:       42,
	}
	kpKey := &types.Keypair{KeyUID: "key-kp", Type: types.KeypairTypeKey}
	kpCold := &types.Keypair{KeyUID: "cold-kp", Type: types.KeypairTypeSeed, ColdWallet: types.ColdWalletTypeStatusKeycard}
	kpDone := &types.Keypair{KeyUID: "done-kp", Type: types.KeypairTypeSeed, XPub: "already-set"}
	// keypair with no master keystore file — skipped with a warning
	kpNoKeystoreFile := &types.Keypair{KeyUID: "no-file-kp", Type: types.KeypairTypeSeed,
		DerivedFrom: "0x000000000000000000000000000000000000dead"}

	s.persistence.EXPECT().GetActiveKeypairs().Return([]*types.Keypair{kpRegular, kpKey, kpCold, kpDone, kpNoKeystoreFile}, nil).Times(1)
	// only the regular keypair with a decryptable master keystore file gets backfilled
	s.persistence.EXPECT().UpdateKeypairXPub(kpRegular.KeyUID, xpub, types.ColdWalletTypeNone, uint64(42)).Return(nil).Times(1)

	err := s.accManager.BackfillKeypairsXPub(s.password)
	s.Require().NoError(err)
	s.Require().Equal(xpub, kpRegular.XPub)
	s.Require().Empty(kpNoKeystoreFile.XPub)
}

func (s *ManagerTestSuite) TestBackfillKeypairsXPubReturnsPersistenceError() {
	_ = s.createAndStoreProfileKeypair()

	kpRegular := &types.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        types.KeypairTypeProfile,
		DerivedFrom: s.masterAccount.Address().Hex(),
		Clock:       42,
	}

	s.persistence.EXPECT().GetActiveKeypairs().Return([]*types.Keypair{kpRegular}, nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(kpRegular.KeyUID, s.expectedWalletXPub(), types.ColdWalletTypeNone, uint64(42)).
		Return(errors.New("db failure")).Times(1)

	err := s.accManager.BackfillKeypairsXPub(s.password)
	s.Require().Error(err)
	s.Require().ErrorContains(err, kpRegular.KeyUID)
	s.Require().Empty(kpRegular.XPub)
}

func (s *ManagerTestSuite) TestMigrateKeypairToColdWalletBackfillsXPub() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.XPub = "" // keypair added before the xpub was tracked

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	// the xpub is captured from the master keystore file before the keystore files are deleted
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, s.expectedWalletXPub(), types.ColdWalletTypeNone, keypair.Clock).Return(nil).Times(1)
	s.persistence.EXPECT().MarkKeypairFullyOperable(keypair.KeyUID, uint64(99), true).Return(nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(keypair.KeyUID, "", types.ColdWalletTypeStatusKeycard, uint64(99)).Return(nil).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet(keypair.KeyUID, s.password, types.ColdWalletTypeStatusKeycard, 99)
	s.Require().NoError(err)
	s.Require().Equal(s.expectedWalletXPub(), keypair.XPub)
	// all keystore files of the keypair are deleted
	s.Require().Equal(0, s.countKeystoreFiles())
}
