package account

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestVerifyAccountPassword(t *testing.T) {
	accManager := NewManager(nil)
	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(t, err)
	defer os.RemoveAll(keyStoreDir) //nolint: errcheck

	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts_empty")
	require.NoError(t, err)
	defer os.RemoveAll(emptyKeyStoreDir) //nolint: errcheck

	// import account keys
	require.NoError(t, ImportTestAccount(keyStoreDir, GetAccount1PKFile()))
	require.NoError(t, ImportTestAccount(keyStoreDir, GetAccount2PKFile()))

	account1Address := gethcommon.BytesToAddress(gethcommon.FromHex(TestConfig.Account1.WalletAddress))

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
			TestConfig.Account1.WalletAddress,
			TestConfig.Account1.Password,
			nil,
		},
		{
			"correct address, correct password, non-existent key store",
			filepath.Join(keyStoreDir, "non-existent-folder"),
			TestConfig.Account1.WalletAddress,
			TestConfig.Account1.Password,
			fmt.Errorf("cannot traverse key store folder: lstat %s/non-existent-folder: no such file or directory", keyStoreDir),
		},
		{
			"correct address, correct password, empty key store (pk is not there)",
			emptyKeyStoreDir,
			TestConfig.Account1.WalletAddress,
			TestConfig.Account1.Password,
			fmt.Errorf("cannot locate account for address: %s", account1Address.Hex()),
		},
		{
			"wrong address, correct password",
			keyStoreDir,
			"0x79791d3e8f2daa1f7fec29649d152c0ada3cc535",
			TestConfig.Account1.Password,
			fmt.Errorf("cannot locate account for address: %s", "0x79791d3E8F2dAa1F7FeC29649d152c0aDA3cc535"),
		},
		{
			"correct address, wrong password",
			keyStoreDir,
			TestConfig.Account1.WalletAddress,
			"wrong password", // wrong password
			errors.New("could not decrypt key with given passphrase"),
		},
	}
	for _, testCase := range testCases {
		accountKey, err := accManager.VerifyAccountPassword(testCase.keyPath, testCase.address, testCase.password)
		if !reflect.DeepEqual(err, testCase.expectedError) {
			require.FailNow(t, fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			if accountKey == nil {
				require.Fail(t, "no error reported, but account key is missing")
			}
			accountAddress := gethcommon.BytesToAddress(gethcommon.FromHex(testCase.address))
			if accountKey.Address != accountAddress {
				require.Fail(t, "account mismatch: have %s, want %s", accountKey.Address.Hex(), accountAddress.Hex())
			}
		}
	}
}

// TestVerifyAccountPasswordWithAccountBeforeEIP55 verifies if VerifyAccountPassword
// can handle accounts before introduction of EIP55.
func TestVerifyAccountPasswordWithAccountBeforeEIP55(t *testing.T) {
	keyStoreDir, err := ioutil.TempDir("", "status-accounts-test")
	require.NoError(t, err)
	defer os.RemoveAll(keyStoreDir) //nolint: errcheck

	// Import keys and make sure one was created before EIP55 introduction.
	err = ImportTestAccount(keyStoreDir, "test-account3-before-eip55.pk")
	require.NoError(t, err)

	accManager := NewManager(nil)

	address := gethcommon.HexToAddress(TestConfig.Account3.WalletAddress)
	_, err = accManager.VerifyAccountPassword(keyStoreDir, address.Hex(), TestConfig.Account3.Password)
	require.NoError(t, err)
}

var (
	errKeyStore   = errors.New("Can't return a key store")
	errAccManager = errors.New("Can't return an account manager")
)

