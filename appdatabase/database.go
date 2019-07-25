package appdatabase

import (
	"database/sql"

	"github.com/status-im/status-go/appdatabase/migrations"
	"github.com/status-im/status-go/sqlite"
)

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path, password string) (*sql.DB, error) {
	db, err := sqlite.OpenDB(path, password)
	if err != nil {
		return nil, err
	}
	err = migrations.Migrate(db)
	if err != nil {
		return nil, err
	}
	return db, nil
}
