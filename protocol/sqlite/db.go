package sqlite

import (
	"database/sql"

	"github.com/pkg/errors"

	_ "github.com/mutecomm/go-sqlcipher/v4" // We require go sqlcipher that overrides default implementation
	"github.com/status-im/migrate/v4/database/sqlcipher"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/messaging"
	"github.com/status-im/status-go/sqlite"
)

var migrationsTable = "status_protocol_go_" + sqlcipher.DefaultMigrationsTable

func Migrate(database *sql.DB) error {
	version, _, err := sqlite.GetLastMigrationVersion(database, migrationsTable)
	if err != nil {
		return errors.Wrap(err, "failed to get current migration version")
	}

	// Apply migrations for sub-components.
	err = messaging.SQLiteMigrate(database, version)
	if err != nil {
		return errors.Wrap(err, "failed to apply messaging migrations")
	}

	// Apply migrations for Status protocol.
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
