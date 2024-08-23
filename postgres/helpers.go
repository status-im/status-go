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
	defer func() {
		_ = db.Close()
	}()

	// Drop current and prevent any future connections. Used in tests. Details here:
	// https://stackoverflow.com/questions/17449420/postgresql-unable-to-drop-database-because-of-some-auto-connections-to-db
	_, err = db.Exec("REVOKE CONNECT ON DATABASE postgres FROM public;")
	if err != nil {
		return err
	}

	_, err = db.Exec("SELECT pid, pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'postgres' AND pid <> pg_backend_pid();")
	if err != nil {
		return err
	}

	_, err = db.Exec("DROP DATABASE IF EXISTS postgres;")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE DATABASE postgres;")
	if err != nil {
		return err
	}

	_, err = db.Exec("GRANT CONNECT ON DATABASE postgres TO public;")
	if err != nil {
		return err
	}

	return nil
}
