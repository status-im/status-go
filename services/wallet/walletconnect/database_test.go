package walletconnect

import (
	"strconv"
	"testing"

	"database/sql"

	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

// generateTestData generates alternative disconnected and active pairings starting with the active one
// timestamps start with 1234567890
func generateTestData(count int) []Pairing {
	res := make([]Pairing, count)
	for i := 0; i < count; i++ {
		strI := strconv.Itoa(i)
		res[i] = Pairing{
			Topic:       Topic(strI + "abcdef1234567890"),
			Expiry:      1234567890 + int64(i),
			Active:      (i % 2) == 0,
			AppName:     "TestApp" + strI,
			URL:         "https://test.url/" + strI,
			Description: "Test Description" + strI,
			Icon:        "https://test.icon" + strI,
			Verified: Verified{
				IsScam:     false,
				Origin:     "https://test.origin/" + strI,
				VerifyURL:  "https://test.verify.url/" + strI,
				Validation: "https://test.validation/" + strI,
			},
		}
	}
	return res
}

func insertTestData(t *testing.T, db *sql.DB, entries []Pairing) {
	for _, entry := range entries {
		err := InsertPairing(db, entry)
		require.NoError(t, err)
	}
}

func TestInsertAndGetPairing(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	entry := generateTestData(1)[0]
	err := InsertPairing(db, entry)
	require.NoError(t, err)

	retrievedPairing, err := GetPairingByTopic(db, entry.Topic)
	require.NoError(t, err)

	require.Equal(t, entry, *retrievedPairing)
}

func TestChangePairingState(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	entry := generateTestData(1)[0]
	err := InsertPairing(db, entry)
	require.NoError(t, err)

	err = ChangePairingState(db, entry.Topic, false)
	require.NoError(t, err)

	retrievedPairing, err := GetPairingByTopic(db, entry.Topic)
	require.NoError(t, err)

	require.Equal(t, false, retrievedPairing.Active)
}

func TestGet(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	entries := generateTestData(3)
	insertTestData(t, db, entries)

	retrievedPairing, err := GetPairingByTopic(db, entries[1].Topic)
	require.NoError(t, err)

	require.Equal(t, entries[1], *retrievedPairing)
}

func TestGetActivePairings(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// insert two disconnected and three active pairing
	entries := generateTestData(5)
	insertTestData(t, db, entries)

	activePairings, err := GetActivePairings(db, 1234567892)
	require.NoError(t, err)

	require.Equal(t, 2, len(activePairings))
	// Expect newest on top
	require.Equal(t, entries[4], activePairings[0])
	require.Equal(t, entries[2], activePairings[1])
}

func TestHasActivePairings(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// insert one disconnected and two active pairing
	entries := generateTestData(2)
	insertTestData(t, db, entries)

	hasActivePairings, err := HasActivePairings(db, 1234567890)
	require.NoError(t, err)
	require.True(t, hasActivePairings)

	hasActivePairings, err = HasActivePairings(db, 1234567891)
	require.NoError(t, err)
	require.False(t, hasActivePairings)
}
