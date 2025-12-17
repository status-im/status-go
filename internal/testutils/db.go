package testutils

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/sqlite"
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

type TestDBInitializer struct {
	assetSources []*bindata.AssetSource
}

func NewTestDBInitializer(assetSource []*bindata.AssetSource) TestDBInitializer {
	return TestDBInitializer{
		assetSources: assetSource,
	}
}

func (dbi TestDBInitializer) Initialize(dbPath string, password string, kdfIterations int) (*sql.DB, error) {
	db, err := sqlite.OpenDB(dbPath, password, kdfIterations)
	if err != nil {
		return nil, err
	}

	for _, as := range dbi.assetSources {
		err = sqlite.Migrate(db, as, sqlite.MigrateOptions{
			MigrationTableName: "status_schema_migrations_" + fmt.Sprintf("%x", uuid.New()),
		})
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}