func TestManagerTestSuite(t *testing.T) {
	gethServiceProvider := newMockGethServiceProvider(t)
	accManager := NewManager(gethServiceProvider)

	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(t, err)
	keyStore := keystore.NewKeyStore(keyStoreDir, keystore.LightScryptN, keystore.LightScryptP)
	defer os.RemoveAll(keyStoreDir) //nolint: errcheck

	testPassword := "test-password"

	// Initial test - create test account
	gethServiceProvider.EXPECT().AccountKeyStore().Return(keyStore, nil)
	accountInfo, mnemonic, err := accManager.CreateAccount(testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, accountInfo.WalletAddress)
	require.NotEmpty(t, accountInfo.WalletPubKey)
	require.NotEmpty(t, accountInfo.ChatAddress)
	require.NotEmpty(t, accountInfo.ChatPubKey)
	require.NotEmpty(t, mnemonic)

	// Before the complete decoupling of the keys, wallet and chat keys are the same
	assert.Equal(t, accountInfo.WalletAddress, accountInfo.ChatAddress)
	assert.Equal(t, accountInfo.WalletPubKey, accountInfo.ChatPubKey)

	s := &ManagerTestSuite{
		testAccount: testAccount{
			"test-password",
			accountInfo.WalletAddress,
			accountInfo.WalletPubKey,
			accountInfo.ChatAddress,
			accountInfo.ChatPubKey,
			mnemonic,
		},
		gethServiceProvider: gethServiceProvider,
		accManager:          accManager,
		keyStore:            keyStore,
		gethAccManager:      accounts.NewManager(),
	}

	suite.Run(t, s)
}

func newMockGethServiceProvider(t *testing.T) *MockGethServiceProvider {
	ctrl := gomock.NewController(t)
	return NewMockGethServiceProvider(ctrl)
}

type ManagerTestSuite struct {
	suite.Suite
	testAccount
	gethServiceProvider *MockGethServiceProvider
	accManager          *Manager
	keyStore            *keystore.KeyStore
	gethAccManager      *accounts.Manager
}

type testAccount struct {
	password      string
	walletAddress string
	walletPubKey  string
	chatAddress   string
	chatPubKey    string
	mnemonic      string
}

// reinitMock is for reassigning a new mock node manager to account manager.
// Stating the amount of times for mock calls kills the flexibility for
// development so this is a good workaround to use with EXPECT().Func().AnyTimes()
func (s *ManagerTestSuite) reinitMock() {
	s.gethServiceProvider = newMockGethServiceProvider(s.T())
	s.accManager.geth = s.gethServiceProvider
}

// SetupTest is used here for reinitializing the mock before every
// test function to avoid faulty execution.
func (s *ManagerTestSuite) SetupTest() {
	s.reinitMock()
}

func (s *ManagerTestSuite) TestCreateAccount() {
	// Don't fail on empty password
	s.gethServiceProvider.EXPECT().AccountKeyStore().Return(s.keyStore, nil)
	_, _, err := s.accManager.CreateAccount(s.password)
	s.NoError(err)

	s.gethServiceProvider.EXPECT().AccountKeyStore().Return(nil, errKeyStore)
	_, _, err = s.accManager.CreateAccount(s.password)
	s.Equal(errKeyStore, err)
}

func (s *ManagerTestSuite) TestRecoverAccount() {
	s.gethServiceProvider.EXPECT().AccountKeyStore().Return(s.keyStore, nil)
	accountInfo, err := s.accManager.RecoverAccount(s.password, s.mnemonic)
	s.NoError(err)
	s.Equal(s.walletAddress, accountInfo.WalletAddress)
	s.Equal(s.walletPubKey, accountInfo.WalletPubKey)
	s.Equal(s.chatAddress, accountInfo.ChatAddress)
	s.Equal(s.chatPubKey, accountInfo.ChatPubKey)

	s.gethServiceProvider.EXPECT().AccountKeyStore().Return(nil, errKeyStore)
	_, err = s.accManager.RecoverAccount(s.password, s.mnemonic)
	s.Equal(errKeyStore, err)
}

func (s *ManagerTestSuite) TestOnboarding() {
	// generates 5 random accounts
	count := 5
	accounts, err := s.accManager.NewOnboarding(count, 24)
	s.Require().NoError(err)
	s.Equal(count, len(accounts))

	// choose one account and encrypt it with password
	password := "test-onboarding-account"
	account := accounts[0]
	s.gethServiceProvider.EXPECT().AccountKeyStore().Return(s.keyStore, nil)
	info, mnemonic, err := s.accManager.ImportOnboardingAccount(account.ID, password)
	s.Require().NoError(err)
	s.Equal(account.Info, info)
	s.Equal(account.mnemonic, mnemonic)

	// try to decrypt it with password
	s.gethServiceProvider.EXPECT().AccountKeyStore().Return(s.keyStore, nil)
	decAccount, _, err := s.accManager.AddressToDecryptedAccount(info.WalletAddress, password)
	s.Require().NoError(err)
	s.Equal(info.WalletAddress, decAccount.Address.Hex())
}

