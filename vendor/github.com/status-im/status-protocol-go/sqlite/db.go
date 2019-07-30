package sqlite

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pkg/errors"

	_ "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation
	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/sqlcipher"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
)

// The default number of kdf iterations in sqlcipher (from version 3.0.0)
// https://github.com/sqlcipher/sqlcipher/blob/fda4c68bb474da7e955be07a2b807bda1bb19bd2/CHANGELOG.md#300---2013-11-05
// https://www.zetetic.net/sqlcipher/sqlcipher-api/#kdf_iter
const defaultKdfIterationsNumber = 64000 // nolint: deadcode,varcheck,unused

// The reduced number of kdf iterations (for performance reasons) which is
// currently used for derivation of the database key
// https://github.com/status-im/status-go/pull/1343
// https://notes.status.im/i8Y_l7ccTiOYq09HVgoFwA
const reducedKdfIterationsNumber = 3200

// MigrationConfig is a struct that allows to define bindata migrations.
type MigrationConfig struct {
	AssetNames  []string
	AssetGetter func(name string) ([]byte, error)
}

// Open opens or initializes a new database for a given file path.
// MigrationConfig is optional but if provided migrations are applied automatically.
func Open(path, key string, mc ...MigrationConfig) (*sql.DB, error) {
	return open(path, key, reducedKdfIterationsNumber, mc)
}

// OpenWithIter allows to open a new database with a custom number of kdf iterations.
// Higher kdf iterations number makes it slower to open the database.
func OpenWithIter(path, key string, kdfIter int, mc ...MigrationConfig) (*sql.DB, error) {
	return open(path, key, kdfIter, mc)
}

func open(path string, key string, kdfIter int, configs []MigrationConfig) (*sql.DB, error) {
	_, err := os.OpenFile(path, os.O_CREATE, 0644)
	if err != nil {
		return nil, err
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

	// Apply all provided migrations.
	for _, mc := range configs {
		if err := ApplyMigrations(db, mc.AssetNames, mc.AssetGetter); err != nil {
			return nil, err
		}
	}

	return db, nil
}

// ApplyMigrations allows to apply bindata migrations on the current *sql.DB.
// `assetNames` is a list of assets with migrations and `assetGetter` is responsible
// for returning the content of the asset with a given name.
func ApplyMigrations(db *sql.DB, assetNames []string, assetGetter func(name string) ([]byte, error)) error {
	resources := bindata.Resource(
		assetNames,
		assetGetter,
	)

	source, err := bindata.WithInstance(resources)
	if err != nil {
		return errors.Wrap(err, "failed to create migration source")
	}

	driver, err := sqlcipher.WithInstance(db, &sqlcipher.Config{})
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

	if err = m.Up(); err != migrate.ErrNoChange {
		return errors.Wrap(err, "failed to migrate")
	}

	return nil
}
