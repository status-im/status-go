package postgres

import (
	"database/sql"
	// Import postgres driver
	_ "github.com/lib/pq"
)

const (
	DefaultTestURI = "postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable"
	DropTableURI   = "postgres://postgres@127.0.0.1:5432/template1?sslmode=disable"
)

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
