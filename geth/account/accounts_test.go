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
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(AccountsTestSuite))
}

type AccountsTestSuite struct {
	BaseTestSuite
}

func (s *AccountsTestSuite) SetupTest() {
	require := s.Require()
	s.NodeManager = node.NewNodeManager()
	require.NotNil(s.NodeManager)
}

func (s *AccountsTestSuite) TestCreateAccount() {
	require := s.Require()

	// init node config
	config, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)
	require.NotNil(config)

	// init empty keyStore
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "empty")
	require.NoError(err)
	require.NotNil(emptyKeyStoreDir)
	config.KeyStoreDir = emptyKeyStoreDir

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// test cases
	testCases := []struct {
		context       string
		launchNode    bool
		password      string
		expectedError error
	}{
		{
			context:       "node stopped",
			launchNode:    false,
			expectedError: node.ErrNoRunningNode,
		},
		{
			"node started, non-empty password",
			true,
			"password",
			nil,
		},
		{
			"node started, empty password",
			true,
			"",
			nil,
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
					require.NoError(os.RemoveAll(emptyKeyStoreDir))
				}()
			}

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

	// init node config
	config, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)
	require.NotNil(config)

	// init empty keyStore
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "status-temp")
	require.NoError(err)
	require.NotEmpty(emptyKeyStoreDir)
	defer func() {
		require.NoError(os.RemoveAll(emptyKeyStoreDir))
	}()
	config.KeyStoreDir = emptyKeyStoreDir

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// import account key
	require.NoError(common.ImportTestAccount(emptyKeyStoreDir, "test-account1.pk"))

	// test cases
	testCases := []struct {
		context         string
		launchNode      bool
		selectedAccount *common.SelectedExtKey
		parentAddress   string
		password        string
		expectedError   error
	}{
		{
			context:       "node stopped",
			launchNode:    false,
			expectedError: node.ErrNoRunningNode,
		},
		{
			"node started, parent address is empty, no default account",
			true,
			nil,
			"",
			"password",
			account.ErrNoAccountSelected,
		},
		{
			"node started, parent address is not empty, incorrect address format ",
			true,
			nil,
			"address: invalid format",
			"password",
			account.ErrAddressToAccountMappingFailure,
		},
		/*
					{
						"node started, correct parent address, incorrect password",
						true,
						nil,
						TestConfig.Account1.Address,
						"password: invalid",
						nil,
					},

				{
					"node started, correct parent address, correct password",
					true,
					nil,
					TestConfig.Account1.Address,
					TestConfig.Account1.Password,
					nil,
				},
				{
				"node started, parent address empty, selected account not empty, incorrect password",
				true,
				nil, // to be changed
				"",
				"password: incorrect",
				nil,
			},
			{
				"node started, parent address empty, selected account not empty, correct password",
				true,
				nil, // to be changed
				"",
				TestConfig.Account1.Password,
				account.ErrAddressToAccountMappingFailure,
			},
		*/

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

			address, pubKey, err := accountManager.CreateChildAccount(testCase.parentAddress, testCase.password)
			if err != testCase.expectedError {
				s.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
			}
			if err == nil {
				require.NotEmpty(address)
				require.NotEmpty(pubKey)
			}
		}()
	}
}

/*

func (s *AccountsTestSuite) TestRecoverAccount() {
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

	// start node
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

	// account manager
	accountManager := account.NewManager(s.NodeManager)
	require.NotNil(accountManager)

	// create an account to be recovered later on
	password := "status"
	address, pubkey, mnemonic, err := accountManager.CreateAccount(password)
	require.NotEmpty(address)
	require.NotEmpty(pubkey)
	require.NotEmpty(mnemonic)
	require.NoError(err)

	testCases := []struct {
		context         string
		password        string
		mnemonic        string
		expectedAddress string
		expectedPubkey  string
		expectedError   error
	}{
		{
			"correct mnemonic, correct password",
			password,
			mnemonic,
			address,
			pubkey,
			nil,
		},
		{
			"correct mnemonic, incorrect password",
			"incorrectpassword",
			mnemonic,
			"",
			"",
			account.ErrInvalidMasterKeyCreated,
		},
	}

	for _, test := range testCases {
		address, key, err := accountManager.RecoverAccount(test.password, test.mnemonic)
		if !reflect.DeepEqual(err, test.expectedError) {
			require.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", test.expectedError, err))
		}
		if err == nil {
			require.NotEmpty(address)
			require.NotEmpty(key)
		}
	}
}

*/

func (s *AccountsTestSuite) TestVerifyAccountPassword() {
	require := s.Require()

	// init key store
	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(err)
	require.NotEmpty(keyStoreDir)
	defer os.RemoveAll(keyStoreDir) // nolint: errcheck

	// init empty key store
	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "empty")
	require.NoError(err)
	require.NotEmpty(emptyKeyStoreDir)
	defer os.RemoveAll(emptyKeyStoreDir) // nolint: errcheck

	// import account keys
	require.NoError(common.ImportTestAccount(keyStoreDir, "test-account1.pk"))
	require.NoError(common.ImportTestAccount(keyStoreDir, "test-account2.pk"))

	// account manager
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
			"correct address, correct password (decrypt should succeed)",
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
			"wrong password", // wrong password
			errors.New("could not decrypt key with given passphrase"),
		},
	}
	for _, testCase := range testCases {
		s.T().Log(testCase.context)
		accountKey, err := accountManager.VerifyAccountPassword(testCase.keyPath, testCase.address, testCase.password)
		if !reflect.DeepEqual(err, testCase.expectedError) {
			s.FailNow(fmt.Sprintf("unexpected error: expected \n'%v', got \n'%v'", testCase.expectedError, err))
		}
		if err == nil {
			if accountKey == nil {
				s.T().Error("no error reported, but account key is missing")
			}
			accountAddress := gethcommon.BytesToAddress(gethcommon.FromHex(testCase.address))
			if accountKey.Address != accountAddress {
				s.T().Fatalf("account mismatch: have %s, want %s", accountKey.Address.Hex(), accountAddress.Hex())
			}
		}
	}
}

// TestVerifyAccountPasswordWithAccountBeforeEIP55 verifies if VerifyAccountPassword
// can handle accounts before introduction of EIP55.
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
