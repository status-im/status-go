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
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/require"
)

func TestVerifyAccountPassword(t *testing.T) {
	acctManager := account.NewManager(nil)
	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(t, err)
	defer os.RemoveAll(keyStoreDir) //nolint: errcheck

	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts_empty")
	require.NoError(t, err)
	defer os.RemoveAll(emptyKeyStoreDir) //nolint: errcheck

	// import account keys
	require.NoError(t, common.ImportTestAccount(keyStoreDir, GetAccount1PKFile()))
	require.NoError(t, common.ImportTestAccount(keyStoreDir, GetAccount2PKFile()))

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
		accountKey, err := acctManager.VerifyAccountPassword(testCase.keyPath, testCase.address, testCase.password)
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
	err = common.ImportTestAccount(keyStoreDir, "test-account3-before-eip55.pk")
	require.NoError(t, err)

	acctManager := account.NewManager(nil)

	address := gethcommon.HexToAddress(TestConfig.Account3.Address)
	_, err = acctManager.VerifyAccountPassword(keyStoreDir, address.Hex(), TestConfig.Account3.Password)
	require.NoError(t, err)
}

func TestCreateAndRecoverAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	nodeManager := common.NewMockNodeManager(ctrl)

	// Create an arbitrary keystore for the process
	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(t, err)
	defer os.RemoveAll(keyStoreDir) //nolint: errcheck
	keyStore := keystore.NewKeyStore(keyStoreDir, 4, 5)

	manager := account.NewManager(nodeManager)
	password := "some-pass"

	// Fail if account keystore can't be acquired
	errString := "Non-nil error string"
	nodeManager.EXPECT().AccountKeyStore().Return(nil, errors.New(errString))
	_, _, _, err = manager.CreateAccount(password)
	require.Equal(t, err.Error(), errString)

	nodeManager.EXPECT().AccountKeyStore().Return(keyStore, nil)
	addr1, pubKey1, mnemonic, err := manager.CreateAccount(password)
	require.NoError(t, err)

	// Expect non-nil return values
	require.NotNil(t, addr1)
	require.NotNil(t, pubKey1)
	require.NotNil(t, mnemonic)

	// Now recover the account using the mnemonic seed and the password
	nodeManager.EXPECT().AccountKeyStore().Return(keyStore, nil)
	addr2, pubKey2, err := manager.RecoverAccount(password, mnemonic)
	require.NoError(t, err)

	// Expect the same return values
	require.Equal(t, addr1, addr2)
	require.Equal(t, pubKey1, pubKey2)

	// Fail if account keystore can't be acquired
	nodeManager.EXPECT().AccountKeyStore().Return(nil, errors.New(errString))
	_, _, err = manager.RecoverAccount(password, mnemonic)
	require.Equal(t, err.Error(), errString)
}
