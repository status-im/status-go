package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/types"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

const tableName = "wakuv2_keys"

type messagingPersistence struct {
	db *sql.DB
}

var _ types.Persistence = (*messagingPersistence)(nil)

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
		defer stmt.Close()

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

func (c *messagingPersistence) InsertPendingConfirmation(confirmation *messagingtypes.RawMessageConfirmation) error {
	_, err := c.db.Exec(`INSERT INTO raw_message_confirmations
		 (datasync_id, message_id, public_key)
		 VALUES
		 (?,?,?)`,
		confirmation.DataSyncID,
		confirmation.MessageID,
		confirmation.PublicKey,
	)
	return err
}

func (c *messagingPersistence) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *messagingtypes.ReceivedMessage) error {
	_, err := c.db.Exec(`INSERT INTO hash_ratchet_encrypted_messages(hash, sig, timestamp, topic, payload, dst, padding, group_id, key_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, m.Hash, m.Sig, m.Timestamp, m.Topic.Bytes(), m.Payload, m.Dst, m.Padding, groupID, keyID)
	return err
}

func (c *messagingPersistence) GetHashRatchetMessages(keyID []byte) ([]*messagingtypes.ReceivedMessage, error) {
	var messages []*messagingtypes.ReceivedMessage

	rows, err := c.db.Query(`SELECT hash, sig, timestamp, topic, payload, dst, padding FROM hash_ratchet_encrypted_messages WHERE key_id = ?`, keyID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var topic []byte
		message := &messagingtypes.ReceivedMessage{}

		err := rows.Scan(&message.Hash, &message.Sig, &message.Timestamp, &topic, &message.Payload, &message.Dst, &message.Padding)
		if err != nil {
			return nil, err
		}

		message.Topic = messagingtypes.BytesToContentTopic(topic)
		messages = append(messages, message)
	}

	return messages, nil
}

func (c *messagingPersistence) DeleteHashRatchetMessages(ids [][]byte) error {
	if len(ids) == 0 {
		return nil
	}

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}
	inVector := strings.Repeat("?, ", len(ids)-1) + "?"

	_, err := c.db.Exec("DELETE FROM hash_ratchet_encrypted_messages WHERE hash IN ("+inVector+")", idsArgs...) // nolint: gosec

	return err
}

func (c *messagingPersistence) DeleteHashRatchetMessagesOlderThan(timestamp int64) error {
	_, err := c.db.Exec("DELETE FROM hash_ratchet_encrypted_messages WHERE timestamp < ?", timestamp)
	return err
}

func (c *messagingPersistence) IsMessageAlreadyCompleted(hash []byte) (bool, error) {
	var alreadyCompleted int
	err := c.db.QueryRow("SELECT COUNT(*) FROM message_segments_completed WHERE hash = ?", hash).Scan(&alreadyCompleted)
	if err != nil {
		return false, err
	}
	return alreadyCompleted > 0, nil
}

func (c *messagingPersistence) SaveMessageSegment(segment *messagingtypes.SegmentMessage, sigPubKey *ecdsa.PublicKey, timestamp int64) error {
	sigPubKeyBlob := crypto.CompressPubkey(sigPubKey)

	_, err := c.db.Exec("INSERT INTO message_segments (hash, segment_index, segments_count, parity_segment_index, parity_segments_count, sig_pub_key, payload, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		segment.EntireMessageHash, segment.Index, segment.SegmentsCount, segment.ParitySegmentIndex, segment.ParitySegmentsCount, sigPubKeyBlob, segment.Payload, timestamp)

	return err
}

// Get ordered message segments for given hash
func (c *messagingPersistence) GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*messagingtypes.SegmentMessage, error) {
	sigPubKeyBlob := crypto.CompressPubkey(sigPubKey)

	rows, err := c.db.Query(`
		SELECT
			hash, segment_index, segments_count, parity_segment_index, parity_segments_count, payload
		FROM
			message_segments
		WHERE
			hash = ? AND sig_pub_key = ?
		ORDER BY
			(segments_count = 0) ASC, -- Prioritize segments_count > 0
			segment_index ASC,
			parity_segment_index ASC`,
		hash, sigPubKeyBlob)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []*messagingtypes.SegmentMessage
	for rows.Next() {
		segment := &messagingtypes.SegmentMessage{
			SegmentMessage: &protobuf.SegmentMessage{},
		}
		err := rows.Scan(&segment.EntireMessageHash, &segment.Index, &segment.SegmentsCount, &segment.ParitySegmentIndex, &segment.ParitySegmentsCount, &segment.Payload)
		if err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return segments, nil
}

func (c *messagingPersistence) RemoveMessageSegmentsOlderThan(timestamp int64) error {
	_, err := c.db.Exec("DELETE FROM message_segments WHERE timestamp < ?", timestamp)
	return err
}

func (c *messagingPersistence) CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error {
	tx, err := c.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	sigPubKeyBlob := crypto.CompressPubkey(sigPubKey)

	_, err = tx.Exec("DELETE FROM message_segments WHERE hash = ? AND sig_pub_key = ?", hash, sigPubKeyBlob)
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO message_segments_completed (hash, sig_pub_key, timestamp) VALUES (?,?,?)", hash, sigPubKeyBlob, timestamp)
	if err != nil {
		return err
	}

	return err
}

func (c *messagingPersistence) RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error {
	_, err := c.db.Exec("DELETE FROM message_segments_completed WHERE timestamp < ?", timestamp)
	return err
}

// MarkAsConfirmed marks all the messages with dataSyncID as confirmed and returns
// the messageIDs that can be considered confirmed.
// If atLeastOne is set it will return messageid if at least once of the messages
// sent has been confirmed
func (c *messagingPersistence) MarkAsConfirmed(dataSyncID []byte, atLeastOne bool) (messageID ethtypes.HexBytes, err error) {
	tx, err := c.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	confirmedAt := time.Now().Unix()
	_, err = tx.Exec(`UPDATE raw_message_confirmations SET confirmed_at = ? WHERE datasync_id = ? AND confirmed_at = 0`, confirmedAt, dataSyncID)
	if err != nil {
		return
	}

	// Select any tuple that has a message_id with a datasync_id = ? and that has just been confirmed
	rows, err := tx.Query(`SELECT message_id,confirmed_at FROM raw_message_confirmations WHERE message_id = (SELECT message_id FROM raw_message_confirmations WHERE datasync_id = ? LIMIT 1)`, dataSyncID)
	if err != nil {
		return
	}
	defer rows.Close()

	confirmedResult := true

	for rows.Next() {
		var confirmedAt int64
		err = rows.Scan(&messageID, &confirmedAt)
		if err != nil {
			return
		}
		confirmed := confirmedAt > 0

		if atLeastOne && confirmed {
			// We return, as at least one was confirmed
			return
		}

		confirmedResult = confirmedResult && confirmed
	}

	if !confirmedResult {
		messageID = nil
		return
	}

	return
}
