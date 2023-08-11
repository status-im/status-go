package dbsetup

import "database/sql"

type DatabaseInitializer interface {
	Initialize(path, password string, kdfIterationsNumber int) (*sql.DB, error)
}
