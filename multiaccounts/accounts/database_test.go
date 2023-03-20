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
		accounts    []*Account
		err         error
	}
	for _, tc := range []testCase{
		{
			description: "NoError",
			accounts: []*Account{
				{Address: types.Address{0x01}, Chat: true, Wallet: true},
				{Address: types.Address{0x02}},
			},
		},
		{
			description: "UniqueChat",
			accounts: []*Account{
				{Address: types.Address{0x01}, Chat: true},
				{Address: types.Address{0x02}, Chat: true},
			},
			err: errors.ErrChatNotUnique,
		},
		{
			description: "UniqueWallet",
			accounts: []*Account{
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
	accounts := []*Account{
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
	accounts := []*Account{
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
	accounts := []*Account{
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
	require.NoError(t, db.SaveAccounts([]*Account{{Address: address, Wallet: true}}))
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
	require.NoError(t, db.SaveAccounts([]*Account{{Address: address, Chat: true}}))
	chat, err := db.GetChatAddress()
	require.NoError(t, err)
	require.Equal(t, address, chat)
}

func TestGetAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []*Account{
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
	account := &Account{Address: address, Chat: true, Wallet: true}
	dilute := []*Account{
		{Address: types.Address{0x02}, PublicKey: types.HexBytes{0x01, 0x02}},
		{Address: types.Address{0x03}, PublicKey: types.HexBytes{0x02, 0x03}},
	}

	accounts := append(dilute, account)

	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccountByAddress(address)
	require.NoError(t, err)
	require.Equal(t, account, rst)
}

func TestAddressExists(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	accounts := []*Account{
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

func TestKeypairNameAndIndexWhenAddingNewAccount(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accountsRegular := []*Account{
		// chat account
		{Address: types.Address{0x01}, Chat: true, Wallet: false, KeyUID: "0x0001"},
		// Status Profile keypair
		{Address: types.Address{0x02}, Chat: false, Wallet: true, KeyUID: "0x0001", Path: "m/44'/60'/0'/0/0", LastUsedDerivationIndex: 0, DerivedFrom: "0x1111", KeypairName: "Status Profile"},
		{Address: types.Address{0x03}, Chat: false, Wallet: false, KeyUID: "0x0001", Path: "m/44'/60'/0'/0/1", LastUsedDerivationIndex: 1, DerivedFrom: "0x1111", KeypairName: "Status Profile"},
		{Address: types.Address{0x04}, Chat: false, Wallet: false, KeyUID: "0x0001", Path: "m/44'/60'/0'/0/2", LastUsedDerivationIndex: 2, DerivedFrom: "0x1111", KeypairName: "Status Profile"},
	}
	accountsCustom := []*Account{
		// Keypair1
		{Address: types.Address{0x11}, Chat: false, Wallet: false, KeyUID: "0x0002", Path: "m/44'/60'/0'/0/10", LastUsedDerivationIndex: 0, DerivedFrom: "0x2222", KeypairName: "Keypair11"},
		{Address: types.Address{0x12}, Chat: false, Wallet: false, KeyUID: "0x0002", Path: "m/44'/60'/0'/0/11", LastUsedDerivationIndex: 0, DerivedFrom: "0x2222", KeypairName: "Keypair12"},
		// Keypair2 out of the default Status' derivation tree
		{Address: types.Address{0x22}, Chat: false, Wallet: false, KeyUID: "0x0003", Path: "m/44'/60'/0'/0/0/100", LastUsedDerivationIndex: 0, DerivedFrom: "0x3333", KeypairName: "Keypair21"},
		{Address: types.Address{0x23}, Chat: false, Wallet: false, KeyUID: "0x0003", Path: "m/44'/60'/0'/0/1/100", LastUsedDerivationIndex: 0, DerivedFrom: "0x3333", KeypairName: "Keypair22"},
	}

	err := db.SaveAccounts(accountsRegular)
	require.NoError(t, err)
	err = db.SaveAccounts(accountsCustom)
	require.NoError(t, err)
	accs, err := db.GetAccounts()
	require.NoError(t, err)

	for _, acc := range accs {
		if acc.Chat {
			continue
		}
		if acc.KeyUID == accountsRegular[0].KeyUID {
			require.Equal(t, uint64(2), acc.LastUsedDerivationIndex)
			require.Equal(t, "Status Profile", acc.KeypairName)
		} else if acc.KeyUID == accountsCustom[1].KeyUID {
			require.Equal(t, uint64(0), acc.LastUsedDerivationIndex)
			require.Equal(t, "Keypair12", acc.KeypairName)
		} else if acc.KeyUID == accountsCustom[3].KeyUID {
			require.Equal(t, uint64(0), acc.LastUsedDerivationIndex)
			require.Equal(t, "Keypair22", acc.KeypairName)
		}
	}

	accountsCustom = []*Account{
		// Status Profile keypair
		{Address: types.Address{0x05}, Chat: false, Wallet: false, KeyUID: "0x0001", Path: "m/44'/60'/0'/0/100/1", LastUsedDerivationIndex: 2, DerivedFrom: "0x1111", KeypairName: "Status Profile"},
	}

	err = db.SaveAccounts(accountsCustom)
	require.NoError(t, err)

	result, err := db.GetAccountsByKeyUID(accountsCustom[0].KeyUID)
	require.NoError(t, err)
	require.Equal(t, 5, len(result))
	require.Equal(t, uint64(2), accountsCustom[0].LastUsedDerivationIndex)
	require.Equal(t, "Status Profile", accountsCustom[0].KeypairName)

	accountsRegular = []*Account{
		// Status Profile keypair
		{Address: types.Address{0x06}, Chat: false, Wallet: false, KeyUID: "0x0001", Path: "m/44'/60'/0'/0/3", LastUsedDerivationIndex: 3, DerivedFrom: "0x1111", KeypairName: "Status Profile"},
	}

	err = db.SaveAccounts(accountsRegular)
	require.NoError(t, err)

	result, err = db.GetAccountsByKeyUID(accountsCustom[0].KeyUID)
	require.NoError(t, err)
	require.Equal(t, 6, len(result))
	require.Equal(t, uint64(3), accountsRegular[0].LastUsedDerivationIndex)
	require.Equal(t, "Status Profile", accountsRegular[0].KeypairName)
}
