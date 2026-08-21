package walletdb

import (
	"database/sql"

	sqlite "github.com/status-im/status-go/internal/db/sqlite"
)

type DbInitializer struct {
}

func (a DbInitializer) Initialize(path, password string, kdfIterationsNumber int) (*sql.DB, error) {
	return InitializeDB(path, password, kdfIterationsNumber)
}

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path, password string, kdfIterationsNumber int) (*sql.DB, error) {
	db, err := sqlite.OpenDB(path, password, kdfIterationsNumber)
	if err != nil {
		return nil, err
	}

	err = doMigration(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func OpenDB(path, password string, kdfIterationsNumber int) (*sql.DB, error) {
	return sqlite.OpenDB(path, password, kdfIterationsNumber)
}
