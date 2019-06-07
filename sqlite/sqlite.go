package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation
)

func openDB(path, key string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}
	keyString := fmt.Sprintf("PRAGMA key = '%s'", key)
	if _, err = db.Exec(keyString); err != nil {
		return nil, errors.New("failed to set key pragma")
	}

	// readers do not block writers and faster i/o operations
	// https://www.sqlite.org/draft/wal.html
	// must be set after db is encrypted
	var mode string
	err = db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode)
	if err != nil {
		return nil, err
	}
	if mode != "wal" {
		return nil, fmt.Errorf("unable to set journal_mode to WAL. actual mode %s", mode)
	}

	return db, nil
}

// OpenDB opens not-encrypted database.
func OpenDB(path, key string) (*sql.DB, error) {
	return openDB(path, key)
}
