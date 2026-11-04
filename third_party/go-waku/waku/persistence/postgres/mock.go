package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/avast/retry-go/v4"
)

var dbUrlTemplate = "postgres://postgres@localhost:%s/%s?sslmode=disable"

func ResetDefaultTestPostgresDB(dropDBUrl string) error {
	db, err := sql.Open("pgx", dropDBUrl)
	if err != nil {
		return err
	}

	deletePrevConnectionsSql := `
	SELECT pid, pg_terminate_backend(pid)
	FROM pg_stat_activity
	WHERE datname in ('template1', 'postgres') AND pid <> pg_backend_pid();`
	_, err = db.Exec(deletePrevConnectionsSql)
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

func NewMockPgDB() *sql.DB {
	mockPgDBPort := os.Getenv("TEST_DB_PORT")
	if mockPgDBPort == "" {
		mockPgDBPort = "5432"
	}
	//
	err := retry.Do(
		func() error {

			dropDBUrl := fmt.Sprintf(dbUrlTemplate, mockPgDBPort, "template1")
			if err := ResetDefaultTestPostgresDB(dropDBUrl); err != nil {
				return err
			}
			return nil
		}, retry.Attempts(3))
	if err != nil {
		log.Fatalf("an error '%s' while reseting the db", err)
	}
	mockDBUrl := fmt.Sprintf(dbUrlTemplate, mockPgDBPort, "postgres")
	db, err := sql.Open("pgx", mockDBUrl)
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	return db
}
