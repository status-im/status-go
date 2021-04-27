package postgres

import (
	"database/sql"

	// Import postgres driver
	_ "github.com/lib/pq"
	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/postgres"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
)

func NewMigratedDB(uri string, migrationResource *bindata.AssetSource) (*sql.DB, error) {
	db, err := sql.Open("postgres", uri)
	if err != nil {
		return nil, err
	}

	if err := setup(db, migrationResource); err != nil {
		return nil, err
	}

	return db, nil
}

func setup(d *sql.DB, migrationResource *bindata.AssetSource) error {
	source, err := bindata.WithInstance(migrationResource)
	if err != nil {
		return err
	}

	driver, err := postgres.WithInstance(d, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"postgres",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}

	return nil
}
