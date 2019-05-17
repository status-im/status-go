package db

import (
	"database/sql"
	"fmt"
	"os"

	sqlite "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation
	"github.com/status-im/migrate"
	"github.com/status-im/migrate/database/sqlcipher"
	"github.com/status-im/migrate/source/go_bindata"
	"github.com/status-im/status-go/services/shhext/chat/db/migrations"
)

const exportDB = "SELECT sqlcipher_export('newdb')"

// The default number of kdf iterations in sqlcipher (from version 3.0.0)
// https://github.com/sqlcipher/sqlcipher/blob/fda4c68bb474da7e955be07a2b807bda1bb19bd2/CHANGELOG.md#300---2013-11-05
// https://www.zetetic.net/sqlcipher/sqlcipher-api/#kdf_iter
const defaultKdfIterationsNumber = 64000

// The reduced number of kdf iterations (for performance reasons) which is
// currently used for derivation of the database key
// https://github.com/status-im/status-go/pull/1343
// https://notes.status.im/i8Y_l7ccTiOYq09HVgoFwA
const KdfIterationsNumber = 3200

func MigrateDBFile(oldPath string, newPath string, oldKey string, newKey string) error {
	_, err := os.Stat(oldPath)

	// No files, nothing to do
	if os.IsNotExist(err) {
		return nil
	}

	// Any other error, throws
	if err != nil {
		return err
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	db, err := Open(newPath, oldKey, defaultKdfIterationsNumber)
	if err != nil {
		return err
	}

	keyString := fmt.Sprintf("PRAGMA rekey = '%s'", newKey)

	if _, err = db.Exec(keyString); err != nil {
		return err
	}

	return nil

}

// MigrateDBKeyKdfIterations changes the number of kdf iterations executed
// during the database key derivation. This change is necessary because
// of performance reasons.
// https://github.com/status-im/status-go/pull/1343
// `sqlcipher_export` is used for migration, check out this link for details:
// https://www.zetetic.net/sqlcipher/sqlcipher-api/#sqlcipher_export
func MigrateDBKeyKdfIterations(oldPath string, newPath string, key string) error {
	_, err := os.Stat(oldPath)

	// No files, nothing to do
	if os.IsNotExist(err) {
		return nil
	}

	// Any other error, throws
	if err != nil {
		return err
	}

	isEncrypted, err := sqlite.IsEncrypted(oldPath)
	if err != nil {
		return err
	}

	// Nothing to do, move db to the next migration
	if !isEncrypted {
		return os.Rename(oldPath, newPath)
	}

	db, err := Open(oldPath, key, defaultKdfIterationsNumber)
	if err != nil {
		return err
	}

	attach := fmt.Sprintf(
		"ATTACH DATABASE '%s' AS newdb KEY '%s'",
		newPath,
		key)

	if _, err = db.Exec(attach); err != nil {
		return err
	}

	changeKdfIter := fmt.Sprintf(
		"PRAGMA newdb.kdf_iter = %d",
		KdfIterationsNumber)

	if _, err = db.Exec(changeKdfIter); err != nil {
		return err
	}

	if _, err = db.Exec(exportDB); err != nil {
		return err
	}

	if err = db.Close(); err != nil {
		return err
	}

	return os.Remove(oldPath)
}

// EncryptDatabase encrypts an unencrypted database with key
func EncryptDatabase(oldPath string, newPath string, key string) error {
	_, err := os.Stat(oldPath)

	// No files, nothing to do
	if os.IsNotExist(err) {
		return nil
	}

	// Any other error, throws
	if err != nil {
		return err
	}

	isEncrypted, err := sqlite.IsEncrypted(oldPath)
	if err != nil {
		return err
	}

	// Nothing to do, already encrypted
	if isEncrypted {
		return os.Rename(oldPath, newPath)
	}

	db, err := Open(oldPath, "", defaultKdfIterationsNumber)
	if err != nil {
		return err
	}

	attach := fmt.Sprintf(
		"ATTACH DATABASE '%s' AS newdb KEY '%s'",
		newPath,
		key)

	if _, err = db.Exec(attach); err != nil {
		return err
	}

	changeKdfIter := fmt.Sprintf(
		"PRAGMA newdb.kdf_iter = %d",
		KdfIterationsNumber)

	if _, err = db.Exec(changeKdfIter); err != nil {
		return err
	}

	if _, err = db.Exec(exportDB); err != nil {
		return err
	}

	if err = db.Close(); err != nil {
		return err
	}

	return os.Remove(oldPath)
}

func migrateDB(db *sql.DB) error {
	resources := bindata.Resource(
		migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)

	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	driver, err := sqlcipher.WithInstance(db, &sqlcipher.Config{})
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

func Open(path string, key string, kdfIter int) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	keyString := fmt.Sprintf("PRAGMA key = '%s'", key)

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}

	if _, err = db.Exec(keyString); err != nil {
		return nil, err
	}

	kdfString := fmt.Sprintf("PRAGMA kdf_iter = '%d'", kdfIter)

	if _, err = db.Exec(kdfString); err != nil {
		return nil, err
	}

	// Migrate db
	if err = migrateDB(db); err != nil {
		return nil, err
	}

	return db, nil
}
