package localnotifications

import "database/sql"

type Database struct {
	db      *sql.DB
	network uint64
}

func NewDB(db *sql.DB, network uint64) *Database {
	return &Database{db: db, network: network}
}
