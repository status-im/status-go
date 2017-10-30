package account_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/require"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/suite"
)

const (
	tmpDirName        = "status-tmp"
	account1          = "test-account1.pk"
	account2          = "test-account2.pk"
	accountEIP55      = "test-account1-before-eip55.pk"
	incorrectAddress  = "incorrect_address"
	incorrectPassword = "incorrect_password"
)

func TestAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(AccountsTestSuite))
}

type AccountsTestSuite struct {
	BaseTestSuite
	config *params.NodeConfig
}

func (s *AccountsTestSuite) SetupSuite() {
	require := s.Require()

	// setup node config
	config, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)
	require.NotNil(config)

	// make sure that whisper is enabled
	require.True(config.WhisperConfig.Enabled)

	s.config = config
}

func (s *AccountsTestSuite) SetupTest() {
	require := s.Require()
	s.NodeManager = node.NewNodeManager()
	require.NotNil(s.NodeManager)
}

func (s *AccountsTestSuite) BeforeTest(suiteName, testName string) {
	require := s.Require()

	// setup empty key store
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), tmpDirName)
	require.NoError(err)
	require.NotNil(emptyKeyStoreDir)

	// update config
	s.config.KeyStoreDir = emptyKeyStoreDir

	//@NOTE(rgeraldes) account1 file will not be present in the keystore
	// import accounts
	require.NoError(common.ImportTestAccount(emptyKeyStoreDir, account2))

	// start node
	nodeStarted, err := s.NodeManager.StartNode(s.config)
	require.NoError(err)
	require.NotNil(nodeStarted)
	<-nodeStarted
}

func (s *AccountsTestSuite) AfterTest(suiteName, testName string) {
	require := s.Require()

	// stop node
	nodeStopped, err := s.NodeManager.StopNode()
	require.NoError(err)
	require.NotNil(nodeStopped)
	<-nodeStopped

	require.NoError(os.RemoveAll(s.config.KeyStoreDir))
}

func (s *AccountsTestSuite) TestCreateAccount() {
	require := s.Require()

	// setup account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test cases
	testCases := []struct {
		context       string
		password      string
		expectedError error
	}{
		{
			"non-empty password",
			"password",
			nil,
		},
		{
			"empty password",
			"",
			nil,
		},
	}

	// test
	for _, testCase := range testCases {
		func() {
			s.T().Log(testCase.context)

			addr, pubKey, mnemonic, err := accountManager.CreateAccount(testCase.password)
			if err != testCase.expectedError {
				require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
			}
			if err == nil {
				require.NotEmpty(addr)
				require.NotEmpty(pubKey)
				require.NotEmpty(mnemonic)
			}
		}()
	}
}

func (s *AccountsTestSuite) TestCreateChildAccount() {
	require := s.Require()

	// test cases
	testCases := []struct {
		context            string
		parentAddress      string
		parentPassword     string
		defaultKeyAddress  string
		defaultKeyPassword string
		expectedError      error
	}{
		{
			context:        "parent address is empty, no default account",
			parentAddress:  "",
			parentPassword: "password",
			expectedError:  account.ErrNoAccountSelected,
		},
		{
			context:        "parent address is not empty, incorrect address format",
			parentAddress:  incorrectAddress,
			parentPassword: "password",
			expectedError:  account.ErrAddressToAccountMappingFailure,
		},

		{
			context:        "correct parent address, incorrect password",
			parentAddress:  TestConfig.Account2.Address,
			parentPassword: incorrectPassword,
			expectedError:  errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase"),
		},

		{
			context:        "correct parent address, correct password",
			parentAddress:  TestConfig.Account2.Address,
			parentPassword: TestConfig.Account2.Password,
			expectedError:  nil,
		},
		{
			context:            "parent address is empty, default account is not empty, incorrect password",
			parentAddress:      "",
			parentPassword:     incorrectPassword,
			defaultKeyAddress:  TestConfig.Account2.Address,
			defaultKeyPassword: TestConfig.Account2.Password,
			expectedError:      errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase"),
		},
		{
			context:            "parent address is empty, selected account is not empty, correct password",
			parentAddress:      "",
			parentPassword:     TestConfig.Account2.Password,
			defaultKeyAddress:  TestConfig.Account2.Address,
			defaultKeyPassword: TestConfig.Account2.Password,
			expectedError:      nil,
		},
	}

	// test
	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		// account manager
		accountManager := account.NewManager(s.NodeManager)
		require.NotNil(accountManager)

		// select default account if details were provided
		if len(testCase.defaultKeyAddress) > 0 && len(testCase.defaultKeyPassword) > 0 {
			require.NoError(accountManager.SelectAccount(testCase.defaultKeyAddress, testCase.defaultKeyPassword))
		}

		address, pubKey, err := accountManager.CreateChildAccount(testCase.parentAddress, testCase.parentPassword)
		if !reflect.DeepEqual(err, testCase.expectedError) {
			s.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			require.NotEmpty(address)
			require.NotEmpty(pubKey)
		}
	}
}

