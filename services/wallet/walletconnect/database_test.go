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

// generateTestData generates alternative disconnected and active sessions starting with the active one
// timestamps start with 1234567890
func generateTestData(count int) []DbSession {
	res := make([]DbSession, count)
	j := 0
	for i := 0; i < count; i++ {
		strI := strconv.Itoa(i)
		if i%4 == 0 {
			j++
		}
		strJ := strconv.Itoa(j)
		res[i] = DbSession{
			Topic:           Topic(strI + "aaaaaa1234567890"),
			PairingTopic:    Topic(strJ + "bbbbbb1234567890"),
			Expiry:          1234567890 + int64(i),
			Active:          (i % 2) == 0,
			DappName:        "TestApp" + strI,
			DappURL:         "https://test.url/" + strI,
			DappDescription: "Test Description" + strI,
			DappIcon:        "https://test.icon" + strI,
			DappVerifyURL:   "https://test.verify.url/" + strI,
			DappPublicKey:   strI + "1234567890",
		}
	}
	return res
}

func insertTestData(t *testing.T, db *sql.DB, entries []DbSession) {
	for _, entry := range entries {
		err := UpsertSession(db, entry)
		require.NoError(t, err)
	}
}

func TestInsertUpdateAndGetSession(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	entry := generateTestData(1)[0]
	err := UpsertSession(db, entry)
	require.NoError(t, err)

	retrievedSession, err := GetSessionByTopic(db, entry.Topic)
	require.NoError(t, err)

	require.Equal(t, entry, *retrievedSession)

	entry.Active = false
	entry.Expiry = 1111111111
	err = UpsertSession(db, entry)

	require.NoError(t, err)

	retrievedSession, err = GetSessionByTopic(db, entry.Topic)
	require.NoError(t, err)

	require.Equal(t, entry, *retrievedSession)
}

func TestInsertAndGetSessionsByPairingTopic(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	generatedSessions := generateTestData(10)
	for _, session := range generatedSessions {
		err := UpsertSession(db, session)
		require.NoError(t, err)
	}

	retrievedSessions, err := GetSessionsByPairingTopic(db, generatedSessions[4].Topic)
	require.NoError(t, err)
	require.Equal(t, 0, len(retrievedSessions))

	retrievedSessions, err = GetSessionsByPairingTopic(db, generatedSessions[4].PairingTopic)
	require.NoError(t, err)
	require.Equal(t, 4, len(retrievedSessions))

	for i := 4; i < 8; i++ {
		found := false
		for _, session := range retrievedSessions {
			if session.Topic == generatedSessions[i].Topic {
				found = true
				require.Equal(t, generatedSessions[i], session)
			}
		}
		require.True(t, found)
	}
}

func TestChangeSessionState(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	entry := generateTestData(1)[0]
	err := UpsertSession(db, entry)
	require.NoError(t, err)

	err = ChangeSessionState(db, entry.Topic, false)
	require.NoError(t, err)

	retrievedSession, err := GetSessionByTopic(db, entry.Topic)
	require.NoError(t, err)

	require.Equal(t, false, retrievedSession.Active)
}

func TestGet(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	entries := generateTestData(3)
	insertTestData(t, db, entries)

	retrievedSession, err := GetSessionByTopic(db, entries[1].Topic)
	require.NoError(t, err)

	require.Equal(t, entries[1], *retrievedSession)
}

func TestGetActiveSessions(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// insert two disconnected and three active sessions
	entries := generateTestData(5)
	insertTestData(t, db, entries)

	activeSessions, err := GetActiveSessions(db, 1234567892)
	require.NoError(t, err)

	require.Equal(t, 2, len(activeSessions))
	// Expect newest on top
	require.Equal(t, entries[4], activeSessions[0])
	require.Equal(t, entries[2], activeSessions[1])
}

// func TestHasActivePairings(t *testing.T) {
// 	db, close := setupTestDB(t)
// 	defer close()

// 	// insert one disconnected and two active pairing
// 	entries := generateTestData(2)
// 	insertTestData(t, db, entries)

// 	hasActivePairings, err := HasActivePairings(db, 1234567890)
// 	require.NoError(t, err)
// 	require.True(t, hasActivePairings)

// 	hasActivePairings, err = HasActivePairings(db, 1234567891)
// 	require.NoError(t, err)
// 	require.False(t, hasActivePairings)
// }
