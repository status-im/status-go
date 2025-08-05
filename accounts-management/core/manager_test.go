package core

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/accounts-management/keystore"
	mock_persistence "github.com/status-im/status-go/accounts-management/mock"
	"github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/utils"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	gomock "go.uber.org/mock/gomock"
)

const testPassword = "test-password"
const newTestPassword = "new-test-password"

func TestVerifyAccountPassword(t *testing.T) {
	accManager, err := NewAccountsManager(tt.MustCreateTestLogger())
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	persistence := mock_persistence.NewMockPersistence(ctrl)
	accManager.SetPersistence(persistence)

	utils.Init() // initialize the test config

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
			utils.TestConfig.Account1.KeyUID,
			utils.TestConfig.Account1.WalletAddress,
			utils.TestConfig.Account1.Password,
			true,
			true,
			nil,
		},
		{
			"correct address, correct password, non-existent key store",
			utils.TestConfig.Account1.KeyUID,
			utils.TestConfig.Account1.WalletAddress,
			utils.TestConfig.Account1.Password,
			false,
			false,
			ErrAccountKeyStoreMissing,
		},
		{
			"correct address, correct password, empty key store (pk is not there)",
			utils.TestConfig.Account1.KeyUID,
			utils.TestConfig.Account1.WalletAddress,
			utils.TestConfig.Account1.Password,
			true,
			false,
			keystore.ErrNoMatch,
		},
		{
			"wrong address, correct password",
			utils.TestConfig.Account1.KeyUID,
			"0x79791d3e8f2daa1f7fec29649d152c0ada3cc535",
			utils.TestConfig.Account1.Password,
			true,
			true,
			keystore.ErrNoMatch,
		},
		{
			"correct address, wrong password",
			utils.TestConfig.Account1.KeyUID,
			utils.TestConfig.Account1.WalletAddress,
			"wrong password", // wrong password
			true,
			true,
			keystore.ErrDecrypt,
		},
	}
	for _, testCase := range testCases {

		rootDataDir := t.TempDir()
		accManager.SetRootDataDir(rootDataDir)
		keystore, err := accManager.createKeystore(testCase.keyUID)
		require.NoError(t, err)

		if testCase.importToLocation {
			err = utils.ImportTestAccount(keystore.KeystorePath(), utils.GetAccount1PKFile())
			require.NoError(t, err)

			// now we need to re-create the keystore in order to make the get-keystore aware of the copied account
			keystore, err = accManager.createKeystore(testCase.keyUID)
			require.NoError(t, err)
		}

		if testCase.keystoreSet {
			accManager.setKeystore(keystore)
		} else {
			accManager.setKeystore(nil)
		}

		ok, err := accManager.VerifyAccountPassword(ethtypes.HexToAddress(testCase.address), testCase.password)
		if testCase.expectedError != nil && err != nil && testCase.expectedError.Error() != err.Error() ||
			((testCase.expectedError == nil || err == nil) && testCase.expectedError != err) {
			require.FailNow(t, fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			require.True(t, ok)
		} else {
			require.False(t, ok)
		}
	}
}

// TestVerifyAccountPasswordWithAccountBeforeEIP55 verifies if VerifyAccountPassword
// can handle accounts before introduction of EIP55.
func TestVerifyAccountPasswordWithAccountBeforeEIP55(t *testing.T) {
	rootDataDir := t.TempDir()

	utils.Init() // initialize the test config

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accManager, err := NewAccountsManager(tt.MustCreateTestLogger())
	require.NoError(t, err)

	persistence := mock_persistence.NewMockPersistence(ctrl)
	accManager.SetPersistence(persistence)

	accManager.SetRootDataDir(rootDataDir)

	keystore, err := accManager.createKeystore(utils.TestConfig.Account3.KeyUID)
	require.NoError(t, err)

	err = utils.ImportTestAccount(keystore.KeystorePath(), "test-account3-before-eip55.pk") // Import keys and make sure one was created before EIP55 introduction.
	require.NoError(t, err)

	// now we need to reload the keystore (re-create it) in order to make the get-keystore aware of the copied account
	err = accManager.ReloadKeystore()
	require.Error(t, err)
	require.Equal(t, ErrNoAccountSelected, err)

	persistence.EXPECT().GetProfileKeypair().Return(
		&types.Keypair{
			KeyUID: utils.TestConfig.Account3.KeyUID,
		},
		nil,
	).Times(1)

	// Set the chat account, this will create a new keystore
	err = accManager.SetChatAccount(ethtypes.HexToAddress(utils.TestConfig.Account3.ChatAddress), utils.TestConfig.Account3.Password, nil)
	require.NoError(t, err)

	address := ethtypes.HexToAddress(utils.TestConfig.Account3.ChatAddress)
	ok, err := accManager.VerifyAccountPassword(address, utils.TestConfig.Account3.Password)
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
	walletAddress ethtypes.Address
	walletPubKey  string
	chatAddress   ethtypes.Address
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
	s.accManager, err = NewAccountsManager(tt.MustCreateTestLogger())
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

func (s *ManagerTestSuite) createAndStoreProfileKeypair() {
	s.persistence.EXPECT().GetKeypairByKeyUID(s.masterAccount.KeyUID()).Return(
		nil, types.ErrDbKeypairNotFound,
	).Times(1)

	s.persistence.EXPECT().GetPositionForNextNewAccount().Return(int64(0), nil).Times(1)

	s.persistence.EXPECT().SaveOrUpdateKeypair(gomock.Any()).Return(nil).Times(1)

	s.persistence.EXPECT().GetProfileKeypair().Return(
		&types.Keypair{
			KeyUID: s.masterAccount.KeyUID(),
		},
		nil,
	).Times(1)

	walletAccount := &types.AccountCreationDetails{
		Path: common.PathDefaultWalletAccount,
	}

	keypair, err := s.accManager.CreateKeypairFromMnemonicAndStore(s.mnemonic, s.password, "kp-name", walletAccount, true, 0)
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
				bytes.Equal(kpAcc.PublicKey, ethtypes.Hex2Bytes(s.chatPubKey)) &&
				kpAcc.KeyUID == keypair.KeyUID &&
				!kpAcc.Removed &&
				kpAcc.Clock == 0 &&
				!kpAcc.Wallet &&
				kpAcc.AddressWasNotShown &&
				kpAcc.Position == -1 &&
				kpAcc.Operable == types.AccountFullyOperable
		}
		if kpAcc.Wallet {
			walletAccountOk = kpAcc.Path == common.PathDefaultWalletAccount &&
				kpAcc.Address == s.walletAddress &&
				bytes.Equal(kpAcc.PublicKey, ethtypes.Hex2Bytes(s.walletPubKey)) &&
				kpAcc.KeyUID == keypair.KeyUID &&
				!kpAcc.Removed &&
				kpAcc.Clock == 0 &&
				!kpAcc.Chat &&
				kpAcc.AddressWasNotShown &&
				kpAcc.Position == 0 &&
				kpAcc.Operable == types.AccountFullyOperable
		}
	}
	s.Require().True(chatAccountOk)
	s.Require().True(walletAccountOk)
}

