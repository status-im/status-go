package postgres

import (
	"database/sql"
	"fmt"
	"os"

	// Import postgres driver
	_ "github.com/lib/pq"
)

var (
	DefaultTestDBHost = GetEnvDefault("TEST_POSTGRES_HOST", "localhost")
	DefaultTestDBPort = GetEnvDefault("TEST_POSTGRES_PORT", "5432")
	DefaultTestURI    = fmt.Sprintf("postgres://postgres@%s:%s/postgres?sslmode=disable", DefaultTestDBHost, DefaultTestDBPort)
	DropTableURI      = fmt.Sprintf("postgres://postgres@%s:%s/template1?sslmode=disable", DefaultTestDBHost, DefaultTestDBPort)
)

func GetEnvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func ResetDefaultTestPostgresDB() error {
	db, err := sql.Open("postgres", DropTableURI)
	if err != nil {
		return err
	}

	_, err = db.Exec("DROP DATABASE IF EXISTS postgres;")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE DATABASE postgres;")
	return err
}
