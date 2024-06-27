package persistence

import (
	"testing"

	"database/sql"

	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

var testDApp = DApp{
	Name:          "Test DApp",
	URL:           "https://test-dapp-url.com",
	IconURL:       "https://test-dapp-icon-url.com",
	SharedAccount: "0x1234567890",
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

	dAppBack, err := SelectDAppByUrl(db, testDApp.URL)
	require.NoError(t, err)
	require.Equal(t, &testDApp, dAppBack)
}

func TestInsertAndUpdateDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	updatedDApp := DApp{
		Name:    "Updated Test DApp",
		URL:     testDApp.URL,
		IconURL: "https://updated-test-dapp-icon-url.com",
	}

	err = UpsertDApp(db, &updatedDApp)
	require.NoError(t, err)

	dAppBack, err := SelectDAppByUrl(db, testDApp.URL)
	require.NoError(t, err)
	require.Equal(t, &updatedDApp, dAppBack)
	require.NotEqual(t, &testDApp, dAppBack)
}

func TestInsertAndRemoveDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	dAppBack, err := SelectDAppByUrl(db, testDApp.URL)
	require.NoError(t, err)
	require.Equal(t, &testDApp, dAppBack)

	err = DeleteDApp(db, testDApp.URL)
	require.NoError(t, err)

	dAppBack, err = SelectDAppByUrl(db, testDApp.URL)
	require.NoError(t, err)
	require.Empty(t, dAppBack)
}