// @TODO(rgeraldes) - RecoverAccount is not ok functionality-wise.
func (s *AccountsTestSuite) TestRecoverAccount() {
	require := s.Require()

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	testCases := []struct {
		context       string
		password      string
		mnemonic      string
		expectedError error
	}{
		{
			"mnemonic & password provided",
			"password",
			"mnemonic",
			nil,
		},
	}

	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		address, key, err := accountManager.RecoverAccount(testCase.password, testCase.mnemonic)
		if err != testCase.expectedError {
			require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			require.NotEmpty(address)
			require.NotEmpty(key)
		}
	}
}

func (s *AccountsTestSuite) TestVerifyAccountPassword(t *testing.T) {
	require := s.Require()

	keyStoreDir := s.config.KeyStoreDir
	emptyKeyStoreDir, err := ioutil.TempDir("", "temp")
	require.NoError(err)
	require.NotEmpty(emptyKeyStoreDir)

	// import account keys
	require.NoError(common.ImportTestAccount(keyStoreDir, account1))

	// setup account manager
	accountManager := account.NewManager(nil)
	require.NotNil(accountManager)

	account1Address := gethcommon.BytesToAddress(gethcommon.FromHex(TestConfig.Account1.Address))

	testCases := []struct {
		context       string
		keyPath       string
		address       string
		password      string
		expectedError error
	}{
		{
			"correct address, correct password",
			keyStoreDir,
			TestConfig.Account1.Address,
			TestConfig.Account1.Password,
			nil,
		},
		{
			"correct address, correct password, non-existent key store",
			filepath.Join(keyStoreDir, "non-existent-folder"),
			TestConfig.Account1.Address,
			TestConfig.Account1.Password,
			fmt.Errorf("cannot traverse key store folder: lstat %s/non-existent-folder: no such file or directory", keyStoreDir),
		},
		{
			"correct address, correct password, empty key store (pk is not there)",
			emptyKeyStoreDir,
			TestConfig.Account1.Address,
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
			TestConfig.Account1.Address,
			incorrectPassword, // wrong password
			errors.New("could not decrypt key with given passphrase"),
		},
	}
	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		accountKey, err := accountManager.VerifyAccountPassword(testCase.keyPath, testCase.address, testCase.password)
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
	err = common.ImportTestAccount(keyStoreDir, "test-account1-before-eip55.pk")
	require.NoError(t, err)

	acctManager := account.NewManager(nil)

	address := gethcommon.HexToAddress(TestConfig.Account1.Address)
	_, err = acctManager.VerifyAccountPassword(keyStoreDir, address.Hex(), TestConfig.Account1.Password)
	require.NoError(t, err)
}

func (s *AccountsTestSuite) TestVerifyAccountPasswordWithAccountBeforeEIP55() {
	require := s.Require()

	// init empty keyStore
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "status-empty-keystore")
	require.NoError(err)
	require.NotEmpty(emptyKeyStoreDir)
	defer os.RemoveAll(emptyKeyStoreDir)

	// import keys and make sure one was created before EIP55 introduction.
	err = common.ImportTestAccount(emptyKeyStoreDir, "test-account1-before-eip55.pk")
	require.NoError(err)

	// account manager
	accountManager := account.NewManager(nil)

	// test
	address := gethcommon.HexToAddress(TestConfig.Account1.Address)
	_, err = accountManager.VerifyAccountPassword(emptyKeyStoreDir, address.Hex(), TestConfig.Account1.Password)
	require.NoError(err)
}

