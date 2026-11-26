package sqlite

import (
	"testing"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	dbsqlite "github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging"
)

func TestMigrateFromOriginalSDSMarkerVersion(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, messaging.SQLiteMigrate(db, 0))

	names, getter, err := prepareMigrations(defaultMigrations)
	require.NoError(t, err)
	previousVersion := uint(1794253360)
	require.NoError(t, dbsqlite.Migrate(db, bindata.Resource(names, bindata.AssetFunc(getter)), dbsqlite.MigrateOptions{
		MigrationTableName: migrationsTable,
		UntilVersion:       &previousVersion,
	}))

	// Simulate a database that stopped after the original SDS marker migration.
	_, err = db.Exec(`CREATE TABLE processed_sds_messages (message_id TEXT PRIMARY KEY NOT NULL);
		INSERT INTO processed_sds_messages (message_id) VALUES ('before-upgrade')`)
	require.NoError(t, err)
	_, err = db.Exec(dbsqlite.TableQuery("UPDATE %s SET version = ?", migrationsTable), 1794937279)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	var processedAt int64
	require.NoError(t, db.QueryRow(`SELECT processed_at FROM processed_sds_messages WHERE message_id = 'before-upgrade'`).Scan(&processedAt))
	require.Positive(t, processedAt)
	var clockCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM community_member_delete_all_clocks`).Scan(&clockCount))
}
