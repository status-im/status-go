package publisher

import (
	"database/sql"
	"encoding/hex"
	"sync"
)

type sqlitePersistence struct {
	db            *sql.DB
	lastAcksMutex sync.Mutex
	lastAcks      map[string]int64
}

func newSQLitePersistence(db *sql.DB) *sqlitePersistence {
	return &sqlitePersistence{
		db:       db,
		lastAcks: make(map[string]int64),
	}
}

func (s *sqlitePersistence) lastPublished() (int64, error) {
	var lastPublished int64
	statement := "SELECT last_published FROM contact_code_config LIMIT 1"
	err := s.db.QueryRow(statement).Scan(&lastPublished)
	if err != nil {
		return 0, err
	}
	return lastPublished, nil
}

func (s *sqlitePersistence) setLastPublished(lastPublished int64) error {
	statement := "UPDATE contact_code_config SET last_published = ?"
	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(lastPublished)
	return err
}

func (s *sqlitePersistence) lastAck(identity []byte) (int64, error) {
	s.lastAcksMutex.Lock()
	defer s.lastAcksMutex.Unlock()
	return s.lastAcks[hex.EncodeToString(identity)], nil
}

func (s *sqlitePersistence) setLastAck(identity []byte, lastAck int64) {
	s.lastAcksMutex.Lock()
	defer s.lastAcksMutex.Unlock()
	s.lastAcks[hex.EncodeToString(identity)] = lastAck
}
