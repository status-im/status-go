package messaging

import (
	"database/sql"

	"github.com/pkg/errors"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"

	"github.com/status-im/status-go/messaging/internal"
	"github.com/status-im/status-go/messaging/layers/encryption"
	encryptionmigrations "github.com/status-im/status-go/messaging/layers/encryption/migrations"
	"github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/sqlite"
)

/*
 Reference implementation of the Persistence interface using SQLite.

 Currently, only EncryptionPersistence is supported.
 Other components remain in the Status client and may be migrated here as needed.
*/

// SQLiteMigrate applies necessary migrations to the SQLite database schema.
func SQLiteMigrate(database *sql.DB, version uint) error {
	if version > 0 {
		err := createMigrationTables(database, version)
		if err != nil {
			return errors.Wrap(err, "failed to create migration tables")
		}
	}

	err := mvdsmigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply mvds migrations")
	}

	err = encryptionmigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply encryption migrations")
	}

	return nil
}

// Migration tables were transitioned from a single shared table in the client (status_protocol_go)
// to dedicated pre-component tables. To maintain migration consistency and prevent reapplication
// of migrations already executed in the client, it is essential to initialize any newly created
// migration tables with the latest version. This ensures that components introduced into the client
// do not re-run migrations that have previously been applied.
func createMigrationTables(database *sql.DB, version uint) error {
	latestApplied := sqlite.LatestMigrationUpToVersion(encryptionmigrations.AssetNames(), version)
	if latestApplied == 0 {
		return nil
	}
	return sqlite.CreateMigrationTable(database, encryptionmigrations.TableName, latestApplied)
}

// Returns internal implementation of EncryptionPersistence.
// Clients should not use this interface directly.
func NewEncryptionSQLitePersistence(db *sql.DB) types.EncryptionPersistence {
	return &internal.SQLiteEncryptionPersistence{
		SQLitePersistence: encryption.NewSQLitePersistence(db),
	}
}
