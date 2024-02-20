package storenodes

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
)

func setupTestDB(t *testing.T, communityIDs ...types.HexBytes) (*Database, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "storenodes-tests-")
	require.NoError(t, err)

	err = sqlite.Migrate(db)
	require.NoError(t, err)

	for _, communityID := range communityIDs {
		err = saveTestCommunity(db, communityID)
		require.NoError(t, err)
	}

	return NewDB(db), func() { require.NoError(t, cleanup()) }
}

func saveTestCommunity(db *sql.DB, communityID types.HexBytes) error {
	_, err := db.Exec(
		`INSERT INTO communities_communities ("id", "private_key", "description", "joined", "verified", "synced_at", "muted") VALUES (?, ?, ?, ?, ?, ?, ?)`,
		communityID,
		[]byte("private_key"),
		[]byte("description"),
		true,
		true,
		0,
		false,
	)
	return err
}
