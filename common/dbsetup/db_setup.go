package dbsetup

import (
	"database/sql"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/sqlite"
)

type DatabaseInitializer interface {
	Initialize(path, password string, kdfIterationsNumber int) (*sql.DB, error)
}

// DecryptDatabase creates an unencrypted copy of the database and copies it
// over to the given directory
func DecryptDatabase(oldPath, newPath, password string, kdfIterationsNumber int) error {
	return sqlite.DecryptDB(oldPath, newPath, password, kdfIterationsNumber)
}

// EncryptDatabase creates an encrypted copy of the database and copies it to the
// user path
func EncryptDatabase(oldPath, newPath, password string, kdfIterationsNumber int, onStart func(), onEnd func()) error {
	return sqlite.EncryptDB(oldPath, newPath, password, kdfIterationsNumber, onStart, onEnd)
}

func ExportDB(path string, password string, kdfIterationsNumber int, newDbPath string, newPassword string, onStart func(), onEnd func()) error {
	return sqlite.ExportDB(path, password, kdfIterationsNumber, newDbPath, newPassword, onStart, onEnd)
}

func ChangeDatabasePassword(path string, password string, kdfIterationsNumber int, newPassword string, onStart func(), onEnd func()) error {
	return sqlite.ChangeEncryptionKey(path, password, kdfIterationsNumber, newPassword, onStart, onEnd)
}

// GetDBFilename takes an instance of sql.DB and returns the filename of the "main" database
func GetDBFilename(db *sql.DB) (string, error) {
	if db == nil {
		logger := log.New()
		logger.Warn("GetDBFilename was passed a nil pointer sql.DB")
		return "", nil
	}

	var i, category, filename string
	rows, err := db.Query("PRAGMA database_list;")
	if err != nil {
		return "", err
	}

	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&i, &category, &filename)
		if err != nil {
			return "", err
		}

		// The "main" database is the one we care about
		if category == "main" {
			return filename, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	return "", errors.New("no main database found")
}
