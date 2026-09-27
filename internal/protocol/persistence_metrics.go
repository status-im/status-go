package protocol

import (
	"database/sql"
	"fmt"

	"github.com/status-im/status-go/internal/db/sqlutil"
)

const selectTimestampsQuery = "SELECT whisper_timestamp FROM user_messages WHERE %s whisper_timestamp >= ? AND whisper_timestamp <= ?"
const selectCountQuery = "SELECT COUNT(*) FROM user_messages WHERE %s whisper_timestamp >= ? AND whisper_timestamp <= ?"

func chatsPeriodQuery(query string, chatIDs []string, startTimestamp uint64, endTimestamp uint64) (string, []interface{}) {
	args := make([]interface{}, 0, len(chatIDs)+2)
	chatsFilter := ""
	if len(chatIDs) > 0 {
		chatsFilter = sqlutil.In("local_chat_id IN (%s) AND", len(chatIDs))
		for _, chatID := range chatIDs {
			args = append(args, chatID)
		}
	}
	args = append(args, startTimestamp, endTimestamp)
	return fmt.Sprintf(query, chatsFilter), args
}

func (db sqlitePersistence) SelectMessagesTimestampsForChatsByPeriod(chatIDs []string, startTimestamp uint64, endTimestamp uint64) ([]uint64, error) {
	query, args := chatsPeriodQuery(selectTimestampsQuery, chatIDs, startTimestamp, endTimestamp)

	rows, err := db.db.Query(query, args...)
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

	err = rows.Err()
	if err != nil {
		return []uint64{}, err
	}

	return timestamps, nil
}

func (db sqlitePersistence) SelectMessagesCountForChatsByPeriod(chatIDs []string, startTimestamp uint64, endTimestamp uint64) (int, error) {
	query, args := chatsPeriodQuery(selectCountQuery, chatIDs, startTimestamp, endTimestamp)

	var count int
	if err := db.db.QueryRow(query, args...).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}
