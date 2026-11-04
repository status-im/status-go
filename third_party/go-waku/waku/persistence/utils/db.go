package utils

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/postgres"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	"go.uber.org/zap"
)

func validateDBUrl(val string) error {
	matched, err := regexp.Match(`^[\w\+]+:\/\/[\w\/\\\.\:\@]+\?{0,1}.*$`, []byte(val))
	if !matched || err != nil {
		return errors.New("invalid db url option format")
	}
	return nil
}

// DBSettings hold db specific configuration settings required during the db initialization
type DBSettings struct {
	// TODO: add any DB specific setting here
}

// ParseURL will return a database connection, and migration function that should be used depending on a database connection string
func ParseURL(databaseURL string, dbSettings DBSettings, logger *zap.Logger) (*sql.DB, func(*sql.DB, *zap.Logger) error, error) {
	var db *sql.DB
	var migrationFn func(*sql.DB, *zap.Logger) error
	var err error

	logger = logger.Named("db-setup")

	dbURL := ""
	if databaseURL != "" {
		err := validateDBUrl(databaseURL)
		if err != nil {
			return nil, nil, err
		}
		dbURL = databaseURL
	} else {
		// In memoryDB
		dbURL = "sqlite3://:memory:"
	}

	dbURLParts := strings.Split(dbURL, "://")
	dbEngine := dbURLParts[0]
	dbParams := dbURLParts[1]
	switch dbEngine {
	case "sqlite3":
		db, err = sqlite.NewDB(dbParams, logger)
		migrationFn = sqlite.Migrations
	case "postgres", "postgresql":
		db, err = postgres.NewDB(dbURL, logger)
		migrationFn = postgres.Migrations
	default:
		err = errors.New("unsupported database engine")
	}

	if err != nil {
		return nil, nil, err
	}

	return db, migrationFn, nil

}

func NewQueries(tbl string, db *sql.DB) (*persistence.Queries, error) {
	switch persistence.GetDriverType(db) {
	case persistence.SQLiteDriver:
		return sqlite.NewQueries(tbl, db)
	case persistence.PostgresDriver:
		return postgres.NewQueries(tbl, db)
	}
	return nil, errors.New("unsupported database engine")
}
