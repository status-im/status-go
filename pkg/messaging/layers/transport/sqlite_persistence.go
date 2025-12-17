package transport

import (
	"context"
	"database/sql"
	"strings"
)

type SQLiteKeysPersistence struct {
	db *sql.DB
}

func NewSQLiteKeysPersistence(db *sql.DB) *SQLiteKeysPersistence {
	return &SQLiteKeysPersistence{db: db}
}

func (s *SQLiteKeysPersistence) Add(chatID string, key []byte) error {
	stmt, err := s.db.Prepare("INSERT INTO wakuv2_keys(chat_id, key) VALUES(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(chatID, key)
	return err
}

func (s *SQLiteKeysPersistence) All() (map[string][]byte, error) {
	keys := make(map[string][]byte)

	stmt, err := s.db.Prepare("SELECT chat_id, key FROM wakuv2_keys")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

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

type SQLiteProcessedMessageIDsCachePersistence struct {
	db *sql.DB
}

func NewSQLiteProcessedMessageIDsCachePersistence(db *sql.DB) *SQLiteProcessedMessageIDsCachePersistence {
	return &SQLiteProcessedMessageIDsCachePersistence{db: db}
}

func (c *SQLiteProcessedMessageIDsCachePersistence) Clear() error {
	_, err := c.db.Exec("DELETE FROM transport_message_cache")
	return err
}

func (c *SQLiteProcessedMessageIDsCachePersistence) Hits(ids []string) (map[string]bool, error) {
	hits := make(map[string]bool)

	// Split the results into batches of 999 items.
	// To prevent excessive memory allocations, the maximum value of a host parameter number
	// is SQLITE_MAX_VARIABLE_NUMBER, which defaults to 999
	batch := 999
	for i := 0; i < len(ids); i += batch {
		j := i + batch
		if j > len(ids) {
			j = len(ids)
		}

		currentBatch := ids[i:j]

		idsArgs := make([]interface{}, 0, len(currentBatch))
		for _, id := range currentBatch {
			idsArgs = append(idsArgs, id)
		}

		inVector := strings.Repeat("?, ", len(currentBatch)-1) + "?"
		query := "SELECT id FROM transport_message_cache WHERE id IN (" + inVector + ")" // nolint: gosec

		rows, err := c.db.Query(query, idsArgs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			err := rows.Scan(&id)
			if err != nil {
				return nil, err
			}
			hits[id] = true
		}
	}

	return hits, nil
}

func (c *SQLiteProcessedMessageIDsCachePersistence) Add(ids []string, timestamp uint64) (err error) {
	var tx *sql.Tx
	tx, err = c.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	for _, id := range ids {

		var stmt *sql.Stmt
		stmt, err = tx.Prepare(`INSERT INTO transport_message_cache(id,timestamp) VALUES (?, ?)`)
		if err != nil {
			return
		}

		_, err = stmt.Exec(id, timestamp)
		if err != nil {
			return
		}
	}

	return
}

func (c *SQLiteProcessedMessageIDsCachePersistence) Clean(timestamp uint64) error {
	_, err := c.db.Exec(`DELETE FROM transport_message_cache WHERE timestamp < ?`, timestamp)
	return err
}