func (s *ManagerTestSuite) TestSelectAccount() {
	testCases := []struct {
		name                  string
		accountKeyStoreReturn []interface{}
		walletAddress         string
		chatAddress           string
		password              string
		expectedError         error
	}{
		{
			"success",
			[]interface{}{s.keyStore, nil},
			s.walletAddress,
			s.chatAddress,
			s.password,
			nil,
		},
		{
			"fail_keyStore",
			[]interface{}{nil, errKeyStore},
			s.walletAddress,
			s.chatAddress,
			s.password,
			errKeyStore,
		},
		{
			"fail_wrongWalletAddress",
			[]interface{}{s.keyStore, nil},
			"wrong-wallet-address",
			s.chatAddress,
			s.password,
			ErrAddressToAccountMappingFailure,
		},
		{
			"fail_wrongChatAddress",
			[]interface{}{s.keyStore, nil},
			s.walletAddress,
			"wrong-chat-address",
			s.password,
			ErrAddressToAccountMappingFailure,
		},
		{
			"fail_wrongPassword",
			[]interface{}{s.keyStore, nil},
			s.walletAddress,
			s.chatAddress,
			"wrong-password",
			errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase"),
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			s.reinitMock()
			s.gethServiceProvider.EXPECT().AccountKeyStore().Return(testCase.accountKeyStoreReturn...).AnyTimes()
			err := s.accManager.SelectAccount(testCase.walletAddress, testCase.chatAddress, testCase.password)
			s.Equal(testCase.expectedError, err)

			selectedWalletAccount, walletErr := s.accManager.SelectedWalletAccount()
			selectedChatAccount, chatErr := s.accManager.SelectedChatAccount()

			if testCase.expectedError == nil {
				s.Equal(selectedWalletAccount.AccountKey, selectedChatAccount.AccountKey)
				s.NoError(walletErr)
				s.NoError(chatErr)
			} else {
				s.Nil(selectedWalletAccount)
				s.Nil(selectedChatAccount)
				s.Equal(walletErr, ErrNoAccountSelected)
				s.Equal(chatErr, ErrNoAccountSelected)
			}

			s.accManager.Logout()
		})
	}
}

func (s *ManagerTestSuite) TestSetChatAccount() {
	s.accManager.Logout()

	privKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	address := crypto.PubkeyToAddress(privKey.PublicKey)

	s.accManager.SetChatAccount(privKey)
	selectedChatAccount, err := s.accManager.SelectedChatAccount()
	s.Require().NoError(err)
	s.Require().NotNil(selectedChatAccount)
	s.Equal(privKey, selectedChatAccount.AccountKey.PrivateKey)
	s.Equal(address, selectedChatAccount.Address)

	selectedWalletAccount, err := s.accManager.SelectedWalletAccount()
	s.Error(err)
	s.Nil(selectedWalletAccount)
}

