package persistence

import (
	"testing"

	"database/sql"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

var testDApp = DApp{
	Name:          "Test DApp",
	URL:           "https://test-dapp-url.com",
	IconURL:       "https://test-dapp-icon-url.com",
	ClientID:      "",
	SharedAccount: types.HexToAddress("0x1234567890"),
	ChainID:       0x1,
}

func setupTestDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

func TestInsertAndSelectDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	dAppBack, err := SelectDAppByUrlAndClientID(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &testDApp, dAppBack)
}

func TestInsertAndUpdateDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	updatedDApp := DApp{
		Name:     "Updated Test DApp",
		URL:      testDApp.URL,
		IconURL:  "https://updated-test-dapp-icon-url.com",
		ClientID: testDApp.ClientID,
	}

	err = UpsertDApp(db, &updatedDApp)
	require.NoError(t, err)

	dAppBack, err := SelectDAppByUrlAndClientID(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &updatedDApp, dAppBack)
	require.NotEqual(t, &testDApp, dAppBack)
}

func TestInsertAndRemoveDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	dAppBack, err := SelectDAppByUrlAndClientID(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &testDApp, dAppBack)

	err = DeleteDApp(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)

	dAppBack, err = SelectDAppByUrlAndClientID(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Empty(t, dAppBack)
}

func TestSelectAllDApps(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	dApps, err := SelectAllDApps(db)
	require.NoError(t, err)
	require.Len(t, dApps, 1)
	require.Equal(t, testDApp, dApps[0])
}

func TestMultipleClientsWithSameURL(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// Create two dApps with same URL but different clientIds
	dApp1 := DApp{
		Name:          "DApp Client 1",
		URL:           "https://same-url.com",
		IconURL:       "https://same-icon.com",
		ClientID:      "client1",
		SharedAccount: types.HexToAddress("0x1111111111"),
		ChainID:       0x1,
	}

	dApp2 := DApp{
		Name:          "DApp Client 2",
		URL:           "https://same-url.com", // Same URL
		IconURL:       "https://same-icon.com",
		ClientID:      "client2",
		SharedAccount: types.HexToAddress("0x2222222222"),
		ChainID:       0x89,
	}

	err := UpsertDApp(db, &dApp1)
	require.NoError(t, err)
	err = UpsertDApp(db, &dApp2)
	require.NoError(t, err)

	retrievedDApp1, err := SelectDAppByUrlAndClientID(db, dApp1.URL, dApp1.ClientID)
	require.NoError(t, err)
	require.Equal(t, &dApp1, retrievedDApp1)

	retrievedDApp2, err := SelectDAppByUrlAndClientID(db, dApp2.URL, dApp2.ClientID)
	require.NoError(t, err)
	require.Equal(t, &dApp2, retrievedDApp2)

	allDApps, err := SelectAllDApps(db)
	require.NoError(t, err)
	require.Len(t, allDApps, 2)

	foundDApp1 := false
	foundDApp2 := false
	for _, dApp := range allDApps {
		if dApp.ClientID == "client1" {
			require.Equal(t, dApp1, dApp)
			foundDApp1 = true
		}
		if dApp.ClientID == "client2" {
			require.Equal(t, dApp2, dApp)
			foundDApp2 = true
		}
	}
	require.True(t, foundDApp1, "client1 dApp not found")
	require.True(t, foundDApp2, "client2 dApp not found")
}

func TestDeleteSpecificClient(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// Create two dApps with same URL but different clientIds
	dApp1 := DApp{
		Name:          "DApp Client 1",
		URL:           "https://test-delete.com",
		IconURL:       "https://test-icon.com",
		ClientID:      "client1",
		SharedAccount: types.HexToAddress("0x1111111111"),
		ChainID:       0x1,
	}

	dApp2 := DApp{
		Name:          "DApp Client 2",
		URL:           "https://test-delete.com",
		IconURL:       "https://test-icon.com",
		ClientID:      "client2",
		SharedAccount: types.HexToAddress("0x2222222222"),
		ChainID:       0x89,
	}

	err := UpsertDApp(db, &dApp1)
	require.NoError(t, err)
	err = UpsertDApp(db, &dApp2)
	require.NoError(t, err)

	// Delete only client1
	err = DeleteDApp(db, dApp1.URL, dApp1.ClientID)
	require.NoError(t, err)

	deletedDApp1, err := SelectDAppByUrlAndClientID(db, dApp1.URL, dApp1.ClientID)
	require.NoError(t, err)
	require.Nil(t, deletedDApp1)

	// Verify client2 still exists
	stillExistsDApp2, err := SelectDAppByUrlAndClientID(db, dApp2.URL, dApp2.ClientID)
	require.NoError(t, err)
	require.Equal(t, &dApp2, stillExistsDApp2)
}

func TestBackwardCompatibilityEmptyClientID(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	oldDApp := DApp{
		Name:          "Old Client DApp",
		URL:           "https://old-client.com",
		IconURL:       "https://old-icon.com",
		ClientID:      "",
		SharedAccount: types.HexToAddress("0x0000000000"),
		ChainID:       0x1,
	}

	newDApp := DApp{
		Name:          "New Client DApp",
		URL:           "https://old-client.com", // Same URL
		IconURL:       "https://old-icon.com",
		ClientID:      "newclient123",
		SharedAccount: types.HexToAddress("0x3333333333"),
		ChainID:       0x89,
	}

	err := UpsertDApp(db, &oldDApp)
	require.NoError(t, err)
	err = UpsertDApp(db, &newDApp)
	require.NoError(t, err)

	retrievedOldDApp, err := SelectDAppByUrlAndClientID(db, oldDApp.URL, "")
	require.NoError(t, err)
	require.Equal(t, &oldDApp, retrievedOldDApp)

	retrievedNewDApp, err := SelectDAppByUrlAndClientID(db, newDApp.URL, newDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &newDApp, retrievedNewDApp)

	// Delete old client (empty clientId)
	err = DeleteDApp(db, oldDApp.URL, "")
	require.NoError(t, err)

	deletedOldDApp, err := SelectDAppByUrlAndClientID(db, oldDApp.URL, "")
	require.NoError(t, err)
	require.Nil(t, deletedOldDApp)

	// Verify new client still exists
	stillExistsNewDApp, err := SelectDAppByUrlAndClientID(db, newDApp.URL, newDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &newDApp, stillExistsNewDApp)
}
