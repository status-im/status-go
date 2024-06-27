package walletconnect

import (
	"strconv"
	"testing"

	"database/sql"

	"github.com/status-im/status-go/services/wallet/common"

	"github.com/stretchr/testify/require"
)

type urlOverride *string
type timestampOverride *int64

// testSession will override defaults for the fields that are not null
type testSession struct {
	url          urlOverride
	created      timestampOverride
	expiry       timestampOverride
	disconnected *bool
	testChains   *bool
}

const testDappUrl = "https://test.url/"

// generateTestData generates alternative disconnected and active sessions starting with the active one
// timestamps start with 1234567890 and increase by 1 for each session
// all sessions will share the same two pairing sessions (roll over after index 1)
// testChains is false if not overridden
func generateTestData(sessions []testSession) []DBSession {
	res := make([]DBSession, len(sessions))
	pairingIdx := 0
	for i := 0; i < len(res); i++ {
		strI := strconv.Itoa(i)
		if i%2 == 0 {
			pairingIdx++
		}
		pairingIdxStr := strconv.Itoa(pairingIdx)

		s := sessions[i]

		url := testDappUrl + strI
		if s.url != nil {
			url = *s.url
		}

		createdTimestamp := 1234567890 + int64(i)
		if s.created != nil {
			createdTimestamp = *s.created
		}

		expiryTimestamp := createdTimestamp + 1000 + int64(i)
		if s.expiry != nil {
			expiryTimestamp = *s.expiry
		}

		disconnected := (i % 2) != 0
		if s.disconnected != nil {
			disconnected = *s.disconnected
		}

		testChains := false
		if s.testChains != nil {
			testChains = *s.testChains
		}

		res[i] = DBSession{
			Topic:            Topic(strI + "aaaaaa1234567890"),
			Disconnected:     disconnected,
			SessionJSON:      "{}",
			Expiry:           expiryTimestamp,
			CreatedTimestamp: createdTimestamp,
			PairingTopic:     Topic(pairingIdxStr + "bbbbbb1234567890"),
			TestChains:       testChains,
			DBDApp: DBDApp{
				URL:     url,
				Name:    "TestApp" + strI,
				IconURL: "https://test.icon" + strI,
			},
		}
	}
	return res
}

func insertTestData(t *testing.T, db *sql.DB, entries []DBSession) {
	for _, entry := range entries {
		err := UpsertSession(db, entry)
		require.NoError(t, err)
	}
}

func TestInsertUpdateAndGetSession(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	entry := generateTestData(make([]testSession, 1))[0]
	err := UpsertSession(db, entry)
	require.NoError(t, err)

	retrievedSession, err := GetSessionByTopic(db, entry.Topic)
	require.NoError(t, err)

	require.Equal(t, entry, *retrievedSession)

	updatedEntry := entry
	updatedEntry.Disconnected = true
	updatedEntry.Expiry = 1111111111
	err = UpsertSession(db, updatedEntry)

	require.NoError(t, err)

	retrievedSession, err = GetSessionByTopic(db, updatedEntry.Topic)
	require.NoError(t, err)

	require.Equal(t, updatedEntry, *retrievedSession)
}

