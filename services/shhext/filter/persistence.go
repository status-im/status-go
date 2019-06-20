package filter

import (
	"database/sql"
	"strings"
)

type Persistence interface {
	Get(chatID string) ([]byte, error)
	Add(chatID string, key []byte) error
	GetMany(chatIDs []string) (map[string][]byte, error)
	All() (map[string][]byte, error)
}

type SQLLitePersistence struct {
	db *sql.DB
}

func NewSQLLitePersistence(db *sql.DB) *SQLLitePersistence {
	return &SQLLitePersistence{db: db}
}

func (s *SQLLitePersistence) Get(chatID string) ([]byte, error) {
	var key []byte
	statement := "SELECT key FROM whisper_keys LIMIT 1"
	err := s.db.QueryRow(statement).Scan(&key)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return key, nil
}

func (s *SQLLitePersistence) All() (map[string][]byte, error) {

	keys := make(map[string][]byte)

	statement := "SELECT chat_id, key FROM whisper_keys"

	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	for rows.Next() {
		var chatID string
		var key []byte
		err := rows.Scan(&chatID, &key)
		if err != nil {
			return nil, err
		}
		keys[chatID] = key
	}

	return keys, nil
}

func (s *SQLLitePersistence) GetMany(chatIDs []string) (map[string][]byte, error) {

	keys := make(map[string][]byte)
	if len(chatIDs) == 0 {
		return keys, nil
	}

	args := make([]interface{}, len(chatIDs))

	for i, chatID := range chatIDs {
		args[i] = chatID
	}

	statement := "SELECT chat_id, key FROM whisper_keys WHERE chat_id IN (?" + strings.Repeat(",?", len(chatIDs)-1) + ")"

	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	for rows.Next() {
		var chatID string
		var key []byte
		err := rows.Scan(&chatID, &key)
		if err != nil {
			return nil, err
		}
		keys[chatID] = key
	}

	return keys, nil
}

func (s *SQLLitePersistence) Add(chatID string, key []byte) error {
	statement := "INSERT INTO whisper_keys(chat_id,key) VALUES(?,?)"
	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(chatID, key)
	return err
}
