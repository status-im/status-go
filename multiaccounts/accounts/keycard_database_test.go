package accounts

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
)

func TestKeycards(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	kp1 := GetProfileKeypairForTest(true, true, true)
	keycard1 := GetProfileKeycardForTest()

	kp2 := GetSeedImportedKeypair1ForTest()
	keycard2 := GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"
	keycard2Copy.Position = keycard2Copy.Position + 1

	// Pre-condition
	err := db.SaveOrUpdateKeypair(kp1)
	require.NoError(t, err)
	err = db.SaveOrUpdateKeypair(kp2)
	require.NoError(t, err)
	dbKeypairs, err := db.GetKeypairs()
	require.NoError(t, err)
	require.Equal(t, 2, len(dbKeypairs))

	// Test adding/reading keycards
	err = db.SaveOrUpdateKeycard(*keycard1, 0, false)
	require.NoError(t, err)
	dbKeycard1, err := db.GetKeycardByKeycardUID(keycard1.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard1, dbKeycard1))

	err = db.SaveOrUpdateKeycard(*keycard2, 0, false)
	require.NoError(t, err)
	dbKeycard2, err := db.GetKeycardByKeycardUID(keycard2.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard2, dbKeycard2))

	err = db.SaveOrUpdateKeycard(*keycard2Copy, 0, false)
	require.NoError(t, err)
	dbKeycard2Copy, err := db.GetKeycardByKeycardUID(keycard2Copy.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard2Copy, dbKeycard2Copy))

	dbKeycards, err := db.GetKeycardsWithSameKeyUID(keycard2.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 2, len(dbKeycards))
	require.True(t, Contains(dbKeycards, keycard2, SameKeycards))
	require.True(t, Contains(dbKeycards, keycard2Copy, SameKeycards))

	dbKeycards, err = db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 3, len(dbKeycards))
	require.True(t, Contains(dbKeycards, keycard1, SameKeycards))
	require.True(t, Contains(dbKeycards, keycard2, SameKeycards))
	require.True(t, Contains(dbKeycards, keycard2Copy, SameKeycards))

	nextPosition, err := db.GetPositionForNextNewKeycard()
	require.NoError(t, err)
	require.Equal(t, uint64(len(dbKeycards)), nextPosition)

	// test adding additional accounts to keycard
	keycard1.AccountsAddresses = append(keycard1.AccountsAddresses, types.Address{0x05}, types.Address{0x06})
	err = db.SaveOrUpdateKeycard(*keycard1, 0, false)
	require.NoError(t, err)
	dbKeycard1, err = db.GetKeycardByKeycardUID(keycard1.KeycardUID)
	require.NoError(t, err)
	require.Equal(t, len(keycard1.AccountsAddresses), len(dbKeycard1.AccountsAddresses))
	require.True(t, SameKeycards(keycard1, dbKeycard1))

	// Test seting a new keycard name
	keycard1.KeycardName = "Card101"
	err = db.SetKeycardName(keycard1.KeycardUID, keycard1.KeycardName, 1000)
	require.NoError(t, err)
	dbKeycard1, err = db.GetKeycardByKeycardUID(keycard1.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard1, dbKeycard1))

	// Test locking a keycard
	keycard1.KeycardLocked = true
	err = db.KeycardLocked(keycard1.KeycardUID, 1001)
	require.NoError(t, err)
	dbKeycard1, err = db.GetKeycardByKeycardUID(keycard1.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard1, dbKeycard1))

	// Test unlocking a keycard
	keycard1.KeycardLocked = false
	err = db.KeycardUnlocked(keycard1.KeycardUID, 1002)
	require.NoError(t, err)
	dbKeycard1, err = db.GetKeycardByKeycardUID(keycard1.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard1, dbKeycard1))

	// Test update keycard uid
	oldKeycardUID := keycard1.KeycardUID
	keycard1.KeycardUID = "00000000000000000000000000000000"
	err = db.UpdateKeycardUID(oldKeycardUID, keycard1.KeycardUID, 1003)
	require.NoError(t, err)
	dbKeycard1, err = db.GetKeycardByKeycardUID(keycard1.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard1, dbKeycard1))

	// Test detleting accounts (addresses) for a certain keycard
	const numOfAccountsToRemove = 2
	require.Greater(t, len(keycard1.AccountsAddresses), numOfAccountsToRemove)
	accountsToRemove := keycard1.AccountsAddresses[:numOfAccountsToRemove]
	keycard1.AccountsAddresses = keycard1.AccountsAddresses[numOfAccountsToRemove:]
	err = db.DeleteKeycardAccounts(keycard1.KeycardUID, accountsToRemove, 1004)
	require.NoError(t, err)
	dbKeycard1, err = db.GetKeycardByKeycardUID(keycard1.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard1, dbKeycard1))

	// Test detleting a keycard
	err = db.DeleteKeycard(keycard1.KeycardUID, 1006)
	require.NoError(t, err)
	dbKeycards, err = db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 2, len(dbKeycards))
	dbKeycards, err = db.GetKeycardsWithSameKeyUID(keycard1.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 0, len(dbKeycards))
	dbKeycard, err := db.GetKeycardByKeycardUID(keycard1.KeycardUID)
	require.Error(t, err)
	require.True(t, err == ErrNoKeycardForPassedKeycardUID)
	require.Nil(t, dbKeycard)

	// Test detleting all keycards for KeyUID
	dbKeycards, err = db.GetKeycardsWithSameKeyUID(keycard2.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 2, len(dbKeycards))
	err = db.DeleteAllKeycardsWithKeyUID(keycard2.KeyUID, 1007)
	require.NoError(t, err)
	dbKeycards, err = db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbKeycards))
	dbKeycards, err = db.GetKeycardsWithSameKeyUID(keycard2.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 0, len(dbKeycards))
	dbKeycard, err = db.GetKeycardByKeycardUID(keycard2.KeycardUID)
	require.Error(t, err)
	require.True(t, err == ErrNoKeycardForPassedKeycardUID)
	require.Nil(t, dbKeycard)
}

