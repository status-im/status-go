package keypairs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
)

func setupTestDB(t *testing.T) (*KeyPairs, func()) {
	db, stop, err := appdatabase.SetupTestSQLDB("settings-tests-")
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	d := NewKeyPairs(db)

	return d, func() {
		require.NoError(t, stop())
	}
}

func TestKeypairs(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	keycardUID := "00000000000000000000000000000000"
	keyPair1 := KeyPair{
		KeycardUID:        "00000000000000000000000000000001",
		KeycardName:       "Card01",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}, {0x03}, {0x04}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000001",
	}
	keyPair2 := KeyPair{
		KeycardUID:        "00000000000000000000000000000002",
		KeycardName:       "Card02",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000002",
	}
	keyPair3 := KeyPair{
		KeycardUID:        "00000000000000000000000000000003",
		KeycardName:       "Card02 Copy",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000002",
	}
	keyPair4 := KeyPair{
		KeycardUID:        "00000000000000000000000000000004",
		KeycardName:       "Card04",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}, {0x03}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000004",
	}

	// Test adding key pairs
	err := db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair1)
	require.NoError(t, err)
	err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair2)
	require.NoError(t, err)
	err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair3)
	require.NoError(t, err)
	err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(KeyPair{
		KeycardUID:        keyPair3.KeycardUID,
		AccountsAddresses: []types.Address{{0x03}},
	})
	require.NoError(t, err)
	err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair4)
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