/*

func (s *AccountsTestSuite) TestSelectAccount() {
	require := s.Require()

	// create node config
	config, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)
	require.NotNil(config)

	// create empty keyStore
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "empty")
	require.NoError(err)
	config.KeyStoreDir = emptyKeyStoreDir

	// start node manager > create & start node
	nodeStarted, err := s.NodeManager.StartNode(config)
	require.NoError(err)
	defer func() {
		nodeStopped, err := s.NodeManager.StopNode()
		require.NoError(err)
		<-nodeStopped
		os.RemoveAll(emptyKeyStoreDir)
	}()
	<-nodeStarted

	// create an instance of the account manager to call
	// the CreateAccount method
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	testCases := []struct {
		context       string
		address       string
		password      string
		expectedError error
	}{
		{
			context: "invalid address",
			address: "invalid_address",
		},
		{
			"correct address, invalid password",
			"",
			"",
			account.ErrAddressToAccountMappingFailure,
		},
		{
			"correct address, correct password",
			"",
			"",
			account.ErrAccountToKeyMappingFailure,
		},
	}

	for _, test := range testCases {
		err = accountManager.SelectAccount(test.address, test.password)
		if !reflect.DeepEqual(err, test.expectedError) {
			require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", test.expectedError, err))
		}
	}
}

func (s *AccountsTestSuite) TestSelectedAccount() {
	var err error
	require := s.Require()

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	testCases := []struct {
		selectedAccount *common.SelectedExtKey
		expectedError   error
	}{
		// selected account = nil
		{
			nil,
			account.ErrNoAccountSelected,
		},
		// selected account != nil
		{
			nil,
			nil,
		},
	}
	for _, test := range testCases {
		selectedAccount, err := accountManager.SelectedAccount()
		if !reflect.DeepEqual(err, test.expectedError) {
			require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", test.expectedError, err))
		}
		if err == nil {
			require.NotNil(selectedAccount)
		}

	}

}

func (s *AccountsTestSuite) ReSelectAccount() {
	require := s.Require()

	// create node config
	config, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)
	require.NotNil(config)

	// create empty keyStore
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "empty")
	require.NoError(err)
	require.NotEmpty(emptyKeyStoreDir)
	config.KeyStoreDir = emptyKeyStoreDir

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	testCases := []struct {
		context       string
		launchNode    bool
		enableWhisper bool
		expectedError error
	}{
		{
			"selected account equals nil",
			false,
			false,
			nil,
		},
		{
			"node started, whisper enabled",
			true,
			true,
			nil,
		},
		{
			"node started, whisper disabled",
			true,
			false,
			nil,
		},
		{
			"node stopped, whisper disabled",
			false,
			false,
			nil,
		},
	}

	for _, testCase := range testCases {
		func() {
			if testCase.enableWhisper {
				config.WhisperConfig.Enabled = true
			} else {
				config.WhisperConfig.Enabled = false
			}

			if testCase.launchNode {
				nodeStarted, err := s.NodeManager.StartNode(config)
				require.NoError(err)
				require.NotNil(nodeStarted)
				defer func() {
					nodeStopped, err := s.NodeManager.StopNode()
					require.NoError(err)
					require.NotNil(nodeStopped)
					<-nodeStopped
					require.NoError(os.RemoveAll(emptyKeyStoreDir))
				}()
				<-nodeStarted
			}

			err = accountManager.ReSelectAccount()
			if err != testCase.expectedError {
				s.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
			}
		}()
	}
}

*/

func (s *AccountsTestSuite) TestLogout() {
	require := s.Require()

	// init node config
	config, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)
	require.NotNil(config)

	// init empty keyStore
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "empty")
	require.NoError(err)
	require.NotEmpty(emptyKeyStoreDir)
	config.KeyStoreDir = emptyKeyStoreDir

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test cases
	testCases := []struct {
		context       string
		launchNode    bool
		enableWhisper bool
		expectedError error
	}{
		{
			context:       "node stopped",
			launchNode:    false,
			expectedError: node.ErrNoRunningNode,
		},
		{
			"node started, whisper service enabled",
			true,
			true,
			nil,
		},
		{
			"node started, whisper service disabled",
			true,
			false,
			node.ErrInvalidWhisperService,
		},
	}

	// test
	for _, testCase := range testCases {
		func() {
			config.WhisperConfig.Enabled = testCase.enableWhisper

			if testCase.launchNode {
				// init node
				nodeStarted, err := s.NodeManager.StartNode(config)
				require.NoError(err)
				require.NotNil(nodeStarted)
				<-nodeStarted

				// defer
				defer func() {
					nodeStopped, err := s.NodeManager.StopNode()
					require.NoError(err)
					require.NotNil(nodeStopped)
					<-nodeStopped
					require.NoError(os.RemoveAll(emptyKeyStoreDir))
				}()
			}

			err := accountManager.Logout()
			if err != testCase.expectedError {
				s.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
			}
		}()
	}
}

