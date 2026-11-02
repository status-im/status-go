package reliability

import (
	"database/sql"
	"fmt"

	"github.com/waku-org/sds-go-bindings/sds"
)

type SQLitePersistence struct {
	db *sql.DB
}

func NewSQLitePersistence(db *sql.DB) sds.PersistenceCallbacks {
	return &SQLitePersistence{db: db}
}

func (p *SQLitePersistence) SaveChannelMeta(channelID string, meta []byte) error {
	_, err := p.db.Exec(
		`INSERT INTO sds_channel_metadata (channel_id, metadata)
		VALUES (?, ?)
		ON CONFLICT(channel_id) DO UPDATE SET metadata = excluded.metadata`,
		channelID, meta,
	)
	if err != nil {
		return fmt.Errorf("save SDS channel metadata: %w", err)
	}
	return nil
}

func (p *SQLitePersistence) UpdateHistory(channelID string, update sds.HistoryUpdate) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("begin SDS history update: %w", err)
	}
	for _, messageID := range update.Evict {
		if messageID == "" {
			_ = tx.Rollback()
			return fmt.Errorf("evict SDS history: empty message ID")
		}
		if _, err = tx.Exec(
			`DELETE FROM sds_channel_history WHERE channel_id = ? AND message_id = ?`,
			channelID, messageID,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("evict SDS history message %q: %w", messageID, err)
		}
		if _, err = tx.Exec(
			`DELETE FROM sds_retrieval_hints WHERE channel_id = ? AND message_id = ?`,
			channelID, messageID,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("evict SDS retrieval hints for message %q: %w", messageID, err)
		}
	}
	for _, message := range update.Append {
		if message.MessageID == "" {
			_ = tx.Rollback()
			return fmt.Errorf("append SDS history: empty message ID")
		}
		if len(message.Data) == 0 {
			_ = tx.Rollback()
			return fmt.Errorf("append SDS history message %q: empty serialized message", message.MessageID)
		}
		if _, err = tx.Exec(
			`INSERT INTO sds_channel_history (channel_id, message_id, lamport_timestamp, message)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(channel_id, message_id) DO UPDATE SET
				lamport_timestamp = excluded.lamport_timestamp,
				message = excluded.message`,
			channelID, message.MessageID, message.LamportTimestamp, message.Data,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("append SDS history message %q: %w", message.MessageID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit SDS history update: %w", err)
	}
	return nil
}

func (p *SQLitePersistence) LoadChannel(channelID string) (sds.ChannelData, error) {
	data := sds.ChannelData{
		Meta:           []byte{},
		MessageHistory: make([][]byte, 0),
	}
	err := p.db.QueryRow(
		`SELECT metadata FROM sds_channel_metadata WHERE channel_id = ?`,
		channelID,
	).Scan(&data.Meta)
	if err != nil && err != sql.ErrNoRows {
		return sds.ChannelData{}, fmt.Errorf("load SDS channel metadata: %w", err)
	}

	rows, err := p.db.Query(
		`SELECT message FROM sds_channel_history
		WHERE channel_id = ?
		ORDER BY lamport_timestamp, message_id`,
		channelID,
	)
	if err != nil {
		return sds.ChannelData{}, fmt.Errorf("load SDS message history: %w", err)
	}
	for rows.Next() {
		var message []byte
		if err := rows.Scan(&message); err != nil {
			_ = rows.Close()
			return sds.ChannelData{}, fmt.Errorf("scan SDS message history: %w", err)
		}
		data.MessageHistory = append(data.MessageHistory, message)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return sds.ChannelData{}, fmt.Errorf("read SDS message history: %w", err)
	}
	if err := rows.Close(); err != nil {
		return sds.ChannelData{}, fmt.Errorf("close SDS message history query: %w", err)
	}
	return data, nil
}

func (p *SQLitePersistence) DropChannel(channelID string) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("begin SDS channel drop: %w", err)
	}
	if _, err := tx.Exec(
		`DELETE FROM sds_channel_history WHERE channel_id = ?`,
		channelID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("drop SDS channel history: %w", err)
	}
	if _, err := tx.Exec(
		`DELETE FROM sds_retrieval_hints WHERE channel_id = ?`,
		channelID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("drop SDS retrieval hints: %w", err)
	}
	if _, err := tx.Exec(
		`DELETE FROM sds_channel_metadata WHERE channel_id = ?`,
		channelID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("drop SDS channel metadata: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit SDS channel drop: %w", err)
	}
	return nil
}

type sqliteSDSRetrievalHintPersistence struct {
	db *sql.DB
}

var _ SDSRetrievalHintPersistence = (*sqliteSDSRetrievalHintPersistence)(nil)

// NewSQLiteRetrievalHintPersistence creates durable SDS retrieval-hint storage.
func NewSQLiteRetrievalHintPersistence(db *sql.DB) SDSRetrievalHintPersistence {
	return &sqliteSDSRetrievalHintPersistence{db: db}
}

func (p *sqliteSDSRetrievalHintPersistence) SaveSDSRetrievalHints(channelID, messageID string, envelopeHashes [][]byte) error {
	if channelID == "" {
		return fmt.Errorf("save SDS retrieval hints: empty channel ID")
	}
	if messageID == "" {
		return fmt.Errorf("save SDS retrieval hints: empty message ID")
	}
	if len(envelopeHashes) == 0 {
		return fmt.Errorf("save SDS retrieval hints for message %q: no envelope hashes", messageID)
	}

	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("begin SDS retrieval hint save: %w", err)
	}
	for _, hash := range envelopeHashes {
		if len(hash) == 0 {
			_ = tx.Rollback()
			return fmt.Errorf("save SDS retrieval hints for message %q: empty envelope hash", messageID)
		}
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO sds_retrieval_hints (channel_id, message_id, envelope_hash)
			VALUES (?, ?, ?)`,
			channelID, messageID, hash,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("save SDS retrieval hint for message %q: %w", messageID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit SDS retrieval hint save: %w", err)
	}
	return nil
}

func (p *sqliteSDSRetrievalHintPersistence) LoadSDSRetrievalHints(channelID, messageID string) ([][]byte, error) {
	if channelID == "" {
		return nil, fmt.Errorf("load SDS retrieval hints: empty channel ID")
	}
	if messageID == "" {
		return nil, fmt.Errorf("load SDS retrieval hints: empty message ID")
	}
	rows, err := p.db.Query(
		`SELECT envelope_hash FROM sds_retrieval_hints
		WHERE channel_id = ? AND message_id = ? ORDER BY envelope_hash`,
		channelID, messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("load SDS retrieval hints for channel %q message %q: %w", channelID, messageID, err)
	}

	var hashes [][]byte
	for rows.Next() {
		var hash []byte
		if err := rows.Scan(&hash); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan SDS retrieval hint for message %q: %w", messageID, err)
		}
		hashes = append(hashes, hash)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("read SDS retrieval hints for message %q: %w", messageID, err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close SDS retrieval hint query for message %q: %w", messageID, err)
	}
	return hashes, nil
}