func TestKeycardsRemovalWhenDeletingKeypair(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	kp2 := GetSeedImportedKeypair1ForTest()
	keycard2 := GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"
	keycard2Copy.Position = keycard2Copy.Position + 1

	// Pre-condition
	err := db.SaveOrUpdateKeypair(kp2)
	require.NoError(t, err)
	dbKeypairs, err := db.GetKeypairs()
	require.NoError(t, err)
	require.Equal(t, 1, len(dbKeypairs))

	// Pre-condition - save keycards referring to previously added keypair
	err = db.SaveOrUpdateKeycard(*keycard2, 0, false)
	require.NoError(t, err)
	dbKeycard2, err := db.GetKeycardByKeycardUID(keycard2.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard2, dbKeycard2))

	err = db.SaveOrUpdateKeycard(*keycard2Copy, 0, false)
	require.NoError(t, err)
	dbKeycard2Copy, err := db.GetKeycardByKeycardUID(keycard2Copy.KeycardUID)
	require.NoError(t, err)
	require.True(t, SameKeycards(keycard2Copy, dbKeycard2Copy))

	// Check db state
	dbKeycards, err := db.GetKeycardsWithSameKeyUID(keycard2.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 2, len(dbKeycards))
	require.True(t, Contains(dbKeycards, keycard2, SameKeycards))
	require.True(t, Contains(dbKeycards, keycard2Copy, SameKeycards))

	// Remove keypair
	err = db.DeleteKeypair(kp2.KeyUID)
	require.NoError(t, err)

	// Check db state after deletion
	dbKeypairs, err = db.GetKeypairs()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbKeypairs))

	dbKeycards, err = db.GetAllKnownKeycards()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbKeycards))
	dbKeycards, err = db.GetKeycardsWithSameKeyUID(kp2.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 0, len(dbKeycards))
	dbKeycard, err := db.GetKeycardByKeycardUID(keycard2.KeycardUID)
	require.Error(t, err)
	require.True(t, err == ErrNoKeycardForPassedKeycardUID)
	require.Nil(t, dbKeycard)
}
