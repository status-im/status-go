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
		LastUpdateClock:   100,
	}
	keyPair2 := KeyPair{
		KeycardUID:        "00000000000000000000000000000002",
		KeycardName:       "Card02",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000002",
		LastUpdateClock:   200,
	}
	keyPair3 := KeyPair{
		KeycardUID:        "00000000000000000000000000000003",
		KeycardName:       "Card02 Copy",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000002",
		LastUpdateClock:   300,
	}
	keyPair4 := KeyPair{
		KeycardUID:        "00000000000000000000000000000004",
		KeycardName:       "Card04",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}, {0x03}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000004",
		LastUpdateClock:   400,
	}

	// Test adding key pairs
	result, err := db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair1)
	require.NoError(t, err)
	require.Equal(t, true, result)
	result, err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair2)
	require.NoError(t, err)
	require.Equal(t, true, result)
	result, err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair3)
	require.NoError(t, err)
	require.Equal(t, true, result)
	// this should be added
	result, err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(KeyPair{
		KeycardUID:        keyPair3.KeycardUID,
		AccountsAddresses: []types.Address{{0x03}},
		LastUpdateClock:   keyPair3.LastUpdateClock + 1,
	})
	require.NoError(t, err)
	require.Equal(t, true, result)
	// this should not be added as it has clock value less than last updated clock value
	result, err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(KeyPair{
		KeycardUID:        keyPair3.KeycardUID,
		AccountsAddresses: []types.Address{{0x04}},
		LastUpdateClock:   keyPair3.LastUpdateClock,
	})
	require.NoError(t, err)
	require.Equal(t, false, result)
	result, err = db.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair4)
	require.NoError(t, err)
	require.Equal(t, true, result)

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
	require.Equal(t, keyPair1.LastUpdateClock, rows[0].LastUpdateClock)

	rows, err = db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 4, len(rows))
	for _, kp := range rows {
		if kp.KeycardUID == keyPair1.KeycardUID {
			require.Equal(t, keyPair1.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair1.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair1.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair1.AccountsAddresses), len(kp.AccountsAddresses))
			require.Equal(t, keyPair1.LastUpdateClock, kp.LastUpdateClock)
		} else if kp.KeycardUID == keyPair2.KeycardUID {
			require.Equal(t, keyPair2.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair2.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair2.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair2.AccountsAddresses), len(kp.AccountsAddresses))
			require.Equal(t, keyPair2.LastUpdateClock, kp.LastUpdateClock)
		} else if kp.KeycardUID == keyPair3.KeycardUID {
			require.Equal(t, keyPair3.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair3.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair3.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair3.AccountsAddresses)+1, len(kp.AccountsAddresses)) // Add 1, cause one account is additionally added.
			require.Equal(t, keyPair3.LastUpdateClock+1, kp.LastUpdateClock)
		} else {
			require.Equal(t, keyPair4.KeycardUID, kp.KeycardUID)
			require.Equal(t, keyPair4.KeycardName, kp.KeycardName)
			require.Equal(t, keyPair4.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keyPair4.AccountsAddresses), len(kp.AccountsAddresses))
			require.Equal(t, keyPair4.LastUpdateClock, kp.LastUpdateClock)
		}
	}

	// Test seting a new keycard name
	result, err = db.SetKeycardName(keyPair1.KeycardUID, "Card101", 1000)
	require.NoError(t, err)
	require.Equal(t, true, result)
	rows, err = db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	newKeycardName := ""
	for _, kp := range rows {
		if kp.KeyUID == keyPair1.KeyUID {
			newKeycardName = kp.KeycardName
		}
	}
	require.Equal(t, "Card101", newKeycardName)

	// Test seting a new keycard name with an old clock value
	result, err = db.SetKeycardName(keyPair1.KeycardUID, "Card102", 999) // clock is less than the last one
	require.NoError(t, err)
	require.Equal(t, false, result)
	rows, err = db.GetAllMigratedKeyPairs()
	require.NoError(t, err)
	newKeycardName = ""
	for _, kp := range rows {
		if kp.KeyUID == keyPair1.KeyUID {
			newKeycardName = kp.KeycardName
		}
	}
	require.Equal(t, "Card101", newKeycardName)

	// Test locking a keycard
	result, err = db.KeycardLocked(keyPair1.KeycardUID, 1001)
	require.NoError(t, err)
	require.Equal(t, true, result)
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
	result, err = db.RemoveMigratedAccountsForKeycard(keyPair1.KeycardUID, accountsToRemove, 1002)
	require.NoError(t, err)
	require.Equal(t, true, result)
	rows, err = db.GetMigratedKeyPairByKeyUID(keyPair1.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 1, len(rows))
	require.Equal(t, len(keyPair1.AccountsAddresses)-numOfAccountsToRemove, len(rows[0].AccountsAddresses))

	// Test deleting accounts one by one, with the last deleted account keycard should be delete as well
	for i, addr := range keyPair4.AccountsAddresses {
		result, err = db.RemoveMigratedAccountsForKeycard(keyPair4.KeycardUID, []types.Address{addr}, 1003+uint64(i))
		require.NoError(t, err)
		require.Equal(t, true, result)
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
	result, err = db.UpdateKeycardUID(keyPair1.KeycardUID, keycardUID, 1100)
	require.NoError(t, err)
	require.Equal(t, true, result)

	// Test unlocking a locked keycard
	result, err = db.KeycardUnlocked(keycardUID, 1101)
	require.NoError(t, err)
	require.Equal(t, true, result)
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
	result, err = db.DeleteKeycard(keycardUID, 1102)
	require.NoError(t, err)
	require.Equal(t, true, result)
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
