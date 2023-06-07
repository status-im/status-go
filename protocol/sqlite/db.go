package sqlite

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pkg/errors"

	_ "github.com/mutecomm/go-sqlcipher/v4" // We require go sqlcipher that overrides default implementation
	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/sqlcipher"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	mvdsmigrations "github.com/vacp2p/mvds/persistenceutil"
)

// The reduced number of kdf iterations (for performance reasons) which is
// currently used for derivation of the database key
// https://github.com/status-im/status-go/pull/1343
// https://notes.status.im/i8Y_l7ccTiOYq09HVgoFwA
const ReducedKDFIterationsNumber = 3200

const InMemoryPath = ":memory:"

var migrationsTable = "status_protocol_go_" + sqlcipher.DefaultMigrationsTable

// MigrationConfig is a struct that allows to define bindata migrations.
type MigrationConfig struct {
	AssetNames  []string
	AssetGetter func(name string) ([]byte, error)
}

// Open opens or initializes a new database for a given file path.
// MigrationConfig is optional but if provided migrations are applied automatically.
func Open(path, key string, kdfIterationNumber int) (*sql.DB, error) {
	return openAndMigrate(path, key, kdfIterationNumber)
}

// OpenInMemory opens an in memory SQLite database.
// Number of KDF iterations is reduced to 0.
func OpenInMemory() (*sql.DB, error) {
	return openAndMigrate(InMemoryPath, "", 0)
}

// OpenWithIter allows to open a new database with a custom number of kdf iterations.
// Higher kdf iterations number makes it slower to open the database.
func OpenWithIter(path, key string, kdfIter int) (*sql.DB, error) {
	return openAndMigrate(path, key, kdfIter)
}

func open(path string, key string, kdfIter int) (*sql.DB, error) {
	if path != InMemoryPath {
		_, err := os.OpenFile(path, os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	keyString := fmt.Sprintf("PRAGMA key = '%s'", key)

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}

	if _, err = db.Exec(keyString); err != nil {
		return nil, err
	}

	kdfString := fmt.Sprintf("PRAGMA kdf_iter = '%d'", kdfIter)

	if _, err = db.Exec(kdfString); err != nil {
		return nil, err
	}

	return db, nil
}

func openAndMigrate(path string, key string, kdfIter int) (*sql.DB, error) {
	db, err := open(path, key, kdfIter)
	if err != nil {
		return nil, err
	}

	if err := Migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

// applyMigrations allows to apply bindata migrations on the current *sql.DB.
// `assetNames` is a list of assets with migrations and `assetGetter` is responsible
// for returning the content of the asset with a given name.
func applyMigrations(db *sql.DB, assetNames []string, assetGetter func(name string) ([]byte, error)) error {
	resources := bindata.Resource(
		assetNames,
		assetGetter,
	)

	source, err := bindata.WithInstance(resources)
	if err != nil {
		return errors.Wrap(err, "failed to create migration source")
	}

	driver, err := sqlcipher.WithInstance(db, &sqlcipher.Config{
		MigrationsTable: migrationsTable,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create driver")
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"sqlcipher",
		driver,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create migration instance")
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return errors.Wrap(err, "could not get version")
	}

	err = ApplyAdHocMigrations(version, dirty, m, db)
	if err != nil {
		return errors.Wrap(err, "failed to apply ad-hoc migrations")
	}

	if dirty {
		err = ReplayLastMigration(version, m)
		if err != nil {
			return errors.Wrap(err, "failed to replay last migration")
		}
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return errors.Wrap(err, "failed to migrate")
	}

	return nil
}

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
	err = applyMigrations(database, migrationNames, migrationGetter)
	if err != nil {
		return errors.Wrap(err, "failed to apply status-go/protocol migrations")
	}
	return nil
}