/*
func (s *AccountsTestSuite) TestAccounts() {
	require := s.Require()

}
*/

func (s *AccountsTestSuite) TestAccountsRPCHandler() {
	require := s.Require()

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test
	require.NotNil(accountManager.AccountsRPCHandler())
}

func (s *AccountsTestSuite) TestAddressToDecryptedAccount() {
	require := s.Require()

	// init node config
	config, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)
	require.NotNil(config)

	// init empty keyStore
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "status-empty-keystore")
	require.NoError(err)
	require.NotEmpty(emptyKeyStoreDir)
	defer func() {
		require.NoError(os.RemoveAll(emptyKeyStoreDir))
	}()
	config.KeyStoreDir = emptyKeyStoreDir

	// import account key
	require.NoError(common.ImportTestAccount(emptyKeyStoreDir, "test-account1.pk"))

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test cases
	testCases := []struct {
		context       string
		launchNode    bool
		address       string
		password      string
		expectedError error
	}{
		{
			context:       "node stopped",
			launchNode:    false,
			expectedError: node.ErrNoRunningNode,
		},
		{
			"node started, correct address, correct password",
			true,
			TestConfig.Account1.Address,
			TestConfig.Account1.Password,
			nil,
		},
		{
			"node started, correct address, incorrect password",
			true,
			TestConfig.Account1.Address,
			"invalidpassword", // value different than TestConfig.Account1.Password
			keystore.ErrDecrypt,
		},
		{
			context:       "node started, incorrect address format (password is not relevant)",
			launchNode:    true,
			address:       "12345",
			expectedError: account.ErrAddressToAccountMappingFailure,
		},
		{
			context:       "node started, address does not exist (password is not relevant)",
			launchNode:    true,
			address:       TestConfig.Account2.Address,
			expectedError: keystore.ErrNoMatch,
		},
	}

	// test
	for _, testCase := range testCases {
		func() {
			s.T().Log(testCase.context)

			if testCase.launchNode {
				// init node
				nodeStarted, err := s.NodeManager.StartNode(config)
				require.NoError(err)
				require.NotNil(nodeStarted)
				<-nodeStarted

				// defer
				defer func() {
					nodeStopped, err := s.NodeManager.StopNode()
					require.NoError(err)
					require.NotNil(nodeStopped)
					<-nodeStopped
				}()
			}

			account, key, err := accountManager.AddressToDecryptedAccount(testCase.address, testCase.password)
			if err != testCase.expectedError {
				require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
			}
			if err == nil {
				require.NotNil(account)
				require.NotNil(key)
			}
		}()
	}
}

func (s *AccountsTestSuite) TestVerifyAccountPasswordWithAccountBeforeEIP55() {
	require := s.Require()

	// import key created before EIP55 introduction.
	require.NoError(common.ImportTestAccount(s.config.KeyStoreDir, accountEIP55))

	// account manager
	accountManager := account.NewManager(nil)

	// test
	_, err := accountManager.VerifyAccountPassword(s.config.KeyStoreDir, TestConfig.Account1.Address, TestConfig.Account1.Password)
	require.NoError(err)
}

func (s *AccountsTestSuite) TestSelectAccount() {
	require := s.Require()

	// setup account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// testCases
	testCases := []struct {
		context       string
		address       string
		password      string
		expectedError error
	}{
		{
			context:       "incorrect address",
			address:       incorrectAddress,
			expectedError: account.ErrAddressToAccountMappingFailure,
		},
		{
			context:       "correct address, incorrect password",
			address:       TestConfig.Account2.Address,
			password:      incorrectPassword,
			expectedError: errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase"),
		},
		{
			"correct address, correct password, whisper enabled",
			TestConfig.Account2.Address,
			TestConfig.Account2.Password,
			nil,
		},
	}

	// test
	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		if err := accountManager.SelectAccount(testCase.address, testCase.password); !reflect.DeepEqual(err, testCase.expectedError) {
			require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
	}
}

func (s *AccountsTestSuite) TestSelectedAccount() {
	require := s.Require()

	// setup account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test cases
	testCases := []struct {
		context       string
		address       string
		password      string
		expectedError error
	}{
		{
			"selected account is nil",
			"",
			"",
			account.ErrNoAccountSelected,
		},
		{
			"selected account is account2",
			TestConfig.Account2.Address,
			TestConfig.Account2.Password,
			nil,
		},
	}

	// test
	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		// select account if requested
		if len(testCase.address) > 0 && len(testCase.password) > 0 {
			require.NoError(accountManager.SelectAccount(testCase.address, testCase.password))
		}

		key, err := accountManager.SelectedAccount()
		if err != testCase.expectedError {
			s.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			require.NotNil(key)
		}
	}
}

