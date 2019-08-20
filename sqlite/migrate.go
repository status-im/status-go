package sqlite

import (
	"database/sql"

	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/sqlcipher"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
)

// Migrate database using provided resources.
func Migrate(db *sql.DB, resources *bindata.AssetSource) error {
	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	driver, err := sqlcipher.WithInstance(db, &sqlcipher.Config{
		MigrationsTable: "status_go_" + sqlcipher.DefaultMigrationsTable,
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"sqlcipher",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}
	return nil
}
