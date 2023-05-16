package accounts

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
)

func TestKeycards(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	keycardUID := "00000000000000000000000000000000"

	kp1 := GetProfileKeypairForTest(false)
	keycard1 := GetProfileKeycardForTest()

	kp2 := GetSeedImportedKeypair1ForTest()
	keycard2 := GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"
	keycard2Copy.LastUpdateClock = keycard2Copy.LastUpdateClock + 1

	kp3 := GetSeedImportedKeypair2ForTest()
	keycard3 := GetKeycardForSeedImportedKeypair2ForTest()

	// Pre-condition
	err := db.SaveOrUpdateKeypair(kp1)
	require.NoError(t, err)
	err = db.SaveOrUpdateKeypair(kp2)
	require.NoError(t, err)
	err = db.SaveOrUpdateKeypair(kp3)
	require.NoError(t, err)
	dbKeypairs, err := db.GetKeypairs()
	require.NoError(t, err)
	require.Equal(t, 3, len(dbKeypairs))

	// Test adding key pairs
	addedKc, addedAccs, err := db.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard1)
	require.NoError(t, err)
	require.Equal(t, true, addedKc)
	require.Equal(t, false, addedAccs)
	addedKc, addedAccs, err = db.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2)
	require.NoError(t, err)
	require.Equal(t, true, addedKc)
	require.Equal(t, false, addedAccs)
	addedKc, addedAccs, err = db.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2Copy)
	require.NoError(t, err)
	require.Equal(t, true, addedKc)
	require.Equal(t, false, addedAccs)
	// this should be added
	addedKc, addedAccs, err = db.AddKeycardOrAddAccountsIfKeycardIsAdded(Keycard{
		KeycardUID:        keycard2Copy.KeycardUID,
		AccountsAddresses: []types.Address{{0x03}},
		LastUpdateClock:   keycard2Copy.LastUpdateClock + 1,
	})
	require.NoError(t, err)
	require.Equal(t, false, addedKc)
	require.Equal(t, true, addedAccs)
	// this should not be added as it has clock value less than last updated clock value
	addedKc, addedAccs, err = db.AddKeycardOrAddAccountsIfKeycardIsAdded(Keycard{
		KeycardUID:        keycard2Copy.KeycardUID,
		AccountsAddresses: []types.Address{{0x04}},
		LastUpdateClock:   keycard2Copy.LastUpdateClock,
	})
	require.NoError(t, err)
	require.Equal(t, false, addedKc)
	require.Equal(t, false, addedAccs)
	addedKc, addedAccs, err = db.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard3)
	require.NoError(t, err)
	require.Equal(t, true, addedKc)
	require.Equal(t, false, addedAccs)

	// Test reading migrated key pairs
	rows, err := db.GetAllKnownKeycardsGroupedByKeyUID()
	require.NoError(t, err)
	require.Equal(t, 3, len(rows))
	for _, kp := range rows {
		if kp.KeyUID == keycard1.KeyUID {
			require.Equal(t, keycard1.KeycardUID, kp.KeycardUID)
			require.Equal(t, keycard1.KeycardName, kp.KeycardName)
			require.Equal(t, keycard1.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keycard1.AccountsAddresses), len(kp.AccountsAddresses))
		} else if kp.KeyUID == keycard2.KeyUID { // keycard 2 and 3, cause 3 is a copy of 2
			require.Equal(t, keycard2.KeycardUID, kp.KeycardUID)
			require.Equal(t, keycard2.KeycardName, kp.KeycardName)
			require.Equal(t, keycard2.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keycard2.AccountsAddresses)+1, len(kp.AccountsAddresses)) // Add 1, cause one account is additionally added for the same keycard.
		} else {
			require.Equal(t, keycard3.KeycardUID, kp.KeycardUID)
			require.Equal(t, keycard3.KeycardName, kp.KeycardName)
			require.Equal(t, keycard3.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keycard3.AccountsAddresses), len(kp.AccountsAddresses))
		}
	}

	rows, err = db.GetKeycardByKeyUID(keycard1.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 1, len(rows))
	require.Equal(t, keycard1.KeyUID, rows[0].KeyUID)
	require.Equal(t, keycard1.KeycardUID, rows[0].KeycardUID)
	require.Equal(t, keycard1.KeycardName, rows[0].KeycardName)
	require.Equal(t, keycard1.KeycardLocked, rows[0].KeycardLocked)
	require.Equal(t, len(keycard1.AccountsAddresses), len(rows[0].AccountsAddresses))
	require.Equal(t, keycard1.LastUpdateClock, rows[0].LastUpdateClock)

	rows, err = db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 4, len(rows))
	for _, kp := range rows {
		if kp.KeycardUID == keycard1.KeycardUID {
			require.Equal(t, keycard1.KeycardUID, kp.KeycardUID)
			require.Equal(t, keycard1.KeycardName, kp.KeycardName)
			require.Equal(t, keycard1.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keycard1.AccountsAddresses), len(kp.AccountsAddresses))
			require.Equal(t, keycard1.LastUpdateClock, kp.LastUpdateClock)
		} else if kp.KeycardUID == keycard2.KeycardUID {
			require.Equal(t, keycard2.KeycardUID, kp.KeycardUID)
			require.Equal(t, keycard2.KeycardName, kp.KeycardName)
			require.Equal(t, keycard2.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keycard2.AccountsAddresses), len(kp.AccountsAddresses))
			require.Equal(t, keycard2.LastUpdateClock, kp.LastUpdateClock)
		} else if kp.KeycardUID == keycard2Copy.KeycardUID {
			require.Equal(t, keycard2Copy.KeycardUID, kp.KeycardUID)
			require.Equal(t, keycard2Copy.KeycardName, kp.KeycardName)
			require.Equal(t, keycard2Copy.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keycard2Copy.AccountsAddresses)+1, len(kp.AccountsAddresses)) // Add 1, cause one account is additionally added.
			require.Equal(t, keycard2Copy.LastUpdateClock+1, kp.LastUpdateClock)
		} else {
			require.Equal(t, keycard3.KeycardUID, kp.KeycardUID)
			require.Equal(t, keycard3.KeycardName, kp.KeycardName)
			require.Equal(t, keycard3.KeycardLocked, kp.KeycardLocked)
			require.Equal(t, len(keycard3.AccountsAddresses), len(kp.AccountsAddresses))
			require.Equal(t, keycard3.LastUpdateClock, kp.LastUpdateClock)
		}
	}

	// Test seting a new keycard name
	err = db.SetKeycardName(keycard1.KeycardUID, "Card101", 1000)
	require.NoError(t, err)
	rows, err = db.GetAllKnownKeycardsGroupedByKeyUID()
	require.NoError(t, err)
	newKeycardName := ""
	for _, kp := range rows {
		if kp.KeyUID == keycard1.KeyUID {
			newKeycardName = kp.KeycardName
		}
	}
	require.Equal(t, "Card101", newKeycardName)

	// Test seting a new keycard name with an old clock value
	err = db.SetKeycardName(keycard1.KeycardUID, "Card102", 999) // clock is less than the last one
	require.NoError(t, err)
	rows, err = db.GetAllKnownKeycardsGroupedByKeyUID()
	require.NoError(t, err)
	newKeycardName = ""
	for _, kp := range rows {
		if kp.KeyUID == keycard1.KeyUID {
			newKeycardName = kp.KeycardName
		}
	}
	require.Equal(t, "Card101", newKeycardName)

	// Test locking a keycard
	err = db.KeycardLocked(keycard1.KeycardUID, 1001)
	require.NoError(t, err)
	rows, err = db.GetAllKnownKeycardsGroupedByKeyUID()
	require.NoError(t, err)
	locked := false
	for _, kp := range rows {
		if kp.KeyUID == keycard1.KeyUID {
			locked = kp.KeycardLocked
		}
	}
	require.Equal(t, true, locked)

	// Test detleting accounts (addresses) for a certain keycard
	const numOfAccountsToRemove = 2
	require.Greater(t, len(keycard1.AccountsAddresses), numOfAccountsToRemove)
	accountsToRemove := keycard1.AccountsAddresses[:numOfAccountsToRemove]
	err = db.RemoveMigratedAccountsForKeycard(keycard1.KeycardUID, accountsToRemove, 1002)
	require.NoError(t, err)
	rows, err = db.GetKeycardByKeyUID(keycard1.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 1, len(rows))
	require.Equal(t, len(keycard1.AccountsAddresses)-numOfAccountsToRemove, len(rows[0].AccountsAddresses))

	// Test deleting accounts one by one, with the last deleted account keycard should be delete as well
	for i, addr := range keycard3.AccountsAddresses {
		err = db.RemoveMigratedAccountsForKeycard(keycard3.KeycardUID, []types.Address{addr}, 1003+uint64(i))
		require.NoError(t, err)
	}
	rows, err = db.GetAllKnownKeycardsGroupedByKeyUID()
	require.NoError(t, err)
	// Test if correct keycard is deleted
	deletedKeycard3 := true
	for _, kp := range rows {
		if kp.KeycardUID == keycard3.KeycardUID {
			deletedKeycard3 = false
		}
	}
	require.Equal(t, true, deletedKeycard3)

	// Test update keycard uid
	err = db.UpdateKeycardUID(keycard1.KeycardUID, keycardUID, 1100)
	require.NoError(t, err)

	// Test unlocking a locked keycard
	err = db.KeycardUnlocked(keycardUID, 1101)
	require.NoError(t, err)
	rows, err = db.GetAllKnownKeycardsGroupedByKeyUID()
	require.NoError(t, err)
	locked = true
	for _, kp := range rows {
		if kp.KeycardUID == keycardUID {
			locked = kp.KeycardLocked
		}
	}
	require.Equal(t, false, locked)

	// Test detleting a keycard
	err = db.DeleteKeycard(keycardUID, 1102)
	require.NoError(t, err)
	rows, err = db.GetAllKnownKeycardsGroupedByKeyUID()
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

	// Test detleting a keycard
	err = db.DeleteAllKeycardsWithKeyUID(keycard2.KeyUID)
	require.NoError(t, err)
	rows, err = db.GetAllKnownKeycardsGroupedByKeyUID()
	require.NoError(t, err)
	// Test if correct keycard is deleted
	deletedKeycard2And3 := true
	for _, kp := range rows {
		if kp.KeyUID == keycard2.KeyUID {
			deletedKeycard2And3 = false
		}
	}
	require.Equal(t, true, deletedKeycard2And3)
}

