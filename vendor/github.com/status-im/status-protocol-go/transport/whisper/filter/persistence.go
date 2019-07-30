package filter

import (
	"database/sql"
)

type sqlitePersistence struct {
	db *sql.DB
}

func newSQLitePersistence(db *sql.DB) *sqlitePersistence {
	return &sqlitePersistence{db: db}
}

func (s *sqlitePersistence) Add(chatID string, key []byte) error {
	statement := "INSERT INTO whisper_keys(chat_id, key) VALUES(?, ?)"
	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(chatID, key)
	return err
}

func (s *sqlitePersistence) All() (map[string][]byte, error) {
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
		var (
			chatID string
			key    []byte
		)

		err := rows.Scan(&chatID, &key)
		if err != nil {
			return nil, err
		}
		keys[chatID] = key
	}

	return keys, nil
}
