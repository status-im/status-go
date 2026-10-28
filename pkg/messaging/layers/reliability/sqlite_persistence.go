package reliability

import (
	"database/sql"
	"fmt"

	"github.com/waku-org/sds-go-bindings/sds"
)

func NewSQLitePersistence(db *sql.DB) sds.PersistenceCallbacks {
	return sds.PersistenceCallbacks{
		SaveChannelMeta: func(channelID string, meta []byte) error {
			_, err := db.Exec(
				`INSERT INTO sds_channel_metadata (channel_id, metadata)
				VALUES (?, ?)
				ON CONFLICT(channel_id) DO UPDATE SET metadata = excluded.metadata`,
				channelID, meta,
			)
			if err != nil {
				return fmt.Errorf("save SDS channel metadata: %w", err)
			}
			return nil
		},
		UpdateHistory: func(channelID string, update sds.HistoryUpdate) error {
			tx, err := db.Begin()
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
		},
		LoadChannel: func(channelID string) (sds.ChannelData, error) {
			data := sds.ChannelData{
				Meta:           []byte{},
				MessageHistory: make([][]byte, 0),
			}
			err := db.QueryRow(
				`SELECT metadata FROM sds_channel_metadata WHERE channel_id = ?`,
				channelID,
			).Scan(&data.Meta)
			if err != nil && err != sql.ErrNoRows {
				return sds.ChannelData{}, fmt.Errorf("load SDS channel metadata: %w", err)
			}

			rows, err := db.Query(
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
		},
		DropChannel: func(channelID string) error {
			tx, err := db.Begin()
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
		},
	}
}
