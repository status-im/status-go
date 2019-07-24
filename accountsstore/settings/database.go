package settings

import (
	"database/sql"

	"github.com/status-im/status-go/accountsstore/settings/migrations"
	"github.com/status-im/status-go/sqlite"
)

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path, password string) (*Database, error) {
	db, err := sqlite.OpenDB(path, password)
	if err != nil {
		return nil, err
	}
	err = migrations.Migrate(db)
	if err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
}

func (db *Database) SaveConfig(typ string, value interface{}) error {
	_, err := db.db.Exec("INSERT OR REPLACE INTO settings (type, value) VALUES (?, ?)", typ, &sqlite.JSONBlob{value})
	return err
}

func (db *Database) GetConfig(typ string, value interface{}) error {
	return db.db.QueryRow("SELECT value FROM settings WHERE type = ?", typ).Scan(&sqlite.JSONBlob{value})
}

func (db *Database) GetBlob(typ string) (rst []byte, err error) {
	return rst, db.db.QueryRow("SELECT value FROM settings WHERE type = ?", typ).Scan(&rst)
}
