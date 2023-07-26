package protocol

import (
	"context"
	"database/sql"
)

func (db sqlitePersistence) fetchMessagesTimestampsForPeriod(tx *sql.Tx, chatID string, startTimestamp uint64, endTimestamp uint64) ([]uint64, error) {
	rows, err := tx.Query(`
		SELECT whisper_timestamp FROM user_messages 
		WHERE local_chat_id = ? AND 
		whisper_timestamp >= ? AND 
		whisper_timestamp <= ?`,
		chatID,
		startTimestamp,
		endTimestamp,
	)
	if err != nil {
		return []uint64{}, err
	}
	defer rows.Close()

	var timestamps []uint64
	for rows.Next() {
		var timestamp uint64
		err := rows.Scan(&timestamp)
		if err != nil {
			return nil, err
		}
		timestamps = append(timestamps, timestamp)
	}

	return timestamps, nil
}

func (db sqlitePersistence) FetchMessageTimestampsForChatByPeriod(chatID string, startTimestamp uint64, endTimestamp uint64) ([]uint64, error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return []uint64{}, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	return db.fetchMessagesTimestampsForPeriod(tx, chatID, startTimestamp, endTimestamp)
}

func (db sqlitePersistence) FetchMessageTimestampsForChatsByPeriod(chatIDs []string, startTimestamp uint64, endTimestamp uint64) ([]uint64, error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return []uint64{}, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	var timestamps []uint64
	for _, chatID := range chatIDs {
		chatTimestamps, err := db.fetchMessagesTimestampsForPeriod(tx, chatID, startTimestamp, endTimestamp)
		if err != nil {
			return []uint64{}, err
		}
		timestamps = append(timestamps, chatTimestamps...)
	}
	return timestamps, nil
}
