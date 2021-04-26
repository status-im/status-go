package postgres

import (
	"database/sql"

	// Import postgres driver
	_ "github.com/lib/pq"
	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/postgres"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
)

type DB struct {
	db   *sql.DB
	name string
	done chan struct{}
}

func NewPostgresDB(uri string, migrationResource *bindata.AssetSource) (*DB, error) {
	db, err := sql.Open("postgres", uri)
	if err != nil {
		return nil, err
	}

	instance := &DB{
		db:   db,
		done: make(chan struct{}),
	}
	if err := instance.setup(migrationResource); err != nil {
		return nil, err
	}

	// name is used for metrics labels
	if name, err := instance.getDBName(); err == nil {
		instance.name = name
	}

	return instance, nil
}

func (d *DB) getDBName() (string, error) {
	query := "SELECT current_database()"
	var dbName string
	return dbName, d.db.QueryRow(query).Scan(&dbName)
}

func (d *DB) setup(migrationResource *bindata.AssetSource) error {
	source, err := bindata.WithInstance(migrationResource)
	if err != nil {
		return err
	}

	driver, err := postgres.WithInstance(d.db, &postgres.Config{})
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

func (d *DB) Close() error {
	select {
	case <-d.done:
	default:
		close(d.done)
	}
	return d.db.Close()
}
