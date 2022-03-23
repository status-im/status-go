package appdatabase

import (
	"database/sql"
	"io/ioutil"
	"os"

	"github.com/status-im/status-go/protocol/sqlite"
)

// SetupTestSQLDB creates a temporary sqlite database file, initialises and then returns with a teardown func
func SetupTestSQLDB(prefix string) (*sql.DB, func() error, error) {
	tmpfile, err := ioutil.TempFile("", prefix)
	if err != nil {
		return nil, nil, err
	}
	db, err := InitializeDB(tmpfile.Name(), prefix)
	if err != nil {
		return nil, nil, err
	}

	return db, func() error {
		err := db.Close()
		if err != nil {
			return err
		}
		return os.Remove(tmpfile.Name())
	}, nil
}

func SetupTestMemorySQLDB(prefix string) (*sql.DB, error) {
	db, err := InitializeDB(sqlite.InMemoryPath, prefix)
	if err != nil {
		return nil, err
	}

	return db, nil
}