func (s *ManagerTestSuite) TestRecoverAccount() {
	s.createAndStoreProfileKeypair()
}

func (s *ManagerTestSuite) TestSetChatAccountSuccess() {
	s.testSetChatAccount(s.testAccount.chatAddress, s.testAccount.password, nil)
}

func (s *ManagerTestSuite) TestSetChatAccountWrongAddress() {
	s.testSetChatAccount(ethtypes.HexToAddress("0x0000000000000000000000000000000000000001"), s.testAccount.password, keystore.ErrNoMatch)
}

func (s *ManagerTestSuite) TestSetChatAccountWrongPassword() {
	s.testSetChatAccount(s.testAccount.chatAddress, "wrong", keystore.ErrDecrypt)
}

func (s *ManagerTestSuite) testSetChatAccount(chat ethtypes.Address, password string, expErr error) {
	s.createAndStoreProfileKeypair()
	s.accManager.setChatAccountAndProfileKeyUID(nil, "") // clear the chat account set by `createAndStoreProfileKeypair`

	s.persistence.EXPECT().GetProfileKeypair().Return(
		&types.Keypair{
			KeyUID: s.masterAccount.KeyUID(),
		},
		nil,
	).Times(1)

	err := s.accManager.SetChatAccount(chat, password, nil)
	s.Require().Equal(expErr, err)

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
		&types.Keypair{
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
		&types.Keypair{
			KeyUID: s.masterAccount.KeyUID(),
		},
		nil,
	).Times(1)

	// Select the test account, when the profile keypair is not stored
	err := s.accManager.SetChatAccount(s.chatAddress, s.password, nil)
	s.Require().Error(err)
	s.Equal(keystore.ErrNoMatch, err)

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

	checkAccount := func(address ethtypes.Address) bool {
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
	s.testAddressToAccount(ethtypes.HexToAddress("0x0001"), s.password, keystore.ErrNoMatch)
}

func (s *ManagerTestSuite) TestAddressToAccountWrongPassword() {
	s.testAddressToAccount(s.walletAddress, "wrong", keystore.ErrDecrypt)
}

func (s *ManagerTestSuite) testAddressToAccount(wallet ethtypes.Address, password string, expErr error) {
	s.createAndStoreProfileKeypair()

	key, err := s.accManager.LoadAccount(wallet, password)
	if expErr != nil {
		s.Equal(expErr, err)
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
	s.createAndStoreProfileKeypair()

	keyDir := s.getKeyDir()

	err := s.accManager.ReEncryptKeyStoreDir(testPassword, newTestPassword)
	s.Require().NoError(err)

	err = filepath.Walk(keyDir, func(path string, fileInfo os.FileInfo, err error) error {
		if fileInfo.IsDir() {
			return nil
		}

		// walk should not throw callback errors
		s.Require().NoError(err)

		rawKeyFile, err := os.ReadFile(path)
		s.Require().NoError(err)

		// should not decrypt with old password
		_, decryptError := common.DecryptKey(rawKeyFile, testPassword)
		s.Require().Error(decryptError)

		// should decrypt with new password
		_, decryptError = common.DecryptKey(rawKeyFile, newTestPassword)
		s.Require().NoError(decryptError)

		return nil
	})

	s.Require().NoError(err)
}
