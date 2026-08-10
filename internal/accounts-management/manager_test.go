package accountsmanagement

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	common "github.com/status-im/status-go/internal/accounts-management/common"
	customerrors "github.com/status-im/status-go/internal/accounts-management/errors"
	generator "github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/keystore"
	mock_persistence "github.com/status-im/status-go/internal/accounts-management/mock"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/testutils"
)

const testPassword = "test-password"
const newTestPassword = "new-test-password"

func TestVerifyAccountPassword(t *testing.T) {
	filename := "testdata/test-account1-status-chain.pk"
	account1 := struct {
		KeyUID        string
		WalletAddress string
		ChatAddress   string
		Password      string
	}{
		KeyUID:        "0x0000000000000000000000000000000000000000000000000000000000000001",
		WalletAddress: "0xbF164ca341326a03b547c05B343b2E21eFAe24b9",
		ChatAddress:   "0xbF164ca341326a03b547c05B343b2E21eFAe24b9",
		Password:      "password",
	}

	accManager, err := NewAccountsManager(testutils.MustCreateTestLogger())
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	persistence := mock_persistence.NewMockPersistence(ctrl)
	accManager.SetPersistence(persistence)

	testCases := []struct {
		name             string
		keyUID           string
		address          string
		password         string
		keystoreSet      bool
		importToLocation bool
		expectedError    error
	}{
		{
			"correct address, correct password (decrypt should succeed)",
			account1.KeyUID,
			account1.WalletAddress,
			account1.Password,
			true,
			true,
			nil,
		},
		{
			"correct address, correct password, non-existent key store",
			account1.KeyUID,
			account1.WalletAddress,
			account1.Password,
			false,
			false,
			ErrKeystoreMissing,
		},
		{
			"correct address, correct password, empty key store (pk is not there)",
			account1.KeyUID,
			account1.WalletAddress,
			account1.Password,
			true,
			false,
			keystore.ErrKeystoreFileMissing,
		},
		{
			"wrong address, correct password",
			account1.KeyUID,
			"0x79791d3e8f2daa1f7fec29649d152c0ada3cc535",
			account1.Password,
			true,
			true,
			keystore.ErrKeystoreFileMissing,
		},
		{
			"correct address, wrong password",
			account1.KeyUID,
			account1.WalletAddress,
			"wrong password", // wrong password
			true,
			true,
			keystore.ErrIncorrectPasswordProvided,
		},
	}
	for _, testCase := range testCases {

		rootDataDir := t.TempDir()
		accManager.SetRootDataDir(rootDataDir)
		keystore, err := accManager.createKeystore(testCase.keyUID)
		require.NoError(t, err)

		if testCase.importToLocation {
			err = os.Link(filename, filepath.Join(keystore.KeystorePath(), filepath.Base(filename)))
			require.NoError(t, err)

			// now we need to re-create the keystore in order to make the get-keystore aware of the copied account1
			keystore, err = accManager.createKeystore(testCase.keyUID)
			require.NoError(t, err)
		}

		if testCase.keystoreSet {
			accManager.setKeystore(keystore)
		} else {
			accManager.setKeystore(nil)
		}

		ok, err := accManager.VerifyAccountPassword(cryptotypes.HexToAddress(testCase.address), testCase.password)
		if testCase.expectedError != nil && err != nil {
			if !errors.Is(err, testCase.expectedError) {
				var accountsErr *customerrors.AccountsError
				if errors.As(err, &accountsErr) {
					if testCase.expectedError.Error() == "no key for given address or file" {
						require.Contains(t, accountsErr.Error(), "keystore file is missing")
					} else {
						require.Equal(t, testCase.expectedError.Error(), accountsErr.Error())
					}
				} else {
					require.Equal(t, testCase.expectedError.Error(), err.Error())
				}
			}
		} else if (testCase.expectedError == nil || err == nil) && testCase.expectedError != err {
			require.FailNow(t, fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			require.True(t, ok)
		} else {
			require.False(t, ok)
		}
	}
}

// TestSecretResolver verifies that keystore operations translate the
// client-provided password through the configured secret resolver, and fall
// back to the raw password when the keystore files are still encrypted with
// it (interrupted encryption-scheme migration).
func TestSecretResolver(t *testing.T) {
	// testdata/test-account1-status-chain.pk is encrypted with "password".
	const (
		filename     = "testdata/test-account1-status-chain.pk"
		keyUID       = "0x0000000000000000000000000000000000000000000000000000000000000001"
		address      = "0xbF164ca341326a03b547c05B343b2E21eFAe24b9"
		filePassword = "password"
		clientKEK    = "client-kek"
	)

	newManagerWithTestKey := func(t *testing.T) *AccountsManager {
		accManager, err := NewAccountsManager(testutils.MustCreateTestLogger())
		require.NoError(t, err)
		accManager.SetRootDataDir(t.TempDir())
		ks, err := accManager.createKeystore(keyUID)
		require.NoError(t, err)
		require.NoError(t, os.Link(filename, filepath.Join(ks.KeystorePath(), filepath.Base(filename))))
		ks, err = accManager.createKeystore(keyUID)
		require.NoError(t, err)
		accManager.setKeystore(ks)
		return accManager
	}

	t.Run("resolved secret decrypts the keystore", func(t *testing.T) {
		accManager := newManagerWithTestKey(t)
		// Migrated profile in steady state: the resolver maps the client KEK
		// to the secret the keystore files are encrypted with.
		accManager.SetSecretResolver(func(password string) (string, error) {
			require.Equal(t, clientKEK, password)
			return filePassword, nil
		})

		ok, err := accManager.VerifyAccountPassword(cryptotypes.HexToAddress(address), clientKEK)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("fallback to raw password after interrupted migration", func(t *testing.T) {
		accManager := newManagerWithTestKey(t)
		// Interrupted migration: the resolver yields a DEK, but the keystore
		// files are still encrypted with the raw password.
		accManager.SetSecretResolver(func(password string) (string, error) {
			require.Equal(t, filePassword, password)
			return "dek-the-keystore-does-not-use-yet", nil
		})

		ok, err := accManager.VerifyAccountPassword(cryptotypes.HexToAddress(address), filePassword)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("resolver rejection fails before touching the keystore", func(t *testing.T) {
		accManager := newManagerWithTestKey(t)
		resolverErr := errors.New("invalid key-encryption key")
		accManager.SetSecretResolver(func(string) (string, error) {
			return "", resolverErr
		})

		ok, err := accManager.VerifyAccountPassword(cryptotypes.HexToAddress(address), filePassword)
		require.ErrorIs(t, err, resolverErr)
		require.False(t, ok)
	})

	t.Run("wrong secret with wrong fallback still fails", func(t *testing.T) {
		accManager := newManagerWithTestKey(t)
		accManager.SetSecretResolver(func(string) (string, error) {
			return "some-dek", nil
		})

		ok, err := accManager.VerifyAccountPassword(cryptotypes.HexToAddress(address), "wrong-password")
		require.Error(t, err)
		require.False(t, ok)
	})
}

// TestVerifyAccountPasswordWithAccountBeforeEIP55 verifies if VerifyAccountPassword
// can handle accounts before introduction of EIP55.
func TestVerifyAccountPasswordWithAccountBeforeEIP55(t *testing.T) {
	testKeyFile := "testdata/test-account3-before-eip55.pk"
	account3 := struct {
		KeyUID        string
		WalletAddress string
		ChatAddress   string
		Password      string
	}{
		KeyUID:        "0x2084cb9965e28dacf4d4ef4ea900c4d30144f1ce4729c00e46f5dfb509b91b08",
		WalletAddress: "0x3ad34e698d4806afd08b359b920f5c6b62b68ee4",
		ChatAddress:   "0x3ad34e698d4806afd08b359b920f5c6b62b68ee4",
		Password:      "password",
	}

	rootDataDir := t.TempDir()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accManager, err := NewAccountsManager(testutils.MustCreateTestLogger())
	require.NoError(t, err)

	persistence := mock_persistence.NewMockPersistence(ctrl)
	accManager.SetPersistence(persistence)

	accManager.SetRootDataDir(rootDataDir)

	keystore, err := accManager.createKeystore(account3.KeyUID)
	require.NoError(t, err)

	// Copy file to keystore directory (simplest way is to make a link)
	err = os.Link(testKeyFile, filepath.Join(keystore.KeystorePath(), filepath.Base(testKeyFile)))
	require.NoError(t, err)

	// now we need to reload the keystore (re-create it) in order to make the get-keystore aware of the copied account
	err = accManager.ReloadKeystore()
	require.Error(t, err)
	require.Equal(t, ErrNoAccountSelected, err)

	persistence.EXPECT().GetProfileKeypair().Return(
		&accsmanagementtypes.Keypair{
			KeyUID: account3.KeyUID,
		},
		nil,
	).Times(1)

	// Set the chat account, this will create a new keystore
	err = accManager.SetChatAccount(cryptotypes.HexToAddress(account3.ChatAddress), account3.Password, nil)
	require.NoError(t, err)

	address := cryptotypes.HexToAddress(account3.ChatAddress)
	ok, err := accManager.VerifyAccountPassword(address, account3.Password)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite
	testAccount
	accManager  *AccountsManager
	persistence *mock_persistence.MockPersistence
	rootDataDir string
}

type testAccount struct {
	password      string
	walletAddress cryptotypes.Address
	walletPubKey  string
	chatAddress   cryptotypes.Address
	chatPubKey    string
	mnemonic      string
	masterAccount *generator.Account
}

// SetupTest is used here for reinitializing the mock before every
// test function to avoid faulty execution.
func (s *ManagerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	var err error
	s.accManager, err = NewAccountsManager(testutils.MustCreateTestLogger())
	s.Require().NoError(err)

	s.persistence = mock_persistence.NewMockPersistence(ctrl)
	s.accManager.SetPersistence(s.persistence)

	s.rootDataDir = s.T().TempDir()
	s.accManager.SetRootDataDir(s.rootDataDir)

	// Initial test - create test account
	mnemonic, err := common.CreateRandomMnemonicWithDefaultLength()
	s.Require().NoError(err)

	paths := []string{common.PathEIP1581Chat, common.PathDefaultWalletAccount}
	genAcc, derivedAccounts, err := generator.CreateAndDeriveAccountsFromMnemonic(mnemonic, paths, "")
	s.Require().NoError(err)

	s.testAccount = testAccount{
		testPassword,
		derivedAccounts[common.PathDefaultWalletAccount].Address(),
		derivedAccounts[common.PathDefaultWalletAccount].PublicKeyHex(),
		derivedAccounts[common.PathEIP1581Chat].Address(),
		derivedAccounts[common.PathEIP1581Chat].PublicKeyHex(),
		mnemonic,
		genAcc,
	}
}

func (s *ManagerTestSuite) getKeyDir() string {
	return fmt.Sprintf("%s/keystore/%s", s.rootDataDir, s.masterAccount.KeyUID())
}

func (s *ManagerTestSuite) createAndStoreProfileKeypair() *accsmanagementtypes.Keypair {
	s.persistence.EXPECT().GetKeypairByKeyUID(s.masterAccount.KeyUID()).Return(
		nil, accsmanagementtypes.ErrDbKeypairNotFound,
	).Times(1)

	s.persistence.EXPECT().GetPositionForNextNewAccount().Return(int64(0), nil).Times(1)

	s.persistence.EXPECT().SaveOrUpdateKeypair(gomock.Any()).Return(nil).Times(1)

	s.persistence.EXPECT().GetProfileKeypair().Return(
		&accsmanagementtypes.Keypair{
			KeyUID: s.masterAccount.KeyUID(),
		},
		nil,
	).Times(1)

	walletAccount := &accsmanagementtypes.AccountCreationDetails{
		Path: common.PathDefaultWalletAccount,
	}

	keypair, err := s.accManager.CreateKeypairFromMnemonicAndStore(s.mnemonic, s.password, "kp-name", accsmanagementtypes.ColdWalletTypeNone, walletAccount, true, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(s.mnemonic)
	s.Require().NotNil(keypair)
	s.Require().Equal(s.masterAccount.KeyUID(), keypair.KeyUID)
	s.Require().Equal(s.masterAccount.Address().Hex(), keypair.DerivedFrom)
	s.Require().Len(keypair.Accounts, 2)

	chatAccountOk := false
	walletAccountOk := false
	for _, kpAcc := range keypair.Accounts {
		if kpAcc.Chat {
			chatAccountOk = kpAcc.Path == common.PathEIP1581Chat &&
				kpAcc.Address == s.chatAddress &&
				bytes.Equal(kpAcc.PublicKey, cryptotypes.Hex2Bytes(s.chatPubKey)) &&
				kpAcc.KeyUID == keypair.KeyUID &&
				!kpAcc.Removed &&
				kpAcc.Clock == 0 &&
				!kpAcc.Wallet &&
				kpAcc.AddressWasNotShown &&
				kpAcc.Position == -1 &&
				kpAcc.Operable == accsmanagementtypes.AccountFullyOperable
		}
		if kpAcc.Wallet {
			walletAccountOk = kpAcc.Path == common.PathDefaultWalletAccount &&
				kpAcc.Address == s.walletAddress &&
				bytes.Equal(kpAcc.PublicKey, cryptotypes.Hex2Bytes(s.walletPubKey)) &&
				kpAcc.KeyUID == keypair.KeyUID &&
				!kpAcc.Removed &&
				kpAcc.Clock == 0 &&
				!kpAcc.Chat &&
				kpAcc.AddressWasNotShown &&
				kpAcc.Position == 0 &&
				kpAcc.Operable == accsmanagementtypes.AccountFullyOperable
		}
	}
	s.Require().True(chatAccountOk)
	s.Require().True(walletAccountOk)

	return keypair
}

func (s *ManagerTestSuite) TestRecoverAccount() {
	s.createAndStoreProfileKeypair()
}

func (s *ManagerTestSuite) TestSetChatAccountSuccess() {
	s.testSetChatAccount(s.testAccount.chatAddress, s.testAccount.password, nil)
}

func (s *ManagerTestSuite) TestSetChatAccountWrongAddress() {
	s.testSetChatAccount(cryptotypes.HexToAddress("0x0000000000000000000000000000000000000001"), s.testAccount.password, keystore.ErrKeystoreFileMissing)
}

func (s *ManagerTestSuite) TestSetChatAccountWrongPassword() {
	s.testSetChatAccount(s.testAccount.chatAddress, "wrong", keystore.ErrIncorrectPasswordProvided)
}

func (s *ManagerTestSuite) testSetChatAccount(chat cryptotypes.Address, password string, expErr error) {
	s.createAndStoreProfileKeypair()
	s.accManager.setChatAccountAndProfileKeyUID(nil, "") // clear the chat account set by `createAndStoreProfileKeypair`

	s.persistence.EXPECT().GetProfileKeypair().Return(
		&accsmanagementtypes.Keypair{
			KeyUID: s.masterAccount.KeyUID(),
		},
		nil,
	).Times(1)

	err := s.accManager.SetChatAccount(chat, password, nil)
	if expErr != nil {
		if !errors.Is(err, expErr) {
			var accountsErr *customerrors.AccountsError
			if errors.As(err, &accountsErr) {
				if expErr.Error() == "no key for given address or file" {
					s.Require().Contains(accountsErr.Error(), "keystore file is missing")
				} else {
					s.Require().Equal(expErr.Error(), accountsErr.Error())
				}
			} else {
				s.Require().Equal(expErr, err)
			}
		}
	} else {
		s.Require().NoError(err)
	}

	selectedChatAccount, err := s.accManager.SelectedChatAccount()

	if expErr == nil {
		s.Require().NoError(err)
		s.Equal(chat, crypto.PubkeyToAddress(selectedChatAccount.PrivateKey().PublicKey))
		s.Require().NotNil(s.accManager.keystore)
		s.Equal(s.getKeyDir(), s.accManager.keystore.KeystorePath())
	} else {
		s.Nil(selectedChatAccount)
		s.Equal(err, ErrNoAccountSelected)
	}

	s.accManager.Logout()
}

func (s *ManagerTestSuite) TestSetChatAccountForExistingProfile() {
	s.createAndStoreProfileKeypair()
	s.accManager.setChatAccountAndProfileKeyUID(nil, "") // clear the chat account set by `createAndStoreProfileKeypair`

	genAcc, err := generator.CreateAccountFromMnemonic(s.mnemonic, "")
	s.Require().NoError(err)

	s.persistence.EXPECT().GetProfileKeypair().Return(
		&accsmanagementtypes.Keypair{
			KeyUID: s.masterAccount.KeyUID(),
		},
		nil,
	).Times(1)

	s.Require().NoError(s.accManager.SetChatAccount(genAcc.Address(), s.password, nil))
	selectedChatAccount, err := s.accManager.SelectedChatAccount()
	s.Require().NoError(err)
	s.Require().NotNil(selectedChatAccount)
	s.Equal(genAcc.PrivateKeyHex(), selectedChatAccount.PrivateKeyHex())
	s.Equal(genAcc.Address(), selectedChatAccount.Address())
	s.Nil(selectedChatAccount.ExtendedKey())
}

func (s *ManagerTestSuite) TestSetChatAccountWithProfileKeypairSkipsPersistenceLookup() {
	profileKeypair := s.createAndStoreProfileKeypair()
	s.accManager.setChatAccountAndProfileKeyUID(nil, "")

	s.Require().NoError(s.accManager.SetChatAccountWithProfileKeypair(
		s.chatAddress,
		s.password,
		nil,
		profileKeypair,
	))

	selectedChatAccount, err := s.accManager.SelectedChatAccount()
	s.Require().NoError(err)
	s.Equal(s.chatAddress, selectedChatAccount.Address())
	s.Nil(selectedChatAccount.ExtendedKey())
	s.Equal(profileKeypair.KeyUID, s.accManager.profileKeyUID)
}

func (s *ManagerTestSuite) TestLogout() {
	s.accManager.Logout()
	s.Nil(s.accManager.selectedChatAccount)
	s.Nil(s.accManager.keystore)
	s.Nil(s.accManager.persistence)
	s.Empty(s.accManager.rootDataDir)
}

// TestAccounts tests cases for (*Manager).Accounts.
func (s *ManagerTestSuite) TestAccounts() {
	s.persistence.EXPECT().GetProfileKeypair().Return(
		&accsmanagementtypes.Keypair{
			KeyUID: s.masterAccount.KeyUID(),
		},
		nil,
	).Times(1)

	// Select the test account, when the profile keypair is not stored
	err := s.accManager.SetChatAccount(s.chatAddress, s.password, nil)
	s.Require().Error(err)
	if !errors.Is(err, keystore.ErrKeystoreFileMissing) {
		var accountsErr *customerrors.AccountsError
		if errors.As(err, &accountsErr) {
			s.Contains(accountsErr.Error(), "keystore file is missing")
		} else {
			s.Equal(keystore.ErrKeystoreFileMissing, err)
		}
	}

	s.createAndStoreProfileKeypair()

	selectedChatAccount, err := s.accManager.SelectedChatAccount()
	s.Require().NoError(err)
	s.Equal(s.chatAddress, selectedChatAccount.Address())

	err = s.accManager.SetChatAccount(s.chatAddress, s.password, nil)
	s.Require().NoError(err)

	// Success
	accs, err := s.accManager.Accounts()
	s.NoError(err)
	s.Len(accs, 3)

	checkAccount := func(address cryptotypes.Address) bool {
		return address == s.chatAddress || address == s.walletAddress || address == s.masterAccount.Address()
	}
	s.True(checkAccount(accs[0]))
	s.True(checkAccount(accs[1]))
	s.True(checkAccount(accs[2]))
}

func (s *ManagerTestSuite) TestAddressToAccountSuccess() {
	s.testAddressToAccount(s.walletAddress, s.password, nil)
}

func (s *ManagerTestSuite) TestAddressToAccountWrongAddress() {
	s.testAddressToAccount(cryptotypes.HexToAddress("0x0001"), s.password, keystore.ErrKeystoreFileMissing)
}

func (s *ManagerTestSuite) TestAddressToAccountWrongPassword() {
	s.testAddressToAccount(s.walletAddress, "wrong", keystore.ErrIncorrectPasswordProvided)
}

func (s *ManagerTestSuite) testAddressToAccount(wallet cryptotypes.Address, password string, expErr error) {
	s.createAndStoreProfileKeypair()

	key, err := s.accManager.LoadAccount(wallet, password)
	if expErr != nil {
		if !errors.Is(err, expErr) {
			var accountsErr *customerrors.AccountsError
			if errors.As(err, &accountsErr) {
				s.Equal(expErr.Error(), accountsErr.Error())
			} else {
				s.Equal(expErr, err)
			}
		}
	} else {
		s.Require().NoError(err)
		s.Require().NotNil(key)
		s.Equal(wallet, key.Address())
	}
}

func (s *ManagerTestSuite) TestMigrateKeyStoreDir() {
	s.createAndStoreProfileKeypair()

	oldKeyDir := s.getKeyDir()
	newKeyDir := filepath.Join(oldKeyDir, "new_dir")
	err := os.Mkdir(newKeyDir, 0777)
	s.Require().NoError(err)

	files, _ := os.ReadDir(newKeyDir)
	s.Equal(0, len(files))

	err = s.accManager.MigrateKeyStoreDir(newKeyDir)
	s.Require().NoError(err)

	files, _ = os.ReadDir(newKeyDir)
	s.Equal(3, len(files))
}

func (s *ManagerTestSuite) TestReEncryptKeyStoreDir() {
	keypair := s.createAndStoreProfileKeypair()

	err := s.accManager.ReEncryptKeyStoreDir(testPassword, newTestPassword)
	s.Require().NoError(err)

	accountsToCheck := []string{keypair.DerivedFrom}
	for _, acc := range keypair.Accounts {
		accountsToCheck = append(accountsToCheck, acc.Address.Hex())
	}

	for _, acc := range accountsToCheck {
		account, err := s.accManager.LoadAccount(cryptotypes.HexToAddress(acc), testPassword)
		s.Require().Error(err)
		s.Require().Nil(account)

		account, err = s.accManager.LoadAccount(cryptotypes.HexToAddress(acc), newTestPassword)
		s.Require().NoError(err)
		s.Require().NotNil(account)
	}
}

func (s *ManagerTestSuite) TestReEncryptKeyStoreDirSkipsFilesystemMetadataFiles() {
	keypair := s.createAndStoreProfileKeypair()

	dsStorePath := filepath.Join(s.getKeyDir(), ".DS_Store")
	s.Require().NoError(os.WriteFile(dsStorePath, []byte{0x00, 0x00, 0x00, 0x01}, 0600))

	err := s.accManager.ReEncryptKeyStoreDir(testPassword, newTestPassword)
	s.Require().NoError(err)

	_, err = os.Stat(dsStorePath)
	s.Require().True(os.IsNotExist(err))

	accountsToCheck := []string{keypair.DerivedFrom}
	for _, acc := range keypair.Accounts {
		accountsToCheck = append(accountsToCheck, acc.Address.Hex())
	}

	for _, acc := range accountsToCheck {
		account, err := s.accManager.LoadAccount(cryptotypes.HexToAddress(acc), newTestPassword)
		s.Require().NoError(err)
		s.Require().NotNil(account)
	}
}

func (s *ManagerTestSuite) TestDeleteAccount() {
	keypair := s.createAndStoreProfileKeypair()

	walletAccount := keypair.Accounts[1]
	walletAccount.Wallet = false

	// check if keystore file exists
	account, err := s.accManager.LoadAccount(walletAccount.Address, testPassword)
	s.Require().NoError(err)
	s.Require().NotNil(account)

	s.persistence.EXPECT().GetAccountByAddress(s.walletAddress).Return(
		walletAccount,
		nil,
	).Times(2)

	s.persistence.EXPECT().GetKeypairByKeyUID(walletAccount.KeyUID).Return(
		keypair,
		nil,
	).Times(1)

	s.persistence.EXPECT().RemoveAccount(walletAccount.Address, uint64(0)).Return(nil).Times(1)

	acc, err := s.accManager.DeleteAccount(s.walletAddress, s.password, 0)
	s.Require().NoError(err)
	s.Require().NotNil(acc)
	s.Require().Equal(walletAccount.Address.Hex(), acc.Address.Hex())

	// check if keystore file exists
	account, err = s.accManager.LoadAccount(walletAccount.Address, testPassword)
	s.Require().Error(err)
	s.Require().Nil(account)

	files, _ := os.ReadDir(s.getKeyDir())
	s.Equal(2, len(files))
}

func (s *ManagerTestSuite) TestDeleteKeypair() {
	keypair := s.createAndStoreProfileKeypair()

	keypair.Type = accsmanagementtypes.KeypairTypeSeed

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(
		keypair,
		nil,
	).Times(1)

	s.persistence.EXPECT().RemoveKeypair(keypair.KeyUID, uint64(0)).Return(nil).Times(1)

	deletedKp, err := s.accManager.DeleteKeypair(keypair.KeyUID, s.password, 0)
	s.Require().NoError(err)
	s.Require().NotNil(deletedKp)
	s.Require().Equal(keypair.KeyUID, deletedKp.KeyUID)

	files, _ := os.ReadDir(s.getKeyDir())
	s.Equal(0, len(files))
}

func (s *ManagerTestSuite) TestDeleteProfileKeypairRejected() {
	keypair := s.createAndStoreProfileKeypair()

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	_, err := s.accManager.DeleteKeypair(keypair.KeyUID, s.password, 0)
	s.Require().Error(err, "the profile keypair must never be deletable through the generic keypair-removal path")
	s.Require().ErrorIs(err, ErrCannotRemoveProfileKeypair,
		"the typed cannot-remove-profile-keypair error must be returned for app-side matching")
	s.Require().Equal(3, s.countKeystoreFiles(),
		"chat identity and login keystore files must remain untouched after the rejected deletion")
}

func (s *ManagerTestSuite) TestDeleteLastAccountOfSeedKeypairAlsoDeletesMasterKeystoreFile() {
	keypair := s.createAndStoreProfileKeypair()

	walletAccount := keypair.Accounts[1]
	walletAccount.Wallet = false

	// a non-profile seed keypair whose ONLY account is the wallet account — deleting it
	// must also remove the master keystore file
	kpSingleAccount := &accsmanagementtypes.Keypair{
		KeyUID:      keypair.KeyUID,
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: keypair.DerivedFrom,
		Accounts:    []*accsmanagementtypes.Account{walletAccount},
	}

	s.persistence.EXPECT().GetAccountByAddress(s.walletAddress).Return(walletAccount, nil).Times(2)
	s.persistence.EXPECT().GetKeypairByKeyUID(walletAccount.KeyUID).Return(kpSingleAccount, nil).Times(1)
	s.persistence.EXPECT().RemoveAccount(walletAccount.Address, uint64(0)).Return(nil).Times(1)

	acc, err := s.accManager.DeleteAccount(s.walletAddress, s.password, 0)
	s.Require().NoError(err, "deleting the sole account of a non-cold seed keypair must succeed")
	s.Require().NotNil(acc)

	_, err = s.accManager.loadAccountInternally(s.masterAccount.Address(), s.password)
	s.Require().Error(err,
		"the master keystore file must be deleted with the keypair's last account, else decryptable master key material is orphaned on disk")
	s.Require().Equal(1, s.countKeystoreFiles(),
		"only the chat account file may remain: the account file and the master keystore file must both be gone")
}

func (s *ManagerTestSuite) TestCleanKeystoreFilesRequiresPassword() {
	err := s.accManager.CleanKeystoreFiles("")
	s.Require().Error(err,
		"the RPC-exposed destructive cleanup must reject an empty password before touching any keypair")
	s.Require().ErrorIs(err, ErrNoPasswordProvided,
		"the typed no-password error must be returned for app-side matching")
}

func (s *ManagerTestSuite) TestCleanKeystoreFilesWrongPasswordLeavesFilesIntact() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.Type = accsmanagementtypes.KeypairTypeSeed
	keypair.Removed = true

	s.persistence.EXPECT().GetAllKeypairs().Return([]*accsmanagementtypes.Keypair{keypair}, nil).Times(1)

	err := s.accManager.CleanKeystoreFiles("wrong-password")
	s.Require().Error(err,
		"cleanup with a password that fails keystore decryption must error, not silently pretend the files were removed")
	s.Require().ErrorIs(err, keystore.ErrIncorrectPasswordProvided,
		"the keystore wrong-password error must surface so the caller knows nothing was deleted")
	s.Require().Equal(3, s.countKeystoreFiles(),
		"every keystore file must survive a cleanup attempted with the wrong password")
}

func (s *ManagerTestSuite) TestCreateKeypairSurfacesTypedKeystoreDirectoryError() {
	if os.Geteuid() == 0 {
		s.T().Skip("chmod-based write denial does not apply to root")
	}
	s.Require().NoError(os.Chmod(s.rootDataDir, 0o555))
	defer func() {
		s.Require().NoError(os.Chmod(s.rootDataDir, 0o755))
	}()

	s.persistence.EXPECT().GetKeypairByKeyUID(s.masterAccount.KeyUID()).Return(
		nil, accsmanagementtypes.ErrDbKeypairNotFound,
	).Times(1)

	walletAccount := &accsmanagementtypes.AccountCreationDetails{
		Path: common.PathDefaultWalletAccount,
	}

	_, err := s.accManager.CreateKeypairFromMnemonicAndStore(
		s.mnemonic, s.password, "kp-name", accsmanagementtypes.ColdWalletTypeNone, walletAccount, true, 0)
	s.Require().Error(err, "keystore directory creation failure must fail the keypair creation")
	s.Require().ErrorIs(err, ErrKeystoreDirectoryError(nil),
		"the failure must surface as the typed keystore-directory AccountsError, not a raw os error, so app-side error handling can match it")
}

func (s *ManagerTestSuite) TestDeleteAccountOfColdWalletKeypairWithoutPassword() {
	acc := s.deriveTestAccountAtPath(common.PathDefaultWalletAccount)
	keypair := &accsmanagementtypes.Keypair{
		KeyUID:     s.masterAccount.KeyUID(),
		Type:       accsmanagementtypes.KeypairTypeSeed,
		ColdWallet: accsmanagementtypes.ColdWalletTypeStatusKeycard,
		Accounts:   []*accsmanagementtypes.Account{acc},
	}

	s.persistence.EXPECT().GetAccountByAddress(acc.Address).Return(acc, nil).Times(2)
	s.persistence.EXPECT().GetKeypairByKeyUID(acc.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().RemoveAccount(acc.Address, uint64(0)).Return(nil).Times(1)

	deletedAcc, err := s.accManager.DeleteAccount(acc.Address, "", 0)
	s.Require().NoError(err)
	s.Require().NotNil(deletedAcc)
	s.Require().Equal(acc.Address, deletedAcc.Address)
}

func (s *ManagerTestSuite) TestDeleteColdWalletKeypairWithoutPassword() {
	keypair := &accsmanagementtypes.Keypair{
		KeyUID:     s.masterAccount.KeyUID(),
		Type:       accsmanagementtypes.KeypairTypeSeed,
		ColdWallet: accsmanagementtypes.ColdWalletTypeStatusKeycard,
	}

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)
	s.persistence.EXPECT().RemoveKeypair(keypair.KeyUID, uint64(0)).Return(nil).Times(1)

	deletedKp, err := s.accManager.DeleteKeypair(keypair.KeyUID, "", 0)
	s.Require().NoError(err)
	s.Require().NotNil(deletedKp)
	s.Require().Equal(keypair.KeyUID, deletedKp.KeyUID)
}

func (s *ManagerTestSuite) TestDeleteRegularKeypairWithoutPasswordRejected() {
	keypair := &accsmanagementtypes.Keypair{
		KeyUID: s.masterAccount.KeyUID(),
		Type:   accsmanagementtypes.KeypairTypeSeed,
	}

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	_, err := s.accManager.DeleteKeypair(keypair.KeyUID, "", 0)
	s.Require().Error(err)
	s.Require().True(errors.Is(err, ErrNoPasswordProvided))
}

// Regression: CreateKeypairFromMnemonicAndStore documents that accounts are
// stored to the keystore only when the keypair is not a cold wallet / keycard,
// but the implementation always calls storeKeystoreFilesForAccounts.
func (s *ManagerTestSuite) TestCreateKeypairFromMnemonicAndStoreDoesNotWriteKeystoreWhenCold() {
	s.persistence.EXPECT().GetKeypairByKeyUID(s.masterAccount.KeyUID()).Return(
		nil, accsmanagementtypes.ErrDbKeypairNotFound,
	).Times(1)

	s.persistence.EXPECT().GetPositionForNextNewAccount().Return(int64(0), nil).Times(1)

	s.persistence.EXPECT().SaveOrUpdateKeypair(gomock.Any()).DoAndReturn(
		func(kp *accsmanagementtypes.Keypair) error {
			s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, kp.ColdWallet)
			s.Require().Equal(s.expectedWalletXPub(), kp.XPub,
				"a cold-created keypair must still store the wallet xpub, else every later no-password AddAccounts is accepted with no address validation")
			return nil
		},
	).Times(1)

	s.persistence.EXPECT().GetProfileKeypair().Return(
		&accsmanagementtypes.Keypair{
			KeyUID: s.masterAccount.KeyUID(),
		},
		nil,
	).Times(1)

	walletAccount := &accsmanagementtypes.AccountCreationDetails{
		Path: common.PathDefaultWalletAccount,
	}

	keypair, err := s.accManager.CreateKeypairFromMnemonicAndStore(
		s.mnemonic, s.password, "kp-name", accsmanagementtypes.ColdWalletTypeStatusKeycard, walletAccount, true, 0)
	s.Require().NoError(err)
	s.Require().NotNil(keypair)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, keypair.ColdWallet)

	files, err := os.ReadDir(s.getKeyDir())
	s.Require().NoError(err)
	s.Require().Empty(files, "cold wallet keypair must not persist private keys in the local keystore")
}

// Regression: MigrateKeypairToColdWallet must refuse an empty password instead of
// silently flipping cold_wallet while leaving keystore files on disk.
func (s *ManagerTestSuite) TestMigrateKeypairToColdWalletRequiresPassword() {
	keypair := s.createAndStoreProfileKeypair()
	keypair.Type = accsmanagementtypes.KeypairTypeSeed

	s.persistence.EXPECT().GetKeypairByKeyUID(keypair.KeyUID).Return(keypair, nil).Times(1)

	err := s.accManager.MigrateKeypairToColdWallet(keypair.KeyUID, "", accsmanagementtypes.ColdWalletTypeStatusKeycard, 1)
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrNoPasswordProvided)

	files, err := os.ReadDir(s.getKeyDir())
	s.Require().NoError(err)
	s.Require().Len(files, 3, "keystore files must remain until a password-backed wipe runs")
}

func (s *ManagerTestSuite) TestCleanKeystoreFiles() {
	testCases := []struct {
		name                     string
		keypairRemoved           bool
		keypairMigratedToKeycard bool
		oneAccountRemoved        bool
		allAccountsRemoved       bool
	}{
		{
			"clean keystore files for removed keypair",
			true,
			false,
			false,
			false,
		},
		{
			"clean keystore files for migrated to keycard keypair",
			false,
			true,
			false,
			false,
		},
		{
			"clean keystore files for not removed keypair but one account removed",
			false,
			false,
			true,
			false,
		},
		{
			"clean keystore files for not removed keypair but all accounts removed",
			false,
			false,
			false,
			true,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {

			keypair := s.createAndStoreProfileKeypair()

			files, _ := os.ReadDir(s.getKeyDir())
			s.Equal(3, len(files))

			keypair.Type = accsmanagementtypes.KeypairTypeSeed
			keypair.Removed = testCase.keypairRemoved

			if testCase.keypairMigratedToKeycard {
				keypair.ColdWallet = accsmanagementtypes.ColdWalletTypeStatusKeycard
			}

			if testCase.oneAccountRemoved {
				keypair.Accounts[0].Removed = true
			}

			if testCase.allAccountsRemoved {
				keypair.Accounts[0].Removed = true
				keypair.Accounts[1].Removed = true
			}

			s.persistence.EXPECT().GetAllKeypairs().Return(
				[]*accsmanagementtypes.Keypair{keypair},
				nil,
			).Times(1)

			err := s.accManager.CleanKeystoreFiles(s.password)
			s.Require().NoError(err)

			files, _ = os.ReadDir(s.getKeyDir())
			if testCase.keypairRemoved || testCase.keypairMigratedToKeycard {
				s.Equal(0, len(files))
			} else if testCase.oneAccountRemoved {
				s.Equal(2, len(files))
			} else if testCase.allAccountsRemoved {
				s.Equal(1, len(files)) // only the master account is left
			}
		})
	}
}

// When the profile keypair is itself on a cold wallet there is no profile master keystore
// file to verify the password against, so verification is deliberately skipped and the
// migration proceeds with the password accepted verbatim.
func (s *ManagerTestSuite) TestMigrateColdWalletKeypairToAppSkipsPasswordCheckWhenProfileKeypairIsCold() {
	s.setupProfileKeystore(false)
	mnemonic2, coldKp := s.createColdSeedKeypair()

	s.persistence.EXPECT().GetKeypairByKeyUID(coldKp.KeyUID).Return(coldKp, nil).Times(1)
	// profile keypair is cold and its DerivedFrom has no keystore file on disk — any
	// password verification attempt against it would fail
	s.persistence.EXPECT().GetProfileKeypair().Return(&accsmanagementtypes.Keypair{
		KeyUID:      s.masterAccount.KeyUID(),
		Type:        accsmanagementtypes.KeypairTypeProfile,
		ColdWallet:  accsmanagementtypes.ColdWalletTypeStatusKeycard,
		DerivedFrom: "0x000000000000000000000000000000000000dead",
	}, nil).Times(1)
	s.persistence.EXPECT().UpdateKeypairXPub(coldKp.KeyUID, "", accsmanagementtypes.ColdWalletTypeNone, uint64(3)).Return(nil).Times(1)

	filesBefore := s.countKeystoreFiles()
	keyUID, err := s.accManager.MigrateColdWalletKeypairToApp(mnemonic2, "never-verified-password", 3)
	s.Require().NoError(err, "migration must proceed without password verification when the profile keypair is itself on a cold wallet — this is the only un-migrate path for keycard-profile users")
	s.Require().Equal(coldKp.KeyUID, keyUID)
	s.Require().Equal(filesBefore+2, s.countKeystoreFiles(),
		"keystore files must be recreated for the master key and the account path even though the password could not be verified")
}
