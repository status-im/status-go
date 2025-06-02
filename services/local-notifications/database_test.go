package localnotifications

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/t/helpers"
)

func setupAppTestDb(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "local-notifications-tests-")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func setupTestDB(t *testing.T, db *sql.DB) (*Database, func()) {
	return NewDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func TestPreferences(t *testing.T) {
	appDB, appStop := setupAppTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	enabled := true

	require.NoError(t, db.ChangePreference(
		NotificationPreference{
			Enabled:    enabled,
			Service:    "service",
			Event:      "event",
			Identifier: "identifier",
		},
	))
	rst, err := db.GetPreferences()

	require.Equal(t, 1, len(rst))
	require.Equal(t, enabled, rst[0].Enabled)

	require.NoError(t, err)
}
