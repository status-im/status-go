package sqlite

import (
	"database/sql"

	"github.com/pkg/errors"

	_ "github.com/mutecomm/go-sqlcipher/v4" // We require go sqlcipher that overrides default implementation
	"github.com/status-im/migrate/v4/database/sqlcipher"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"

	"github.com/status-im/status-go/sqlite"
)

var migrationsTable = "status_protocol_go_" + sqlcipher.DefaultMigrationsTable

func Migrate(database *sql.DB) error {
	// Apply migrations for all components.
	err := mvdsmigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply mvds migrations")
	}

	migrationNames, migrationGetter, err := prepareMigrations(defaultMigrations)
	if err != nil {
		return errors.Wrap(err, "failed to prepare status-go/protocol migrations")
	}

	options := sqlite.MigrateOptions{
		MigrationTableName: migrationsTable,
	}
	err = sqlite.Migrate(database, bindata.Resource(migrationNames, bindata.AssetFunc(migrationGetter)), options)
	if err != nil {
		return errors.Wrap(err, "failed to apply status-go/protocol migrations")
	}

	return nil
}
