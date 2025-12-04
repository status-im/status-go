package sqlite

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Scenario:
// - The parent module currently reports migrations up to version 4.
// - A subset of those migrations (e.g., 1, 2, and 5) is moved into a new submodule.
// - The submodule applies the remaining migrations that weren't ran in the parent.
// - Additional migrations are then added and executed within the submodule.
func TestUpdateMigrationTableVersion(t *testing.T) {
	db, err := OpenDB(":memory:", "password", 3200)
	require.NoError(t, err)

	migrationTableName := "schema_migrations_submodule"

	checkVersion := func(expectedVersion uint) {
		row := db.QueryRow(fmt.Sprintf(`SELECT version, dirty FROM %s`, migrationTableName))
		var version uint64
		var dirty bool
		require.NoError(t, row.Scan(&version, &dirty))
		require.EqualValues(t, expectedVersion, version)
		require.False(t, dirty)
	}

	simulateMigration := func(newVersion uint) {
		r, err := db.Exec(fmt.Sprintf(`UPDATE %s SET version = ?`, migrationTableName), newVersion)
		require.NoError(t, err)
		affected, err := r.RowsAffected()
		require.NoError(t, err)
		require.EqualValues(t, 1, affected)
	}

	assetNames := []string{
		"1_A.up.sql",
		"2_B.up.sql",
		"5_E.up.sql",
	}

	currentParentVersion := uint(4)
	currentVersion := uint(2)

	// Initial migration table setup
	err = UpdateMigrationTableVersion(db, migrationTableName, assetNames, currentParentVersion)
	require.NoError(t, err)
	checkVersion(currentVersion)

	// Simulate the migration to version 5
	currentVersion = 5
	simulateMigration(currentVersion)
	checkVersion(currentVersion)

	// Invoke `UpdateMigrationTableVersion` again
	err = UpdateMigrationTableVersion(db, migrationTableName, assetNames, currentParentVersion)
	require.NoError(t, err)
	// Verify version remains at 5
	checkVersion(currentVersion)

	// Add new migration and update to it
	assetNames = append(assetNames, "6_F.up.sql")
	currentVersion = 6
	simulateMigration(currentVersion)
	checkVersion(currentVersion)

	// Invoke `UpdateMigrationTableVersion` once again
	err = UpdateMigrationTableVersion(db, migrationTableName, assetNames, currentParentVersion)
	require.NoError(t, err)
	// Verify version remains at 6
	checkVersion(currentVersion)

	// Simulate parent module updating to version 10
	currentParentVersion = 10
	err = UpdateMigrationTableVersion(db, migrationTableName, assetNames, currentParentVersion)
	require.NoError(t, err)
	// Verify version remains at 6
	checkVersion(currentVersion)
}
