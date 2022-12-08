//go:build !gowaku_skip_migrations
// +build !gowaku_skip_migrations

package migrations

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
)

// Migrate applies migrations.
func Migrate(db *sql.DB) error {
	return migrateDB(db, bindata.Resource(
		AssetNames(),
		func(name string) ([]byte, error) {
			return Asset(name)
		},
	))
}

// Migrate database using provided resources.
func migrateDB(db *sql.DB, resources *bindata.AssetSource) error {
	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: "gowaku_" + sqlite3.DefaultMigrationsTable,
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"sqlite",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}
	return nil
}