func (s *AccountsTestSuite) TestReSelectAccount() {
	require := s.Require()

	// test cases
	testCases := []struct {
		context       string
		address       string
		password      string
		expectedError error
	}{
		{
			context:       "selected account is nil",
			expectedError: nil,
		},
		{
			"selected account is account2",
			TestConfig.Account2.Address,
			TestConfig.Account2.Password,
			nil,
		},
	}

	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		// account manager
		accountManager := account.NewManager(s.NodeManager)
		require.NotNil(accountManager)

		// select account if details were provided
		if len(testCase.address) > 0 && len(testCase.password) > 0 {
			require.NoError(accountManager.SelectAccount(testCase.address, testCase.password))
		}

		if err := accountManager.ReSelectAccount(); err != testCase.expectedError {
			s.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
	}
}

func (s *AccountsTestSuite) TestLogout() {
	require := s.Require()

	// setup account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test cases
	testCases := []struct {
		context       string
		expectedError error
	}{
		{
			"node started, whisper service enabled",
			nil,
		},
	}

	// test
	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		err := accountManager.Logout()
		if err != testCase.expectedError {
			s.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
	}
}

func (s *AccountsTestSuite) TestAccounts() {
	require := s.Require()

	// test cases
	testCases := []struct {
		context         string
		address         string
		password        string
		hasChildAccount bool
		expectedError   error
		nAddresses      int
	}{
		{
			context:       "selected account is nill",
			expectedError: nil,
			nAddresses:    0,
		},
		{
			"selected account is account2 with no child accounts",
			TestConfig.Account2.Address,
			TestConfig.Account2.Password,
			false,
			nil,
			1,
		},
		{
			"selected account is account2 with 1 child account",
			TestConfig.Account2.Address,
			TestConfig.Account2.Password,
			true,
			nil,
			2,
		},
	}

	// test
	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		// setup account manager
		accountManager := account.NewManager(s.NodeManager)
		require.NotNil(accountManager)

		// select default account if requested
		if len(testCase.address) > 0 && len(testCase.password) > 0 {
			require.NoError(accountManager.SelectAccount(testCase.address, testCase.password))
		}

		// create child account for the selected account if requested
		if testCase.hasChildAccount {
			address, pubKey, err := accountManager.CreateChildAccount("", testCase.password)
			require.NoError(err)
			require.NotEmpty(address)
			require.NotEmpty(pubKey)

		}

		addresses, err := accountManager.Accounts()
		if err != testCase.expectedError {
			require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			require.NotNil(addresses)
			require.Equal(testCase.nAddresses, len(addresses))
		}
	}
}

func (s *AccountsTestSuite) TestAccountsRPCHandler() {
	require := s.Require()

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test
	require.NotNil(accountManager.AccountsRPCHandler())
}

func (s *AccountsTestSuite) TestAddressToDecryptedAccount() {
	require := s.Require()

	// import account key
	require.NoError(common.ImportTestAccount(s.config.KeyStoreDir, account1))

	// setup account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test cases
	testCases := []struct {
		context       string
		launchNode    bool
		address       string
		password      string
		expectedError error
	}{
		{
			"correct address, correct password",
			true,
			TestConfig.Account2.Address,
			TestConfig.Account2.Password,
			nil,
		},
		{
			"correct address, incorrect password",
			true,
			TestConfig.Account2.Address,
			"invalidpassword", // value different than TestConfig.Account1.Password
			keystore.ErrDecrypt,
		},
		{
			context:       "incorrect address format",
			launchNode:    true,
			address:       "12345",
			expectedError: account.ErrAddressToAccountMappingFailure,
		},
		{
			context:       "address does not exist",
			launchNode:    true,
			address:       TestConfig.Account1.Address,
			expectedError: keystore.ErrNoMatch,
		},
	}

	// test
	for _, testCase := range testCases {
		s.T().Log(testCase.context)

		account, key, err := accountManager.AddressToDecryptedAccount(testCase.address, testCase.password)
		if err != testCase.expectedError {
			require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			require.NotNil(account)
			require.NotNil(key)
		}
	}

}
