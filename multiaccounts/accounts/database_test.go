package accounts

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "settings-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "settings-tests")
	require.NoError(t, err)
	return NewDB(db), func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestConfig(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	conf := params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}
	require.NoError(t, db.SaveConfig("node-config", conf))
	var rst params.NodeConfig
	require.NoError(t, db.GetConfig("node-config", &rst))
	require.Equal(t, conf, rst)
}

func TestConfigBlob(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	tag := "random-param"
	param := 10
	require.NoError(t, db.SaveConfig(tag, param))
	expected, err := json.Marshal(param)
	require.NoError(t, err)
	rst, err := db.GetConfigBlob(tag)
	require.NoError(t, err)
	require.Equal(t, json.RawMessage(expected), rst)
}

func TestGetConfigBlobs(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	expected := map[string]json.RawMessage{
		"tag1": json.RawMessage("1"),
		"tag2": json.RawMessage("2"),
		"tag3": json.RawMessage("3"),
	}
	types := make([]string, 0, len(expected))
	for k, v := range expected {
		require.NoError(t, db.SaveConfig(k, v))
		types = append(types, k)
	}
	rst, err := db.GetConfigBlobs(types)
	require.NoError(t, err)
	require.Equal(t, expected, rst)
}

func TestSaveAccounts(t *testing.T) {
	type testCase struct {
		description string
		accounts    []Account
		err         error
	}
	for _, tc := range []testCase{
		{
			description: "NoError",
			accounts: []Account{
				{Address: common.Address{0x01}, Chat: true, Wallet: true},
				{Address: common.Address{0x02}},
			},
		},
		{
			description: "UniqueChat",
			accounts: []Account{
				{Address: common.Address{0x01}, Chat: true},
				{Address: common.Address{0x02}, Chat: true},
			},
			err: ErrChatNotUnique,
		},
		{
			description: "UniqueWallet",
			accounts: []Account{
				{Address: common.Address{0x01}, Wallet: true},
				{Address: common.Address{0x02}, Wallet: true},
			},
			err: ErrWalletNotUnique,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			db, stop := setupTestDB(t)
			defer stop()
			require.Equal(t, tc.err, db.SaveAccounts(tc.accounts))
		})
	}
}

func TestUpdateAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: common.Address{0x01}, Chat: true, Wallet: true},
		{Address: common.Address{0x02}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	accounts[0].Chat = false
	accounts[1].Chat = true
	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, accounts, rst)
}

func TestDeleteAccount(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: common.Address{0x01}, Chat: true, Wallet: true},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.NoError(t, db.DeleteAccount(common.Address{0x01}))
	rst2, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(rst2))
}

func TestGetAddresses(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: common.Address{0x01}, Chat: true, Wallet: true},
		{Address: common.Address{0x02}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	addresses, err := db.GetAddresses()
	require.NoError(t, err)
	require.Equal(t, []common.Address{{0x01}, {0x02}}, addresses)
}

func TestGetWalletAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := common.Address{0x01}
	_, err := db.GetWalletAddress()
	require.Equal(t, err, sql.ErrNoRows)
	require.NoError(t, db.SaveAccounts([]Account{{Address: address, Wallet: true}}))
	wallet, err := db.GetWalletAddress()
	require.NoError(t, err)
	require.Equal(t, address, wallet)
}

func TestGetChatAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := common.Address{0x01}
	_, err := db.GetChatAddress()
	require.Equal(t, err, sql.ErrNoRows)
	require.NoError(t, db.SaveAccounts([]Account{{Address: address, Chat: true}}))
	chat, err := db.GetChatAddress()
	require.NoError(t, err)
	require.Equal(t, address, chat)
}

func TestGetAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: common.Address{0x01}, Chat: true, Wallet: true},
		{Address: common.Address{0x02}, PublicKey: hexutil.Bytes{0x01, 0x02}},
		{Address: common.Address{0x03}, PublicKey: hexutil.Bytes{0x02, 0x03}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, accounts, rst)
}

func TestAddressExists(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	accounts := []Account{
		{Address: common.Address{0x01}, Chat: true, Wallet: true},
	}
	require.NoError(t, db.SaveAccounts(accounts))

	exists, err := db.AddressExists(accounts[0].Address)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestAddressDoesntExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	exists, err := db.AddressExists(common.Address{1, 1, 1})
	require.NoError(t, err)
	require.False(t, exists)
}
