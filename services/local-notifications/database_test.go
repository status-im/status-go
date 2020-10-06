package localnotifications

import (
	"database/sql"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
)

func setupAppTestDb(t *testing.T) (*sql.DB, func()) {
	tmpfile, err := ioutil.TempFile("", "local-notifications-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "local-notifications-tests")
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func setupTestDB(t *testing.T) (*Database, func()) {
	db, stop := setupAppTestDb(t)

	return NewDB(db, 1777), func() {
		require.NoError(t, db.Close())
		stop()
	}
}

func TestWalletPreferences(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	enabled := true
	require.NoError(t, db.ChangeWalletPreference(enabled))
	rst, err := db.GetWalletPreference()
	require.Equal(t, enabled, rst.Enabled)

	enabled = false
	require.NoError(t, db.ChangeWalletPreference(enabled))
	rst, err = db.GetWalletPreference()
	require.Equal(t, enabled, rst.Enabled)

	require.NoError(t, err)
}

func TestPreferences(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	enabled := true

	require.NoError(t, db.ChangeWalletPreference(enabled))
	rst, err := db.GetPreferences()

	require.Equal(t, 1, len(rst))
	require.Equal(t, enabled, rst[0].Enabled)

	require.NoError(t, err)
}
