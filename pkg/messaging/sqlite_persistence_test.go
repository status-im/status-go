package messaging

import (
	"testing"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/testutils"
	wakumigrations "github.com/status-im/status-go/pkg/messaging/waku/migrations"
)

func TestResetHistoryCursorsMigration(t *testing.T) {
	db, cleanup, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "reset-history-cursors-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cleanup()) })

	source := &bindata.AssetSource{Names: wakumigrations.AssetNames(), AssetFunc: wakumigrations.Asset}
	options := func(until *uint) sqlite.MigrateOptions {
		return sqlite.MigrateOptions{MigrationTableName: "status_schema_migrations_waku", UntilVersion: until}
	}

	beforeReset := uint(1763548446)
	require.NoError(t, sqlite.Migrate(db, source, options(&beforeReset)))

	_, err = db.Exec(`
		INSERT INTO mailserver_topics (topic, pubsub_topic, last_request) VALUES
			('0x00000001', '/waku/2/rs/16/32', 1791479056),
			('0x00000002', '/waku/2/rs/16/64', 0)`)
	require.NoError(t, err)

	require.NoError(t, sqlite.Migrate(db, source, options(nil)))

	var nonZero, total int
	require.NoError(t, db.QueryRow(
		`SELECT SUM(last_request != 0), COUNT(*) FROM mailserver_topics`,
	).Scan(&nonZero, &total))
	require.Equal(t, 2, total, "topics must be kept")
	require.Zero(t, nonZero, "every history cursor must be reset")
}
