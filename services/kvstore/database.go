package kvstore

import (
	"database/sql"
)

const (
	ConfigRlnRateLimitEnabled = "config/rln-rate-limit-enabled"
)

var DeprecatedKeys = []string{}

// Database sql wrapper for db operations.
type Database struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

// Set stores a key-value pair in kv_store
func (db *Database) Set(key string, value []byte) error {
	query := `INSERT INTO kv_store (key, value) VALUES (?, ?)
	          ON CONFLICT(key) DO UPDATE SET value = excluded.value;`
	_, err := db.db.Exec(query, key, value)
	return err
}

// Get retrieves a value by key
func (db *Database) Get(key string) ([]byte, error) {
	var value []byte
	err := db.db.QueryRow(`SELECT value FROM kv_store WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return value, err
}

// SetBool stores a boolean value in kv_store
func (db *Database) SetBool(key string, value bool) error {
	var boolByte byte
	if value {
		boolByte = 1
	} else {
		boolByte = 0
	}
	return db.Set(key, []byte{boolByte})
}

// GetBool retrieves a boolean value by key
func (db *Database) GetBool(key string) (bool, error) {
	value, err := db.Get(key)
	if err != nil {
		return false, err
	}
	if value == nil {
		return false, nil // Default to false if key is missing
	}
	return value[0] == 1, nil
}

// Delete removes a key from kv_store
func (db *Database) Delete(key string) error {
	_, err := db.db.Exec(`DELETE FROM kv_store WHERE key = ?`, key)
	return err
}

// DropKeys removes unused keys from kv_store
func (db *Database) DropDeprecatedKeys(keys []string) error {
	for _, key := range keys {
		err := db.Delete(key)
		if err != nil {
			return err
		}
	}

	return nil
}
