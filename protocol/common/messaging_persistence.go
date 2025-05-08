package common

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/status-im/status-go/messaging"
)

const tableName = "wakuv2_keys"

type messagingPersistence struct {
	db *sql.DB
}

var _ messaging.Persistence = (*messagingPersistence)(nil)

func NewMessagingPersistence(db *sql.DB) *messagingPersistence {
	return &messagingPersistence{db: db}
}

func (s *messagingPersistence) AddWakuKey(chatID string, key []byte) error {
	statement := fmt.Sprintf("INSERT INTO %s(chat_id, key) VALUES(?, ?)", tableName) // nolint:gosec
	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(chatID, key)
	return err
}

func (s *messagingPersistence) WakuKeys() (map[string][]byte, error) {
	keys := make(map[string][]byte)

	statement := fmt.Sprintf("SELECT chat_id, key FROM %s", tableName) // nolint: gosec

	stmt, err := s.db.Prepare(statement)
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

func (c *messagingPersistence) MessageCacheClear() error {
	_, err := c.db.Exec("DELETE FROM transport_message_cache")
	return err
}

func (c *messagingPersistence) MessageCacheHits(ids []string) (map[string]bool, error) {
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

func (c *messagingPersistence) MessageCacheAdd(ids []string, timestamp uint64) (err error) {
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

func (c *messagingPersistence) MessageCacheClearOlderThan(timestamp uint64) error {
	_, err := c.db.Exec(`DELETE FROM transport_message_cache WHERE timestamp < ?`, timestamp)
	return err
}
