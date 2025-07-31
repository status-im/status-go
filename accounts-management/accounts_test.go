package accountsmanagement

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/accounts-management/keystore/geth"
	mock_persistence "github.com/status-im/status-go/accounts-management/mock"
	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
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
			fmt.Errorf("no key for given address or file"),
		},
		{
			"wrong address, correct password",
			utils.TestConfig.Account1.KeyUID,
			"0x79791d3e8f2daa1f7fec29649d152c0ada3cc535",
			utils.TestConfig.Account1.Password,
			true,
			true,
			fmt.Errorf("no key for given address or file"),
		},
		{
			"correct address, wrong password",
			utils.TestConfig.Account1.KeyUID,
			utils.TestConfig.Account1.WalletAddress,
			"wrong password", // wrong password
			true,
			true,
			geth.ErrDecrypt,
		},
	}
	for _, testCase := range testCases {

		persistence.EXPECT().GetProfileKeypair().Return(
			&accounts.Keypair{
				KeyUID: testCase.keyUID,
			},
			nil,
		)

		rootDataDir := t.TempDir()
		accManager.SetRootDataDir(rootDataDir)
		keystore, err := accManager.createKeystore()
		require.NoError(t, err)

		if testCase.importToLocation {
			err = utils.ImportTestAccount(keystore.KeystorePath(), utils.GetAccount1PKFile())
			require.NoError(t, err)

			persistence.EXPECT().GetProfileKeypair().Return(
				&accounts.Keypair{
					KeyUID: testCase.keyUID,
				},
				nil,
			)

			// now we need to re-create the keystore in order to make the get-keystore aware of the copied account
			keystore, err = accManager.createKeystore()
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

	persistence.EXPECT().GetProfileKeypair().Return(
		&accounts.Keypair{
			KeyUID: utils.TestConfig.Account3.KeyUID,
		},
		nil,
	).Times(2)

	accManager.SetRootDataDir(rootDataDir)

	keystore, err := accManager.createKeystore()
	require.NoError(t, err)

	err = utils.ImportTestAccount(keystore.KeystorePath(), "test-account3-before-eip55.pk") // Import keys and make sure one was created before EIP55 introduction.
	require.NoError(t, err)

	// now we need to reload the keystore (re-create it) in order to make the get-keystore aware of the copied account
	err = accManager.ReloadKeystore()
	require.NoError(t, err)

	address := ethtypes.HexToAddress(utils.TestConfig.Account3.WalletAddress)
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
	accManager *AccountsManager
	keydir     string
}

type testAccount struct {
	password      string
	walletAddress string
	walletPubKey  string
	chatAddress   string
	chatPubKey    string
	mnemonic      string
}

// SetupTest is used here for reinitializing the mock before every
// test function to avoid faulty execution.
func (s *ManagerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	var err error
	s.accManager, err = NewAccountsManager(tt.MustCreateTestLogger())
	s.Require().NoError(err)

	persistence := mock_persistence.NewMockPersistence(ctrl)
	s.accManager.SetPersistence(persistence)

	rootDataDir := s.T().TempDir()
	s.accManager.SetRootDataDir(rootDataDir)

	// Initial test - create test account
	mnemonic, err := common.CreateRandomMnemonicWithDefaultLength()
	s.Require().NoError(err)

	genAcc, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	s.Require().NoError(err)

	persistence.EXPECT().GetProfileKeypair().Return(
		&accounts.Keypair{
			KeyUID: genAcc.KeyUID(),
		},
		nil,
	).AnyTimes()

	genAccount, err := s.accManager.CreateFromMnemonicAndStoreAccount(mnemonic, testPassword, true)
	s.Require().NoError(err)
	s.Require().NotEmpty(mnemonic)
	s.Require().NotNil(genAccount)
	s.Require().Equal(genAcc.KeyUID(), genAccount.KeyUID())
	s.Require().NotNil(genAccount.PrivateKey())
	s.Require().NotNil(genAccount.ExtendedKey())

	accountInfo := genAccount.ToAccountInfo()

	s.keydir = s.accManager.keystore.KeystorePath()

	s.testAccount = testAccount{
		testPassword,
		accountInfo.Address,
		accountInfo.PublicKey,
		accountInfo.Address,
		accountInfo.PublicKey,
		mnemonic,
	}
}

func (s *ManagerTestSuite) TestRecoverAccount() {
	genAccount, err := s.accManager.CreateFromMnemonicAndStoreAccount(s.mnemonic, s.password, false)
	s.NoError(err)
	accountInfo := genAccount.ToAccountInfo()
	s.Equal(s.walletAddress, accountInfo.Address)
	s.Equal(s.walletPubKey, accountInfo.PublicKey)
	s.Equal(s.chatAddress, accountInfo.Address)
	s.Equal(s.chatPubKey, accountInfo.PublicKey)
}

func (s *ManagerTestSuite) TestSetChatAccountSuccess() {
	s.testSetChatAccount(ethtypes.HexToAddress(s.testAccount.chatAddress), ethtypes.HexToAddress(s.testAccount.walletAddress), s.testAccount.password, nil)
}

func (s *ManagerTestSuite) TestSetChatAccountWrongAddress() {
	s.testSetChatAccount(ethtypes.HexToAddress("0x0000000000000000000000000000000000000001"), ethtypes.HexToAddress(s.testAccount.walletAddress), s.testAccount.password, errors.New("no key for given address or file"))
}

func (s *ManagerTestSuite) TestSetChatAccountWrongPassword() {
	s.testSetChatAccount(ethtypes.HexToAddress(s.testAccount.chatAddress), ethtypes.HexToAddress(s.testAccount.walletAddress), "wrong", geth.ErrDecrypt)
}

func (s *ManagerTestSuite) testSetChatAccount(chat, wallet ethtypes.Address, password string, expErr error) {
	err := s.accManager.SetChatAccount(chat, password, nil)
	s.Require().Equal(expErr, err)

	selectedChatAccount, chatErr := s.accManager.SelectedChatAccount()

	if expErr == nil {
		s.Require().NoError(chatErr)
		s.Equal(chat, crypto.PubkeyToAddress(selectedChatAccount.PrivateKey().PublicKey))
		s.Require().NotNil(s.accManager.keystore)
		s.Equal(s.keydir, s.accManager.keystore.KeystorePath())
	} else {
		s.Nil(selectedChatAccount)
		s.Equal(chatErr, ErrNoAccountSelected)
	}

	s.accManager.Logout()
}

func (s *ManagerTestSuite) TestSetChatAccountForExistingProfile() {
	genAcc, err := generator.CreateAccountFromMnemonic(s.mnemonic, "")
	s.Require().NoError(err)

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
	// Select the test account
	err := s.accManager.SetChatAccount(ethtypes.HexToAddress(s.chatAddress), s.password, nil)
	s.NoError(err)

	// Success
	accs, err := s.accManager.Accounts()
	s.NoError(err)
	s.Len(accs, 1)
	s.Equal(ethtypes.HexToAddress(s.chatAddress), accs[0])
}

func (s *ManagerTestSuite) TestAddressToAccountSuccess() {
	s.testAddressToAccount(s.walletAddress, s.password, nil)
}

func (s *ManagerTestSuite) TestAddressToAccountWrongAddress() {
	s.testAddressToAccount("0x0001", s.password, errors.New("no key for given address or file"))
}

func (s *ManagerTestSuite) TestAddressToAccountWrongPassword() {
	s.testAddressToAccount(s.walletAddress, "wrong", geth.ErrDecrypt)
}

func (s *ManagerTestSuite) testAddressToAccount(wallet, password string, expErr error) {
	key, err := s.accManager.LoadAccount(ethtypes.HexToAddress(wallet), password)
	if expErr != nil {
		s.Equal(expErr, err)
	} else {
		s.Require().NoError(err)
		s.Require().NotNil(key)
		s.Equal(wallet, key.Address().Hex())
	}
}

func (s *ManagerTestSuite) TestMigrateKeyStoreDir() {
	oldKeyDir := s.keydir
	newKeyDir := filepath.Join(oldKeyDir, "new_dir")
	err := os.Mkdir(newKeyDir, 0777)
	s.Require().NoError(err)

	files, _ := os.ReadDir(newKeyDir)
	s.Equal(0, len(files))

	err = s.accManager.MigrateKeyStoreDir(newKeyDir)
	s.Require().NoError(err)

	files, _ = os.ReadDir(newKeyDir)
	s.Equal(1, len(files))
}

func (s *ManagerTestSuite) TestReEncryptKeyStoreDir() {

	err := s.accManager.ReEncryptKeyStoreDir(testPassword, newTestPassword)
	s.Require().NoError(err)

	err = filepath.Walk(s.keydir, func(path string, fileInfo os.FileInfo, err error) error {
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
