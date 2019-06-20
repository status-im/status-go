package filter

import (
	"database/sql"
)

type Persistence interface {
	Add(chatID string, key []byte) error
	All() (map[string][]byte, error)
}

type SQLLitePersistence struct {
	db *sql.DB
}

func NewSQLLitePersistence(db *sql.DB) *SQLLitePersistence {
	return &SQLLitePersistence{db: db}
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
