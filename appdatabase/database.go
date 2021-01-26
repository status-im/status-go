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

// DecryptDatabase creates an unencrypted copy of the database and copies it
// over to the given directory
func DecryptDatabase(oldPath, newPath, password string) error {
	return sqlite.DecryptDB(oldPath, newPath, password)
}

// EncryptDatabase creates an encrypted copy of the database and copies it to the
// user path
func EncryptDatabase(oldPath, newPath, password string) error {
	return sqlite.EncryptDB(oldPath, newPath, password)
}
