package sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/t/helpers"
)

// TestCommunitiesMigrationDirty tests the communities migration when
// dirty flag has been set to true.
// We first make it fail, then clean up so that it can be replayed, and
// then execute again, and we should be all migrated.
func TestCommunitiesMigrationDirty(t *testing.T) {
	// Open the db for the first time.
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	// Create a communities table, so that migration will fail
	_, err = db.Exec(`CREATE TABLE communities_communities (a varchar);`)
	require.NoError(t, err)

	// Migrate the database, this should fail
	err = Migrate(db)
	require.Error(t, err)

	// Version and dirty should be true and set to communities migration
	var version uint
	var dirty bool

	err = db.QueryRow(`SELECT version, dirty FROM `+migrationsTable).Scan(&version, &dirty)
	require.NoError(t, err)

	require.True(t, dirty)
	require.Equal(t, communitiesMigrationVersion, version)

	// Drop communities table and re-run migrations

	_, err = db.Exec(`DROP TABLE communities_communities`)

	require.NoError(t, err)

	// Migrate the database, this should work
	err = Migrate(db)
	require.NoError(t, err)

	// Make sure communities table is present

	var name string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='communities_communities'`).Scan(&name)

	require.NoError(t, err)
	require.Equal(t, "communities_communities", name)

}

// TestCommunitiesMigrationNotDirty tests the communities migration when
// dirty flag has been set to false, and the communities migration has
// effectively been skipped.
// We first make it fail, then clean up so that it can be replayed, set
// dirty to false and then execute again, and we should be all migrated.
func TestCommunitiesMigrationNotDirty(t *testing.T) {
	// Open the db for the first time.
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	// Create a communities table, so that migration will fail
	_, err = db.Exec(`CREATE TABLE communities_communities (a varchar);`)
	require.NoError(t, err)

	// Migrate the database, this should fail
	err = Migrate(db)
	require.Error(t, err)

	// Set dirty to false
	// Disabling linter as migrationsTable is controlled by us
	_, err = db.Exec(`UPDATE ` + migrationsTable + ` SET dirty = 0`) // nolint: gosec
	require.NoError(t, err)

	// Version and dirty should be true and set to communities migration
	var version uint
	var dirty bool

	err = db.QueryRow(`SELECT version, dirty FROM `+migrationsTable).Scan(&version, &dirty)
	require.NoError(t, err)

	require.False(t, dirty)
	require.Equal(t, communitiesMigrationVersion, version)

	// Drop communities table and re-run migrations
	_, err = db.Exec(`DROP TABLE communities_communities`)

	require.NoError(t, err)

	// Migrate the database, this should work
	err = Migrate(db)
	require.NoError(t, err)

	// Make sure communities table is present

	var name string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='communities_communities'`).Scan(&name)

	require.NoError(t, err)
	require.Equal(t, "communities_communities", name)

}
