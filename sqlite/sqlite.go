package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/sha3"

	_ "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation
)

// The reduced number of kdf iterations (for performance reasons) which is
// currently used for derivation of the database key
// https://github.com/status-im/status-go/pull/1343
// https://notes.status.im/i8Y_l7ccTiOYq09HVgoFwA
const kdfIterationsNumber = 3200

// OpenDBWithKey opens encrypted database passing key as PRAGMA key.
func OpenDBWithKey(path, key string) (*sql.DB, error) {
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

	if _, err = db.Exec(fmt.Sprintf("PRAGMA kdf_iter = '%d'", kdfIterationsNumber)); err != nil {
		return nil, err
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

// OpenDB opens encrypted database using password.
func OpenDB(path, password string) (*sql.DB, error) {
	passhash := sha3.Sum256([]byte(password))
	return OpenDBWithKey(path, string(passhash[:]))
}
