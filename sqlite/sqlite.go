package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation

	"github.com/status-im/status-go/protocol/sqlite"
)

const (
	// The reduced number of kdf iterations (for performance reasons) which is
	// used as the default value
	// https://github.com/status-im/status-go/pull/1343
	// https://notes.status.im/i8Y_l7ccTiOYq09HVgoFwA
	ReducedKDFIterationsNumber = 3200

	// WALMode for sqlite.
	WALMode      = "wal"
	InMemoryPath = ":memory:"
)

// DecryptDB completely removes the encryption from the db
func DecryptDB(oldPath string, newPath string, key string, kdfIterationsNumber int) error {

	db, err := openDB(oldPath, key, kdfIterationsNumber)
	if err != nil {
		return err
	}

	_, err = db.Exec(`ATTACH DATABASE '` + newPath + `' AS plaintext KEY ''`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`SELECT sqlcipher_export('plaintext')`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`DETACH DATABASE plaintext`)
	return err
}

// EncryptDB takes a plaintext database and adds encryption
func EncryptDB(unencryptedPath string, encryptedPath string, key string, kdfIterationsNumber int) error {
	_ = os.Remove(encryptedPath)

	db, err := OpenUnecryptedDB(unencryptedPath)
	if err != nil {
		return err
	}

	_, err = db.Exec(`ATTACH DATABASE '` + encryptedPath + `' AS encrypted KEY '` + key + `'`)
	if err != nil {
		return err
	}

	if kdfIterationsNumber <= 0 {
		kdfIterationsNumber = sqlite.ReducedKDFIterationsNumber
	}

	_, err = db.Exec(fmt.Sprintf("PRAGMA encrypted.kdf_iter = '%d'", kdfIterationsNumber))
	if err != nil {
		return err
	}

	_, err = db.Exec(`SELECT sqlcipher_export('encrypted')`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`DETACH DATABASE encrypted`)
	return err
}

func openCipher(db *sql.DB, key string, kdfIterationsNumber int, inMemory bool) error {
	keyString := fmt.Sprintf("PRAGMA key = '%s'", key)
	if _, err := db.Exec(keyString); err != nil {
		return errors.New("failed to set key pragma")
	}

	if kdfIterationsNumber <= 0 {
		kdfIterationsNumber = sqlite.ReducedKDFIterationsNumber
	}

	if _, err := db.Exec(fmt.Sprintf("PRAGMA kdf_iter = '%d'", kdfIterationsNumber)); err != nil {
		return err
	}

	// readers do not block writers and faster i/o operations
	// https://www.sqlite.org/draft/wal.html
	// must be set after db is encrypted
	var mode string
	err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode)
	if err != nil {
		return err
	}
	if mode != WALMode && !inMemory {
		return fmt.Errorf("unable to set journal_mode to WAL. actual mode %s", mode)
	}

	return nil
}

func openDB(path string, key string, kdfIterationsNumber int) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}

	err = openCipher(db, key, kdfIterationsNumber, path == InMemoryPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// OpenDB opens not-encrypted database.
func OpenDB(path string, key string, kdfIterationsNumber int) (*sql.DB, error) {
	return openDB(path, key, kdfIterationsNumber)
}

// OpenUnecryptedDB opens database with setting PRAGMA key.
func OpenUnecryptedDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
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
	if mode != WALMode {
		return nil, fmt.Errorf("unable to set journal_mode to WAL. actual mode %s", mode)
	}

	return db, nil
}

func ChangeEncryptionKey(path string, key string, kdfIterationsNumber int, newKey string) error {
	if kdfIterationsNumber <= 0 {
		kdfIterationsNumber = sqlite.ReducedKDFIterationsNumber
	}

	db, err := openDB(path, key, kdfIterationsNumber)

	if err != nil {
		return err
	}

	resetKeyString := fmt.Sprintf("PRAGMA rekey = '%s'", newKey)
	if _, err = db.Exec(resetKeyString); err != nil {
		return errors.New("failed to set rekey pragma")
	}

	return nil
}