func TestInsertAndGetSessionsByPairingTopic(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	generatedSessions := generateTestData(make([]testSession, 4))
	for _, session := range generatedSessions {
		err := UpsertSession(db, session)
		require.NoError(t, err)
	}

	retrievedSessions, err := GetSessionsByPairingTopic(db, generatedSessions[2].Topic)
	require.NoError(t, err)
	require.Equal(t, 0, len(retrievedSessions))

	retrievedSessions, err = GetSessionsByPairingTopic(db, generatedSessions[2].PairingTopic)
	require.NoError(t, err)
	require.Equal(t, 2, len(retrievedSessions))

	for i := 2; i < 4; i++ {
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

func TestGet(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	entries := generateTestData(make([]testSession, 3))
	insertTestData(t, db, entries)

	retrievedSession, err := GetSessionByTopic(db, entries[1].Topic)
	require.NoError(t, err)

	require.Equal(t, entries[1], *retrievedSession)

	err = DeleteSession(db, entries[1].Topic)
	require.NoError(t, err)

	deletedSession, err := GetSessionByTopic(db, entries[1].Topic)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Nil(t, deletedSession)
}

func TestGetActiveSessions(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	// insert two disconnected and three active sessions
	entries := generateTestData(make([]testSession, 5))
	insertTestData(t, db, entries)

	activeSessions, err := GetActiveSessions(db, entries[2].Expiry)
	require.NoError(t, err)

	require.Equal(t, 2, len(activeSessions))
	// Expect newest on top
	require.Equal(t, entries[4], activeSessions[0])
	require.Equal(t, entries[2], activeSessions[1])
}

func TestDeleteSession(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	entries := generateTestData(make([]testSession, 3))
	insertTestData(t, db, entries)

	err := DeleteSession(db, entries[1].Topic)
	require.NoError(t, err)

	sessions, err := GetSessions(db)
	require.NoError(t, err)
	require.Equal(t, 2, len(sessions))

	require.Equal(t, entries[0], sessions[1])
	require.Equal(t, entries[2], sessions[0])

	err = DeleteSession(db, entries[0].Topic)
	require.NoError(t, err)
	err = DeleteSession(db, entries[2].Topic)
	require.NoError(t, err)

	sessions, err = GetSessions(db)
	require.NoError(t, err)
	require.Equal(t, 0, len(sessions))
}

// urlFor prepares a value to be used in testSession
func urlFor(i int) urlOverride {
	return common.NewAndSet(testDappUrl + strconv.Itoa(i))
}

// at prepares a value to be used in testSession
func at(i int) timestampOverride {
	return common.NewAndSet(int64(i))
}

// TestGetActiveDapps_JoinWorksAsExpected also validates that GetActiveDapps returns the dapps in the order of the last first time added
func TestGetActiveDapps_JoinWorksAsExpected(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	not := common.NewAndSet(false)
	// The first creation date is 1, 2, 3 but the last name update is, respectively, 1, 4, 5
	entries := generateTestData([]testSession{
		{url: urlFor(1), created: at(1), disconnected: not},
		{url: urlFor(1), created: at(2), disconnected: not},
		{url: urlFor(2), created: at(3), disconnected: not},
		{url: urlFor(3), created: at(4), disconnected: not},
		{url: urlFor(2), created: at(5), disconnected: not},
		{url: urlFor(3), created: at(6), disconnected: not},
	})
	insertTestData(t, db, entries)

	getTestnet := false
	validAtTimestamp := entries[0].Expiry
	dapps, err := GetActiveDapps(db, validAtTimestamp, getTestnet)
	require.NoError(t, err)
	require.Equal(t, 3, len(dapps))

	require.Equal(t, 3, len(dapps))
	require.Equal(t, entries[5].Name, dapps[0].Name)
	require.Equal(t, entries[4].Name, dapps[1].Name)
	require.Equal(t, entries[1].Name, dapps[2].Name)
}

// TestGetActiveDapps_ActiveWorksAsExpected tests the combination of disconnected and expired sessions
func TestGetActiveDapps_ActiveWorksAsExpected(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	not := common.NewAndSet(false)
	yes := common.NewAndSet(true)
	timeNow := 4
	entries := generateTestData([]testSession{
		{url: urlFor(1), expiry: at(timeNow - 3), disconnected: not},
		{url: urlFor(1), expiry: at(timeNow - 2), disconnected: yes},
		{url: urlFor(2), expiry: at(timeNow - 2), disconnected: not},
		{url: urlFor(3), expiry: at(timeNow - 1), disconnected: yes},
		// ----- timeNow
		{url: urlFor(3), expiry: at(timeNow + 1), disconnected: not},
		{url: urlFor(4), expiry: at(timeNow + 1), disconnected: yes},
		{url: urlFor(4), expiry: at(timeNow + 2), disconnected: not},
		{url: urlFor(5), expiry: at(timeNow + 2), disconnected: yes},
		{url: urlFor(6), expiry: at(timeNow + 3), disconnected: not},
	})
	insertTestData(t, db, entries)

	getTestnet := false
	dapps, err := GetActiveDapps(db, int64(timeNow), getTestnet)
	require.NoError(t, err)
	require.Equal(t, 3, len(dapps))
}

// TestGetActiveDapps_TestChainsWorksAsExpected tests the combination of disconnected and expired sessions
func TestGetActiveDapps_TestChainsWorksAsExpected(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	not := common.NewAndSet(false)
	yes := common.NewAndSet(true)
	timeNow := 4
	entries := generateTestData([]testSession{
		{url: urlFor(1), testChains: not, expiry: at(timeNow - 3), disconnected: not},
		{url: urlFor(2), testChains: yes, expiry: at(timeNow - 2), disconnected: not},
		{url: urlFor(2), testChains: not, expiry: at(timeNow - 1), disconnected: not},
		// ----- timeNow
		{url: urlFor(3), testChains: not, expiry: at(timeNow + 1), disconnected: not},
		{url: urlFor(4), testChains: not, expiry: at(timeNow + 2), disconnected: not},
		{url: urlFor(4), testChains: yes, expiry: at(timeNow + 3), disconnected: not},
		{url: urlFor(5), testChains: yes, expiry: at(timeNow + 4), disconnected: not},
	})
	insertTestData(t, db, entries)

	getTestnet := true
	dapps, err := GetActiveDapps(db, int64(timeNow), getTestnet)
	require.NoError(t, err)
	require.Equal(t, 2, len(dapps))
}

// TestGetDapps_EmptyDB tests that an empty database will return an empty list
func TestGetDapps_EmptyDB(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	entries := generateTestData([]testSession{})
	insertTestData(t, db, entries)

	getTestnet := false
	validAtTimestamp := int64(0)
	dapps, err := GetActiveDapps(db, validAtTimestamp, getTestnet)
	require.NoError(t, err)
	require.Equal(t, 0, len(dapps))
}

// TestGetDapps_OrphanDapps tests that missing session will place the dapp at the end
func TestGetDapps_OrphanDapps(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	not := common.NewAndSet(false)
	entries := generateTestData([]testSession{
		{url: urlFor(1), disconnected: not},
		{url: urlFor(2), disconnected: not},
		{url: urlFor(2), disconnected: not},
	})
	insertTestData(t, db, entries)

	err := DeleteSession(db, entries[1].Topic)
	require.NoError(t, err)
	err = DeleteSession(db, entries[2].Topic)
	require.NoError(t, err)

	getTestnet := false
	validAtTimestamp := entries[0].Expiry
	dapps, err := GetActiveDapps(db, validAtTimestamp, getTestnet)
	require.NoError(t, err)
	// The orphan dapp is not considered active
	require.Equal(t, 1, len(dapps))
	require.Equal(t, entries[0].Name, dapps[0].Name)
}

func TestDisconnectSession(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	not := common.NewAndSet(false)
	entries := generateTestData([]testSession{
		{url: urlFor(1), disconnected: not},
		{url: urlFor(2), disconnected: not},
		{url: urlFor(2), disconnected: not},
	})
	insertTestData(t, db, entries)

	activeSessions, err := GetActiveSessions(db, 0)
	require.NoError(t, err)
	require.Equal(t, 3, len(activeSessions))

	getTestnet := false
	validAtTimestamp := entries[0].Expiry
	dapps, err := GetActiveDapps(db, validAtTimestamp, getTestnet)
	require.NoError(t, err)
	require.Equal(t, 2, len(dapps))

	err = DisconnectSession(db, entries[1].Topic)
	require.NoError(t, err)

	activeSessions, err = GetActiveSessions(db, 0)
	require.NoError(t, err)
	require.Equal(t, 2, len(activeSessions))

	err = DisconnectSession(db, entries[2].Topic)
	require.NoError(t, err)

	activeSessions, err = GetActiveSessions(db, 0)
	require.NoError(t, err)
	require.Equal(t, 1, len(activeSessions))

	dapps, err = GetActiveDapps(db, validAtTimestamp, getTestnet)
	require.NoError(t, err)
	require.Equal(t, 1, len(dapps))
	require.Equal(t, entries[0].Name, dapps[0].Name)
}
