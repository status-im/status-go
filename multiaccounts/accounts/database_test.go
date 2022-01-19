package accounts

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/errors"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	db, stop, err := appdatabase.SetupTestSQLDB("settings-tests-")
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	d, err := NewDB(db)
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	return d, func() {
		require.NoError(t, stop())
	}
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
				{Address: types.Address{0x01}, Chat: true, Wallet: true},
				{Address: types.Address{0x02}},
			},
		},
		{
			description: "UniqueChat",
			accounts: []Account{
				{Address: types.Address{0x01}, Chat: true},
				{Address: types.Address{0x02}, Chat: true},
			},
			err: errors.ErrChatNotUnique,
		},
		{
			description: "UniqueWallet",
			accounts: []Account{
				{Address: types.Address{0x01}, Wallet: true},
				{Address: types.Address{0x02}, Wallet: true},
			},
			err: errors.ErrWalletNotUnique,
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
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
		{Address: types.Address{0x02}},
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
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.NoError(t, db.DeleteAccount(types.Address{0x01}))
	rst2, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(rst2))
}

func TestGetAddresses(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
		{Address: types.Address{0x02}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	addresses, err := db.GetAddresses()
	require.NoError(t, err)
	require.Equal(t, []types.Address{{0x01}, {0x02}}, addresses)
}

func TestGetWalletAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := types.Address{0x01}
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
	address := types.Address{0x01}
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
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
		{Address: types.Address{0x02}, PublicKey: types.HexBytes{0x01, 0x02}},
		{Address: types.Address{0x03}, PublicKey: types.HexBytes{0x02, 0x03}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, accounts, rst)
}

func TestGetAccountByAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := types.Address{0x01}
	account := Account{Address: address, Chat: true, Wallet: true}
	dilute := []Account{
		{Address: types.Address{0x02}, PublicKey: types.HexBytes{0x01, 0x02}},
		{Address: types.Address{0x03}, PublicKey: types.HexBytes{0x02, 0x03}},
	}

	accounts := append(dilute, account)

	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccountByAddress(address)
	require.NoError(t, err)
	require.Equal(t, &account, rst)
}

func TestAddressExists(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	accounts := []Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
	}
	require.NoError(t, db.SaveAccounts(accounts))

	exists, err := db.AddressExists(accounts[0].Address)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestAddressDoesntExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	exists, err := db.AddressExists(types.Address{1, 1, 1})
	require.NoError(t, err)
	require.False(t, exists)
}
