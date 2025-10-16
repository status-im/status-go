package helpers

import (
	"database/sql"
	"io/ioutil"
	"os"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/sqlite"
)

const kdfIterationsNumberForTests = 1

// SetupTestSQLDB creates a temporary sqlite database file, initialises and then returns with a teardown func
func SetupTestSQLDB(dbInit dbsetup.DatabaseInitializer, prefix string) (*sql.DB, func() error, error) {
	tmpfile, err := ioutil.TempFile("", prefix)
	if err != nil {
		return nil, nil, err
	}

	db, err := dbInit.Initialize(tmpfile.Name(), "password", kdfIterationsNumberForTests)
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

func SetupTestMemorySQLDB(dbInit dbsetup.DatabaseInitializer) (*sql.DB, error) {
	db, err := dbInit.Initialize(dbsetup.InMemoryPath, "password", kdfIterationsNumberForTests)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func SetupTestMemorySQLAccountsDB(dbInit dbsetup.DatabaseInitializer) (*sql.DB, error) {
	db, err := multiaccounts.InitializeDB(dbsetup.InMemoryPath)
	if err != nil {
		return nil, err
	}

	return db.DB(), nil
}

func ColumnExists(db *sql.DB, tableName string, columnName string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var cid int
	var name string
	var dataType string
	var notNull bool
	var dFLTValue sql.NullString
	var pk int

	for rows.Next() {
		err := rows.Scan(&cid, &name, &dataType, &notNull, &dFLTValue, &pk)
		if err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}

	if rows.Err() != nil {
		return false, rows.Err()
	}

	return false, nil
}

type TestDBInitializer struct {
	assetSource *bindata.AssetSource
}

func NewTestDBInitializer(assetSource *bindata.AssetSource) TestDBInitializer {
	return TestDBInitializer{
		assetSource: assetSource,
	}
}

func (dbi TestDBInitializer) Initialize(dbPath string, password string, kdfIterations int) (*sql.DB, error) {
	db, err := sqlite.OpenDB(dbPath, password, kdfIterations)
	if err != nil {
		return nil, err
	}

	if dbi.assetSource != nil {
		err = sqlite.Migrate(db, dbi.assetSource, sqlite.MigrateOptions{})
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}