func TestKeycardsRemovalWhenDeletingKeypair(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	kp2 := &Keypair{
		KeyUID:      "0000000000000000000000000000000000000000000000000000000000000002",
		Name:        "Keypair Name 2",
		Type:        KeypairTypeSeed,
		DerivedFrom: "0x0001",
	}
	kp2.Accounts = append(kp2.Accounts, &Account{Address: types.Address{0x11}, KeyUID: kp2.KeyUID})
	kp2.Accounts = append(kp2.Accounts, &Account{Address: types.Address{0x12}, KeyUID: kp2.KeyUID})

	keycard2 := Keycard{
		KeycardUID:        "00000000000000000000000000000002",
		KeycardName:       "Card02",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x11}, {0x12}},
		KeyUID:            kp2.KeyUID,
		LastUpdateClock:   200,
	}
	keycard3 := Keycard{
		KeycardUID:        "00000000000000000000000000000003",
		KeycardName:       "Card02 Copy",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x11}, {0x12}},
		KeyUID:            kp2.KeyUID,
		LastUpdateClock:   300,
	}

	// Pre-condition - save keypair
	err := db.SaveOrUpdateKeypair(kp2)
	require.NoError(t, err)
	dbKeypairs, err := db.GetKeypairs()
	require.NoError(t, err)
	require.Equal(t, 1, len(dbKeypairs))

	// Pre-condition - save keycards referring to previously added keypair
	addedKc, addedAccs, err := db.AddKeycardOrAddAccountsIfKeycardIsAdded(keycard2)
	require.NoError(t, err)
	require.Equal(t, true, addedKc)
	require.Equal(t, false, addedAccs)
	addedKc, addedAccs, err = db.AddKeycardOrAddAccountsIfKeycardIsAdded(keycard3)
	require.NoError(t, err)
	require.Equal(t, true, addedKc)
	require.Equal(t, false, addedAccs)

	// Check db state
	keycardsWithSameKeyUID, err := db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 2, len(keycardsWithSameKeyUID))

	require.Equal(t, len(kp2.KeyUID), len(dbKeypairs[0].KeyUID))
	require.Equal(t, len(kp2.KeyUID), len(keycardsWithSameKeyUID[0].KeyUID))
	require.Equal(t, len(kp2.KeyUID), len(keycardsWithSameKeyUID[1].KeyUID))

	require.Equal(t, len(kp2.Accounts), len(dbKeypairs[0].Accounts))
	require.Equal(t, len(kp2.Accounts), len(keycardsWithSameKeyUID[0].AccountsAddresses))
	require.Equal(t, len(kp2.Accounts), len(keycardsWithSameKeyUID[1].AccountsAddresses))

	// Remove keypair
	err = db.DeleteKeypair(kp2.KeyUID)
	require.NoError(t, err)

	// Check db state after deletion
	dbKeypairs, err = db.GetKeypairs()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbKeypairs))

	keycardsWithSameKeyUID, err = db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 0, len(keycardsWithSameKeyUID))
}
