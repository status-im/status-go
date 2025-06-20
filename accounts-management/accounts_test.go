package accountsmanagement

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/keystore/geth"
	mock_persistence "github.com/status-im/status-go/accounts-management/mock"
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

func setKeystore(accManager *AccountsManager, keyStoreDir string) error {
	keystoreAdapter, err := geth.NewGethKeystoreAdapter(keyStoreDir, keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		return err
	}
	accManager.SetKeystore(keystoreAdapter)
	return nil
}

func TestVerifyAccountPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accManager, err := NewAccountsManager(tt.MustCreateTestLogger())
	require.NoError(t, err)

	persistence := mock_persistence.NewMockPersistence(ctrl)
	accManager.SetPersistence(persistence)

	keyStoreDir := t.TempDir()
	emptyKeyStoreDir := t.TempDir()

	// import account keys
	utils.Init()
	require.NoError(t, utils.ImportTestAccount(keyStoreDir, utils.GetAccount1PKFile()))
	require.NoError(t, utils.ImportTestAccount(keyStoreDir, utils.GetAccount2PKFile()))

	testCases := []struct {
		name          string
		keyPath       string
		address       string
		password      string
		expectedError error
	}{
		{
			"correct address, correct password (decrypt should succeed)",
			keyStoreDir,
			utils.TestConfig.Account1.WalletAddress,
			utils.TestConfig.Account1.Password,
			nil,
		},
		{
			"correct address, correct password, non-existent key store",
			filepath.Join(keyStoreDir, "non-existent-folder"),
			utils.TestConfig.Account1.WalletAddress,
			utils.TestConfig.Account1.Password,
			fmt.Errorf("no key for given address or file"),
		},
		{
			"correct address, correct password, empty key store (pk is not there)",
			emptyKeyStoreDir,
			utils.TestConfig.Account1.WalletAddress,
			utils.TestConfig.Account1.Password,
			fmt.Errorf("no key for given address or file"),
		},
		{
			"wrong address, correct password",
			keyStoreDir,
			"0x79791d3e8f2daa1f7fec29649d152c0ada3cc535",
			utils.TestConfig.Account1.Password,
			fmt.Errorf("no key for given address or file"),
		},
		{
			"correct address, wrong password",
			keyStoreDir,
			utils.TestConfig.Account1.WalletAddress,
			"wrong password", // wrong password
			geth.ErrDecrypt,
		},
	}
	for _, testCase := range testCases {
		err := setKeystore(accManager, testCase.keyPath)
		require.NoError(t, err)

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
	keyStoreDir := t.TempDir()

	// Import keys and make sure one was created before EIP55 introduction.
	utils.Init()
	err := utils.ImportTestAccount(keyStoreDir, "test-account3-before-eip55.pk")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accManager, err := NewAccountsManager(tt.MustCreateTestLogger())
	require.NoError(t, err)

	persistence := mock_persistence.NewMockPersistence(ctrl)
	accManager.SetPersistence(persistence)

	err = setKeystore(accManager, keyStoreDir)
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

	keyStoreDir := s.T().TempDir()
	err = setKeystore(s.accManager, keyStoreDir)
	s.Require().NoError(err)
	s.keydir = keyStoreDir

	// Initial test - create test account
	genAccount, mnemonic, err := s.accManager.CreateAndStoreAccount(testPassword)
	s.Require().NoError(err)
	s.Require().NotEmpty(mnemonic)
	s.Require().NotNil(genAccount.PrivateKey())
	s.Require().NotNil(genAccount.ExtendedKey())

	accountInfo := genAccount.ToAccountInfo()

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
	genAccount, err := s.accManager.CreateFromMnemonicAndStoreAccount(s.mnemonic, s.password)
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
	err := s.accManager.SetChatAccount(chat, password)
	s.Require().Equal(expErr, err)

	selectedChatAccount, chatErr := s.accManager.SelectedChatAccount()

	if expErr == nil {
		s.Require().NoError(chatErr)
		s.Equal(chat, crypto.PubkeyToAddress(selectedChatAccount.PrivateKey().PublicKey))
	} else {
		s.Nil(selectedChatAccount)
		s.Equal(chatErr, ErrNoAccountSelected)
	}

	s.accManager.Logout()
}

func (s *ManagerTestSuite) TestSetChatAccount() {
	s.accManager.Logout()

	privKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	address := crypto.PubkeyToAddress(privKey.PublicKey)

	s.Require().NoError(s.accManager.SetChatAccountWithPrivateKey(privKey))
	selectedChatAccount, err := s.accManager.SelectedChatAccount()
	s.Require().NoError(err)
	s.Require().NotNil(selectedChatAccount)
	s.Equal(privKey, selectedChatAccount.PrivateKey())
	s.Equal(address, selectedChatAccount.Address())
}

func (s *ManagerTestSuite) TestLogout() {
	s.accManager.Logout()
	s.Nil(s.accManager.selectedChatAccount)
	s.Nil(s.accManager.keystore)
}

// TestAccounts tests cases for (*Manager).Accounts.
func (s *ManagerTestSuite) TestAccounts() {
	// Select the test account
	err := s.accManager.SetChatAccount(ethtypes.HexToAddress(s.chatAddress), s.password)
	s.NoError(err)

	// Success
	accs, err := s.accManager.Accounts()
	s.NoError(err)
	s.NotNil(accs)
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

	address := ethtypes.HexToAddress(s.walletAddress).Hex()
	addresses := []string{address}
	err = s.accManager.MigrateKeyStoreDir(newKeyDir, addresses)
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
