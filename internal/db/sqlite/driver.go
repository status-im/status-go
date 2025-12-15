package sqlite

import "database/sql"

// statementCreator allows to pass transaction or database to use in consumer.
type StatementCreator interface {
	Prepare(query string) (*sql.Stmt, error)
}

type StatementExecutor interface {
	StatementCreator
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}
