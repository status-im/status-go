package publisher

import (
	"database/sql"
	"fmt"
	"sync"
)

type Persistence interface {
	GetLastPublished() (int64, error)
	SetLastPublished(int64) error
	GetLastAcked(identity []byte) (int64, error)
	SetLastAcked(identity []byte, lastAcked int64) error
}

type SQLLitePersistence struct {
	db             *sql.DB
	lastAcked      map[string]int64
	lastAckedMutex sync.Mutex
}

func NewSQLLitePersistence(db *sql.DB) *SQLLitePersistence {
	return &SQLLitePersistence{
		db:             db,
		lastAcked:      make(map[string]int64),
		lastAckedMutex: sync.Mutex{},
	}
}

func (s *SQLLitePersistence) GetLastPublished() (int64, error) {
	var lastPublished int64
	statement := "SELECT last_published FROM contact_code_config LIMIT 1"
	err := s.db.QueryRow(statement).Scan(&lastPublished)

	if err != nil {
		return 0, err
	}

	return lastPublished, nil
}

func (s *SQLLitePersistence) SetLastPublished(lastPublished int64) error {
	statement := "UPDATE contact_code_config SET last_published = ?"
	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(lastPublished)
	return err
}

func (s *SQLLitePersistence) GetLastAcked(identity []byte) (int64, error) {
	s.lastAckedMutex.Lock()
	defer s.lastAckedMutex.Unlock()
	return s.lastAcked[fmt.Sprintf("%x", identity)], nil
}

func (s *SQLLitePersistence) SetLastAcked(identity []byte, lastAcked int64) error {
	s.lastAckedMutex.Lock()
	defer s.lastAckedMutex.Unlock()
	s.lastAcked[fmt.Sprintf("%x", identity)] = lastAcked
	return nil
}
