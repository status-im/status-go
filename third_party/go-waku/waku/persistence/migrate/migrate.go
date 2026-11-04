package migrate

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
)

// Migrate applies migrations.
func Migrate(db *sql.DB, driver database.Driver, assetNames []string, assetFunc bindata.AssetFunc) error {
	return migrateDB(db, bindata.Resource(
		assetNames,
		assetFunc,
	), driver, true)
}

// Migrate applies migrations.
func MigrateDown(db *sql.DB, driver database.Driver, assetNames []string, assetFunc bindata.AssetFunc) error {
	return migrateDB(db, bindata.Resource(
		assetNames,
		assetFunc,
	), driver, false)
}

// Migrate database using provided resources.
func migrateDB(db *sql.DB, resources *bindata.AssetSource, driver database.Driver, up bool) error {
	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"gowakudb",
		driver)
	if err != nil {
		return err
	}

	fn := m.Up
	if !up {
		fn = m.Down
	}

	if err = fn(); err != migrate.ErrNoChange {
		return err
	}
	return nil
}
