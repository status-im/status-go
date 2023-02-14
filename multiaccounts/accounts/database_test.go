package accounts

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/multiaccounts/keypairs"
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

func TestKeypairs(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	keycardUID := "00000000000000000000000000000000"
	keyPair1 := keypairs.KeyPair{
		KeycardUID:        "00000000000000000000000000000001",
		KeycardName:       "Card01",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}, {0x03}, {0x04}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000001",
	}
	keyPair2 := keypairs.KeyPair{
		KeycardUID:        "00000000000000000000000000000002",
		KeycardName:       "Card02",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000002",
	}
	keyPair3 := keypairs.KeyPair{
		KeycardUID:        "00000000000000000000000000000003",
		KeycardName:       "Card02 Copy",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000002",
	}
	keyPair4 := keypairs.KeyPair{
		KeycardUID:        "00000000000000000000000000000004",
		KeycardName:       "Card04",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}, {0x03}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000004",
	}

	// Test adding key pairs
	err := db.AddMigratedKeyPair(keyPair1.KeycardUID, keyPair1.KeycardName, keyPair1.KeyUID, keyPair1.AccountsAddresses)
	require.NoError(t, err)
	err = db.AddMigratedKeyPair(keyPair2.KeycardUID, keyPair2.KeycardName, keyPair2.KeyUID, keyPair2.AccountsAddresses)
	require.NoError(t, err)
	err = db.AddMigratedKeyPair(keyPair3.KeycardUID, keyPair3.KeycardName, keyPair3.KeyUID, keyPair3.AccountsAddresses)
	require.NoError(t, err)
	err = db.AddMigratedKeyPair(keyPair3.KeycardUID, keyPair3.KeycardName, keyPair3.KeyUID, []types.Address{{0x03}})
	require.NoError(t, err)
	err = db.AddMigratedKeyPair(keyPair4.KeycardUID, keyPair4.KeycardName, keyPair4.KeyUID, keyPair4.AccountsAddresses)
	require.NoError(t, err)

	// Test reading migrated key pairs
	rows, err := db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	require.Equal(t, 3, len(rows))
	for _, kp := range rows {
		if kp.KeyUID == keyPair1.KeyUID {
			require.Equal(t, keyPair1.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair1.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair1.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair1.AccountsAddresses), len(kp.AccountsAddresses))
		} else if kp.KeyUID == keyPair2.KeyUID { // keypair 2 and 3, cause 3 is a copy of 2
			require.Equal(t, keyPair2.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair2.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair2.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair2.AccountsAddresses)+1, len(kp.AccountsAddresses)) // Add 1, cause one account is additionally added for the same keypair.
		} else {
			require.Equal(t, keyPair4.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair4.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair4.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair4.AccountsAddresses), len(kp.AccountsAddresses))
		}
	}

	rows, err = db.GetMigratedKeyPairByKeyUID(keyPair1.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 1, len(rows))
	require.Equal(t, keyPair1.KeyUID, rows[0].KeyUID)
	require.Equal(t, keyPair1.KeycardUID, rows[0].KeycardUID)
	require.Equal(t, keyPair1.KeycardName, rows[0].KeycardName)
	require.Equal(t, keyPair1.KeycardLocked, rows[0].KeycardLocked)
	require.Equal(t, len(keyPair1.AccountsAddresses), len(rows[0].AccountsAddresses))

	rows, err = db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 4, len(rows))
	for _, kp := range rows {
		if kp.KeycardUID == keyPair1.KeycardUID {
			require.Equal(t, keyPair1.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair1.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair1.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair1.AccountsAddresses), len(kp.AccountsAddresses))
		} else if kp.KeycardUID == keyPair2.KeycardUID {
			require.Equal(t, keyPair2.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair2.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair2.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair2.AccountsAddresses), len(kp.AccountsAddresses))
		} else if kp.KeycardUID == keyPair3.KeycardUID {
			require.Equal(t, keyPair3.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair3.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair3.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair3.AccountsAddresses)+1, len(kp.AccountsAddresses)) // Add 1, cause one account is additionally added.
		} else {
			require.Equal(t, keyPair4.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair4.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair4.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair4.AccountsAddresses), len(kp.AccountsAddresses))
		}
	}

	// Test seting a new keycard name
	err = db.SetKeycardName(keyPair1.KeycardUID, "Card101")
	require.NoError(t, err)
	rows, err = db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	newKeycardName := ""
	for _, kp := range rows {
		if kp.KeyUID == keyPair1.KeyUID {
			newKeycardName = kp.KeycardName
		}
	}
	require.Equal(t, "Card101", newKeycardName)

	// Test locking a keycard
	err = db.KeycardLocked(keyPair1.KeycardUID)
	require.NoError(t, err)
	rows, err = db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	locked := false
	for _, kp := range rows {
		if kp.KeyUID == keyPair1.KeyUID {
			locked = kp.KeycardLocked
		}
	}
	require.Equal(t, true, locked)

	// Test detleting accounts (addresses) for a certain keycard
	const numOfAccountsToRemove = 2
	require.Greater(t, len(keyPair1.AccountsAddresses), numOfAccountsToRemove)
	accountsToRemove := keyPair1.AccountsAddresses[:numOfAccountsToRemove]
	err = db.RemoveMigratedAccountsForKeycard(keyPair1.KeycardUID, accountsToRemove)
	require.NoError(t, err)
	rows, err = db.GetMigratedKeyPairByKeyUID(keyPair1.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 1, len(rows))
	require.Equal(t, len(keyPair1.AccountsAddresses)-numOfAccountsToRemove, len(rows[0].AccountsAddresses))

	// Test deleting accounts one by one, with the last deleted account keycard should be delete as well
	for _, addr := range keyPair4.AccountsAddresses {
		err = db.RemoveMigratedAccountsForKeycard(keyPair4.KeycardUID, []types.Address{addr})
		require.NoError(t, err)
	}
	rows, err = db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	// Test if correct keycard is deleted
	deletedKeyPair4 := true
	for _, kp := range rows {
		if kp.KeycardUID == keyPair4.KeycardUID {
			deletedKeyPair4 = false
		}
	}
	require.Equal(t, true, deletedKeyPair4)

	// Test update keycard uid
	err = db.UpdateKeycardUID(keyPair1.KeycardUID, keycardUID)
	require.NoError(t, err)

	// Test unlocking a locked keycard
	err = db.KeycardUnlocked(keycardUID)
	require.NoError(t, err)
	rows, err = db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	locked = true
	for _, kp := range rows {
		if kp.KeycardUID == keycardUID {
			locked = kp.KeycardLocked
		}
	}
	require.Equal(t, false, locked)

	// Test detleting a keycard
	err = db.DeleteKeycard(keycardUID)
	require.NoError(t, err)
	rows, err = db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	require.Equal(t, 1, len(rows))
	// Test if correct keycard is deleted
	deletedKeyCard := true
	for _, kp := range rows {
		if kp.KeycardUID == keycardUID {
			deletedKeyCard = false
		}
	}
	require.Equal(t, true, deletedKeyCard)

	// Test detleting a keypair
	err = db.DeleteKeypair(keyPair2.KeyUID)
	require.NoError(t, err)
	rows, err = db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	// Test if correct keycard is deleted
	deletedKeyPair2And3 := true
	for _, kp := range rows {
		if kp.KeyUID == keyPair2.KeyUID {
			deletedKeyPair2And3 = false
		}
	}
	require.Equal(t, true, deletedKeyPair2And3)
}
