package walletconnect

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/testutils"
)

func SetupTestDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}
