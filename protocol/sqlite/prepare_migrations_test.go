package sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareMigrationsEmpty(t *testing.T) {
	names, _, err := prepareMigrations(nil)
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestPrepareMigrationsWithDefaultMigrations(t *testing.T) {
	names, _, err := prepareMigrations(defaultMigrations)
	require.NoError(t, err)
	require.NotEmpty(t, names)
}

func TestPrepareMigrationsErrors(t *testing.T) {
	// Two migrations with the same name in the same set.
	names, _, err := prepareMigrations([]migrationsWithGetter{
		{
			Names:  []string{"name1.sql", "name1.sql"},
			Getter: nil,
		},
	})
	require.EqualError(t, err, "migration with name name1.sql already exists")
	require.Empty(t, names)

	// Two migrations with the same name in different sets.
	names, _, err = prepareMigrations([]migrationsWithGetter{
		{
			Names:  []string{"name2.sql"},
			Getter: nil,
		},
		{
			Names:  []string{"name2.sql"},
			Getter: nil,
		},
	})
	require.EqualError(t, err, "migration with name name2.sql already exists")
	require.Empty(t, names)

	// No getter for a migration.
	_, getter, err := prepareMigrations([]migrationsWithGetter{
		{
			Names:  []string{"name3.sql"},
			Getter: nil,
		},
	})
	require.NoError(t, err)
	_, err = getter("non-existing-migration.sql")
	require.EqualError(t, err, "no migration for name non-existing-migration.sql")
}
