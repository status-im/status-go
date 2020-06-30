package protocol

import (
	"database/sql"
)

type Persistence struct {
	db *sql.DB
}

func NewPersistence(db *sql.DB) *Persistence {
	return &Persistence{db: db}
}
