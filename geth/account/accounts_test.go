package account_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/node"
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
	require.IsType(&node.NodeManager{}, s.NodeManager)
}

func (s *AccountsTestSuite) TestVerifyAccountPassword() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	accountManager := account.NewManager(nil)
	require.NotNil(accountManager)

	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(err)
	defer os.RemoveAll(keyStoreDir) // nolint: errcheck

	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "empty")
	require.NoError(err)
	defer os.RemoveAll(emptyKeyStoreDir) // nolint: errcheck

	// import account keys
	require.NoError(common.ImportTestAccount(keyStoreDir, "test-account1.pk"))
	require.NoError(common.ImportTestAccount(keyStoreDir, "test-account2.pk"))

	account1Address := gethcommon.BytesToAddress(gethcommon.FromHex(TestConfig.Account1.Address))

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
		s.T().Log(testCase.name)
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
	keyStoreDir, err := ioutil.TempDir("", "status-accounts-test")
	s.NoError(err)
	defer os.RemoveAll(keyStoreDir)

	// Import keys and make sure one was created before EIP55 introduction.
	err = common.ImportTestAccount(keyStoreDir, "test-account1-before-eip55.pk")
	s.NoError(err)

	acctManager := account.NewManager(nil)

	address := gethcommon.HexToAddress(TestConfig.Account1.Address)
	_, err = acctManager.VerifyAccountPassword(keyStoreDir, address.Hex(), TestConfig.Account1.Password)
	s.NoError(err)
}