func (s *ManagerTestSuite) TestCreateChildAccount() {
	// First, test the negative case where an account is not selected
	// and an address is not provided.
	s.accManager.selectedWalletAccount = nil
	s.T().Run("fail_noAccount", func(t *testing.T) {
		s.gethServiceProvider.EXPECT().AccountKeyStore().Return(s.keyStore, nil).AnyTimes()
		_, _, err := s.accManager.CreateChildAccount("", s.password)
		s.Equal(ErrNoAccountSelected, err)
	})

	// Now, select the test account for rest of the test cases.
	s.reinitMock()
	s.gethServiceProvider.EXPECT().AccountKeyStore().Return(s.keyStore, nil).AnyTimes()
	err := s.accManager.SelectAccount(s.walletAddress, s.chatAddress, s.password)
	s.NoError(err)

	testCases := []struct {
		name                  string
		walletAddress         string
		chatAddress           string
		password              string
		accountKeyStoreReturn []interface{}
		expectedError         error
	}{
		{
			"success",
			s.walletAddress,
			s.chatAddress,
			s.password,
			[]interface{}{s.keyStore, nil},
			nil,
		},
		{
			"fail_keyStore",
			s.walletAddress,
			s.chatAddress,
			s.password,
			[]interface{}{nil, errKeyStore},
			errKeyStore,
		},
		{
			"fail_wrongWalletAddress",
			"wrong-address",
			s.chatAddress,
			s.password,
			[]interface{}{s.keyStore, nil},
			ErrAddressToAccountMappingFailure,
		},
		{
			"fail_wrongPassword",
			s.walletAddress,
			s.chatAddress,
			"wrong-password",
			[]interface{}{s.keyStore, nil},
			errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase"),
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			s.reinitMock()
			s.gethServiceProvider.EXPECT().AccountKeyStore().Return(testCase.accountKeyStoreReturn...).AnyTimes()
			childAddr, childPubKey, err := s.accManager.CreateChildAccount(testCase.walletAddress, testCase.password)
			if testCase.expectedError != nil {
				s.Equal(testCase.expectedError, err)
			} else {
				s.NoError(err)
				s.NotEmpty(childAddr)
				s.NotEmpty(childPubKey)
			}
		})
	}
}

func (s *ManagerTestSuite) TestLogout() {
	s.accManager.Logout()
	s.Nil(s.accManager.selectedWalletAccount)
}

// TestAccounts tests cases for (*Manager).Accounts.
func (s *ManagerTestSuite) TestAccounts() {
	// Select the test account
	s.gethServiceProvider.EXPECT().AccountKeyStore().Return(s.keyStore, nil).AnyTimes()
	err := s.accManager.SelectAccount(s.walletAddress, s.chatAddress, s.password)
	s.NoError(err)

	// Success
	s.gethServiceProvider.EXPECT().AccountManager().Return(s.gethAccManager, nil)
	accs, err := s.accManager.Accounts()
	s.NoError(err)
	s.NotNil(accs)

	// Can't get an account manager
	s.gethServiceProvider.EXPECT().AccountManager().Return(nil, errAccManager)
	_, err = s.accManager.Accounts()
	s.Equal(errAccManager, err)

	// Selected account is nil but doesn't fail
	s.accManager.selectedWalletAccount = nil
	s.gethServiceProvider.EXPECT().AccountManager().Return(s.gethAccManager, nil)
	accs, err = s.accManager.Accounts()
	s.NoError(err)
	s.NotNil(accs)
}

func (s *ManagerTestSuite) TestAddressToDecryptedAccount() {
	testCases := []struct {
		name                  string
		accountKeyStoreReturn []interface{}
		walletAddress         string
		password              string
		expectedError         error
	}{
		{
			"success",
			[]interface{}{s.keyStore, nil},
			s.walletAddress,
			s.password,
			nil,
		},
		{
			"fail_keyStore",
			[]interface{}{nil, errKeyStore},
			s.walletAddress,
			s.password,
			errKeyStore,
		},
		{
			"fail_wrongWalletAddress",
			[]interface{}{s.keyStore, nil},
			"wrong-wallet-address",
			s.password,
			ErrAddressToAccountMappingFailure,
		},
		{
			"fail_wrongPassword",
			[]interface{}{s.keyStore, nil},
			s.walletAddress,
			"wrong-password",
			errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase"),
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			s.reinitMock()
			s.gethServiceProvider.EXPECT().AccountKeyStore().Return(testCase.accountKeyStoreReturn...).AnyTimes()
			acc, key, err := s.accManager.AddressToDecryptedAccount(testCase.walletAddress, testCase.password)
			if testCase.expectedError != nil {
				s.Equal(testCase.expectedError, err)
			} else {
				s.NoError(err)
				s.NotNil(acc)
				s.NotNil(key)
				s.Equal(acc.Address, key.Address)
			}
		})
	}
}
