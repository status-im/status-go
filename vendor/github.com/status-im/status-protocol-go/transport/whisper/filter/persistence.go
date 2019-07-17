package filter

import (
	"database/sql"

	"github.com/status-im/status-protocol-go/sqlite"
	migrations "github.com/status-im/status-protocol-go/transport/whisper/internal/sqlite"
)

type sqlitePersistence struct {
	db *sql.DB
}

func newSQLitePersistence(db *sql.DB) (*sqlitePersistence, error) {
	err := sqlite.ApplyMigrations(
		db,
		migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	if err != nil {
		return nil, nil
	}
	return &sqlitePersistence{db: db}, nil
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
