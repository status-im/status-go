package account

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/common"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/require"
)

func TestVerifyAccountPassword(t *testing.T) {
	acctManager := NewManager(nil)
	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(t, err)
	defer os.RemoveAll(keyStoreDir)

	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts_empty")
	require.NoError(t, err)
	defer os.RemoveAll(emptyKeyStoreDir)

	// import account keys
	require.NoError(t, common.ImportTestAccount(keyStoreDir, "test-account1.pk"))
	require.NoError(t, common.ImportTestAccount(keyStoreDir, "test-account2.pk"))

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
			fmt.Errorf("cannot locate account for address: %x", account1Address),
		},
		{
			"wrong address, correct password",
			keyStoreDir,
			"0x79791d3e8f2daa1f7fec29649d152c0ada3cc535",
			TestConfig.Account1.Password,
			fmt.Errorf("cannot locate account for address: %s", "79791d3e8f2daa1f7fec29649d152c0ada3cc535"),
		},
		{
			"correct address, wrong password",
			keyStoreDir,
			TestConfig.Account1.Address,
			"wrong password", // wrong password
			errors.New("could not decrypt key with given passphrase"),
		},
	}
	for _, tc := range testCases {
		t.Log(tc.name)
		accountKey, err := acctManager.VerifyAccountPassword(tc.keyPath, tc.address, tc.password)
		if tc.expectedError == nil {
			require.Nil(t, err, "err should be nil")
			require.NotNil(t, accountKey, "accountKey can not be nil if there was no error")
			accountAddress := gethcommon.BytesToAddress(gethcommon.FromHex(tc.address))
			require.Equal(t, accountAddress, accountKey.Address)
		} else {
			require.EqualError(t, err, tc.expectedError.Error())
		}
	}
}
