package publisher

import (
	"database/sql"
)

type Persistence interface {
	Get() (int64, error)
	Set(int64) error
}

type SQLLitePersistence struct {
	db *sql.DB
}

func NewSQLLitePersistence(db *sql.DB) *SQLLitePersistence {
	return &SQLLitePersistence{db: db}
}

func (s *SQLLitePersistence) Get() (int64, error) {
	var lastPublished int64
	statement := "SELECT last_published FROM contact_code_config LIMIT 1"
	err := s.db.QueryRow(statement).Scan(&lastPublished)

	if err != nil {
		return 0, err
	}

	return lastPublished, nil
}

func (s *SQLLitePersistence) Set(lastPublished int64) error {
	statement := "UPDATE contact_code_config SET last_published = ?"
	stmt, err := s.db.Prepare(statement)
	defer stmt.Close()

	if err != nil {
		return err
	}

	_, err = stmt.Exec(stmt, lastPublished)
	return err
}
