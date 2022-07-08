package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/keystore"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/t/utils"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const testPassword = "test-password"
const newTestPassword = "new-test-password"

func TestVerifyAccountPassword(t *testing.T) {
	accManager := NewGethManager()
	keyStoreDir := t.TempDir()
	emptyKeyStoreDir := t.TempDir()

	// import account keys
	utils.Init()
	require.NoError(t, utils.ImportTestAccount(keyStoreDir, utils.GetAccount1PKFile()))
	require.NoError(t, utils.ImportTestAccount(keyStoreDir, utils.GetAccount2PKFile()))

	account1Address := types.BytesToAddress(types.FromHex(utils.TestConfig.Account1.WalletAddress))

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
			fmt.Errorf("cannot traverse key store folder: lstat %s/non-existent-folder: no such file or directory", keyStoreDir),
		},
		{
			"correct address, correct password, empty key store (pk is not there)",
			emptyKeyStoreDir,
			utils.TestConfig.Account1.WalletAddress,
			utils.TestConfig.Account1.Password,
			ErrCannotLocateKeyFile{fmt.Sprintf("cannot locate account for address: %s", account1Address.Hex())},
		},
		{
			"wrong address, correct password",
			keyStoreDir,
			"0x79791d3e8f2daa1f7fec29649d152c0ada3cc535",
			utils.TestConfig.Account1.Password,
			ErrCannotLocateKeyFile{fmt.Sprintf("cannot locate account for address: %s", "0x79791d3E8F2dAa1F7FeC29649d152c0aDA3cc535")},
		},
		{
			"correct address, wrong password",
			keyStoreDir,
			utils.TestConfig.Account1.WalletAddress,
			"wrong password", // wrong password
			errors.New("could not decrypt key with given password"),
		},
	}
	for _, testCase := range testCases {
		accountKey, err := accManager.VerifyAccountPassword(testCase.keyPath, testCase.address, testCase.password)
		if testCase.expectedError != nil && err != nil && testCase.expectedError.Error() != err.Error() ||
			((testCase.expectedError == nil || err == nil) && testCase.expectedError != err) {
			require.FailNow(t, fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			if accountKey == nil { // nolint: staticcheck
				require.Fail(t, "no error reported, but account key is missing")
			}
			accountAddress := types.BytesToAddress(types.FromHex(testCase.address))
			if accountKey.Address != accountAddress { // nolint: staticcheck
				require.Fail(t, "account mismatch: have %s, want %s", accountKey.Address.Hex(), accountAddress.Hex())
			}
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

	accManager := NewGethManager()

	address := types.HexToAddress(utils.TestConfig.Account3.WalletAddress)
	_, err = accManager.VerifyAccountPassword(keyStoreDir, address.Hex(), utils.TestConfig.Account3.Password)
	require.NoError(t, err)
}

func TestManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite
	testAccount
	accManager *GethManager
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
	s.accManager = NewGethManager()

	keyStoreDir := s.T().TempDir()
	s.Require().NoError(s.accManager.InitKeystore(keyStoreDir))
	s.keydir = keyStoreDir

	// Initial test - create test account
	_, accountInfo, mnemonic, err := s.accManager.CreateAccount(testPassword)
	s.Require().NoError(err)
	s.Require().NotEmpty(accountInfo.WalletAddress)
	s.Require().NotEmpty(accountInfo.WalletPubKey)
	s.Require().NotEmpty(accountInfo.ChatAddress)
	s.Require().NotEmpty(accountInfo.ChatPubKey)
	s.Require().NotEmpty(mnemonic)

	// Before the complete decoupling of the keys, wallet and chat keys are the same
	s.Equal(accountInfo.WalletAddress, accountInfo.ChatAddress)
	s.Equal(accountInfo.WalletPubKey, accountInfo.ChatPubKey)

	s.testAccount = testAccount{
		testPassword,
		accountInfo.WalletAddress,
		accountInfo.WalletPubKey,
		accountInfo.ChatAddress,
		accountInfo.ChatPubKey,
		mnemonic,
	}
}

func (s *ManagerTestSuite) TestRecoverAccount() {
	accountInfo, err := s.accManager.RecoverAccount(s.password, s.mnemonic)
	s.NoError(err)
	s.Equal(s.walletAddress, accountInfo.WalletAddress)
	s.Equal(s.walletPubKey, accountInfo.WalletPubKey)
	s.Equal(s.chatAddress, accountInfo.ChatAddress)
	s.Equal(s.chatPubKey, accountInfo.ChatPubKey)
}

func (s *ManagerTestSuite) TestOnboarding() {
	// try to choose an account before starting onboarding
	_, _, err := s.accManager.ImportOnboardingAccount("test-id", "test-password")
	s.Equal(ErrOnboardingNotStarted, err)

	// generates 5 random accounts
	count := 5
	accounts, err := s.accManager.StartOnboarding(count, 24)
	s.Require().NoError(err)
	s.Equal(count, len(accounts))

	// try to choose an account with an undefined id
	_, _, err = s.accManager.ImportOnboardingAccount("test-id", "test-password")
	s.Equal(ErrOnboardingAccountNotFound, err)

	// choose one account and encrypt it with password
	password := "test-onboarding-account"
	account := accounts[0]
	info, mnemonic, err := s.accManager.ImportOnboardingAccount(account.ID, password)
	s.Require().NoError(err)
	s.Equal(account.Info, info)
	s.Equal(account.mnemonic, mnemonic)
	s.Nil(s.accManager.onboarding)

	// try to decrypt it with password to check if it's been imported correctly
	decAccount, _, err := s.accManager.AddressToDecryptedAccount(info.WalletAddress, password)
	s.Require().NoError(err)
	s.Equal(info.WalletAddress, decAccount.Address.Hex())

	// try resetting onboarding
	_, err = s.accManager.StartOnboarding(count, 24)
	s.Require().NoError(err)
	s.NotNil(s.accManager.onboarding)

	s.accManager.RemoveOnboarding()
	s.Nil(s.accManager.onboarding)
}

func (s *ManagerTestSuite) TestSelectAccountSuccess() {
	s.testSelectAccount(types.HexToAddress(s.testAccount.chatAddress), types.HexToAddress(s.testAccount.walletAddress), s.testAccount.password, nil)
}

func (s *ManagerTestSuite) TestSelectAccountWrongAddress() {
	s.testSelectAccount(types.HexToAddress("0x0000000000000000000000000000000000000001"), types.HexToAddress(s.testAccount.walletAddress), s.testAccount.password, errors.New("cannot retrieve a valid key for a given account: no key for given address or file"))
}

func (s *ManagerTestSuite) TestSelectAccountWrongPassword() {
	s.testSelectAccount(types.HexToAddress(s.testAccount.chatAddress), types.HexToAddress(s.testAccount.walletAddress), "wrong", errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given password"))
}

func (s *ManagerTestSuite) testSelectAccount(chat, wallet types.Address, password string, expErr error) {
	loginParams := LoginParams{
		ChatAddress: chat,
		MainAccount: wallet,
		Password:    password,
	}
	err := s.accManager.SelectAccount(loginParams)
	s.Require().Equal(expErr, err)

	selectedMainAccountAddress, walletErr := s.accManager.MainAccountAddress()
	selectedChatAccount, chatErr := s.accManager.SelectedChatAccount()

	if expErr == nil {
		s.Require().NoError(walletErr)
		s.Require().NoError(chatErr)
		s.Equal(wallet, selectedMainAccountAddress)
		s.Equal(chat, crypto.PubkeyToAddress(selectedChatAccount.AccountKey.PrivateKey.PublicKey))
	} else {
		s.Equal(types.Address{}, selectedMainAccountAddress)
		s.Nil(selectedChatAccount)
		s.Equal(walletErr, ErrNoAccountSelected)
		s.Equal(chatErr, ErrNoAccountSelected)
	}

	s.accManager.Logout()
}

func (s *ManagerTestSuite) TestSetChatAccount() {
	s.accManager.Logout()

	privKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	address := crypto.PubkeyToAddress(privKey.PublicKey)

	s.Require().NoError(s.accManager.SetChatAccount(privKey))
	selectedChatAccount, err := s.accManager.SelectedChatAccount()
	s.Require().NoError(err)
	s.Require().NotNil(selectedChatAccount)
	s.Equal(privKey, selectedChatAccount.AccountKey.PrivateKey)
	s.Equal(address, selectedChatAccount.Address)

	selectedMainAccountAddress, err := s.accManager.MainAccountAddress()
	s.Error(err)
	s.Equal(types.Address{}, selectedMainAccountAddress)
}

func (s *ManagerTestSuite) TestLogout() {
	s.accManager.Logout()
	s.Equal(types.Address{}, s.accManager.mainAccountAddress)
	s.Nil(s.accManager.selectedChatAccount)
	s.Len(s.accManager.watchAddresses, 0)
}

// TestAccounts tests cases for (*Manager).Accounts.
func (s *ManagerTestSuite) TestAccounts() {
	// Select the test account
	loginParams := LoginParams{
		MainAccount: types.HexToAddress(s.walletAddress),
		ChatAddress: types.HexToAddress(s.chatAddress),
		Password:    s.password,
	}
	err := s.accManager.SelectAccount(loginParams)
	s.NoError(err)

	// Success
	accs, err := s.accManager.Accounts()
	s.NoError(err)
	s.NotNil(accs)
	// Selected main account address is zero address but doesn't fail
	s.accManager.mainAccountAddress = types.Address{}
	accs, err = s.accManager.Accounts()
	s.NoError(err)
	s.NotNil(accs)
}

func (s *ManagerTestSuite) TestAddressToDecryptedAccountSuccess() {
	s.testAddressToDecryptedAccount(s.walletAddress, s.password, nil)
}

func (s *ManagerTestSuite) TestAddressToDecryptedAccountWrongAddress() {
	s.testAddressToDecryptedAccount("0x0001", s.password, ErrAddressToAccountMappingFailure)
}

func (s *ManagerTestSuite) TestAddressToDecryptedAccountWrongPassword() {
	s.testAddressToDecryptedAccount(s.walletAddress, "wrong", errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given password"))
}

func (s *ManagerTestSuite) testAddressToDecryptedAccount(wallet, password string, expErr error) {
	acc, key, err := s.accManager.AddressToDecryptedAccount(wallet, password)
	if expErr != nil {
		s.Equal(expErr, err)
	} else {
		s.Require().NoError(err)
		s.Require().NotNil(acc)
		s.Require().NotNil(key)
		s.Equal(acc.Address, key.Address)
	}
}

func (s *ManagerTestSuite) TestMigrateKeyStoreDir() {
	oldKeyDir := s.keydir
	newKeyDir := filepath.Join(oldKeyDir, "new_dir")
	err := os.Mkdir(newKeyDir, 0777)
	s.Require().NoError(err)

	files, _ := ioutil.ReadDir(newKeyDir)
	s.Equal(0, len(files))

	address := types.HexToAddress(s.walletAddress).Hex()
	addresses := []string{address}
	err = s.accManager.MigrateKeyStoreDir(oldKeyDir, newKeyDir, addresses)
	s.Require().NoError(err)

	files, _ = ioutil.ReadDir(newKeyDir)
	s.Equal(1, len(files))
}

func (s *ManagerTestSuite) TestReEncryptKey() {
	var firstKeyPath string
	files, _ := ioutil.ReadDir(s.keydir)

	// thiere is only one file in this dir,
	// is there a better way to reference it?
	for _, f := range files {
		firstKeyPath = filepath.Join(s.keydir, f.Name())
	}

	rawKey, _ := ioutil.ReadFile(firstKeyPath)
	reEncryptedKey, _ := s.accManager.ReEncryptKey(rawKey, testPassword, newTestPassword)

	type Key struct {
		Address string `json:"address"`
	}

	var unmarshaledRaw, unmarshaledReEncrypted Key
	_ = json.Unmarshal(rawKey, &unmarshaledRaw)
	_ = json.Unmarshal(reEncryptedKey, &unmarshaledReEncrypted)

	oldCrypto, _ := keystore.RawKeyToCryptoJSON(rawKey)
	newCrypto, _ := keystore.RawKeyToCryptoJSON(reEncryptedKey)

	// Test address is same post re-encryption
	s.Equal(unmarshaledRaw.Address, unmarshaledReEncrypted.Address)

	// Test cipher changes after re-encryption
	s.NotEqual(oldCrypto.CipherText, newCrypto.CipherText)

	// Test re-encrypted key cannot be decrypted using old testPasswordword
	_, decryptOldError := keystore.DecryptKey(reEncryptedKey, testPassword)
	s.Require().Error(decryptOldError)

	// Test re-encrypted key can be decrypted using new testPassword
	_, decryptNewError := keystore.DecryptKey(reEncryptedKey, newTestPassword)
	s.Require().NoError(decryptNewError)
}

func (s *ManagerTestSuite) TestReEncryptKeyStoreDir() {

	err := s.accManager.ReEncryptKeyStoreDir(s.keydir, testPassword, newTestPassword)
	s.Require().NoError(err)

	err = filepath.Walk(s.keydir, func(path string, fileInfo os.FileInfo, err error) error {
		if fileInfo.IsDir() {
			return nil
		}

		// walk should not throw callback errors
		s.Require().NoError(err)

		rawKeyFile, err := ioutil.ReadFile(path)
		s.Require().NoError(err)

		// should not decrypt with old password
		_, decryptError := keystore.DecryptKey(rawKeyFile, testPassword)
		s.Require().Error(decryptError)

		// should decrypt with new password
		_, decryptError = keystore.DecryptKey(rawKeyFile, newTestPassword)
		s.Require().NoError(decryptError)

		return nil
	})

	s.Require().NoError(err)
}
