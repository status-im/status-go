package protocol

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/status-im/status-go/internal/protocol/common"
)

func (db sqlitePersistence) MuteThread(chatID, threadID string, till time.Time) error {
	_, err := db.setThreadMuted(chatID, threadID, true, till.Format(time.RFC3339))
	return err
}

func (db sqlitePersistence) UnmuteThread(chatID, threadID string) error {
	_, err := db.setThreadMuted(chatID, threadID, false, 0)
	return err
}

// Write and response metadata belong to one transaction. A read failure must not
// leave a committed preference behind an RPC error.
func (db sqlitePersistence) setThreadMuted(chatID, threadID string, muted bool, till interface{}) (*Thread, error) {
	tx, err := db.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck
	result, err := tx.Exec(`UPDATE threads SET muted = ?, muted_till = ? WHERE chat_id = ? AND thread_id = ?`, muted, till, chatID, threadID)
	if err != nil {
		return nil, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, common.ErrRecordNotFound
	}
	thread, err := db.threadByID(tx, chatID, threadID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return thread, nil
}

// Notification checks need only the preference, without scanning message counts.
func (db sqlitePersistence) IsThreadMuted(chatID, threadID string) (bool, error) {
	var muted bool
	err := db.db.QueryRow(`SELECT muted FROM threads WHERE chat_id = ? AND thread_id = ?`, chatID, threadID).Scan(&muted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return muted, err
}

// Test the current deadline in the UPDATE itself, so a renewed mute cannot be
// cleared using an earlier read. Read response metadata before releasing the writer.
func (db sqlitePersistence) unmuteExpiredThreads(ctx context.Context, now time.Time) ([]*Thread, error) {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck
	rows, err := tx.QueryContext(ctx, `UPDATE threads SET muted = 0, muted_till = 0
  WHERE muted = 1 AND muted_till != ? AND julianday(muted_till) <= julianday(?)
  RETURNING chat_id, thread_id`, time.Time{}.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	var ids []threadIdentity
	for rows.Next() {
		var id threadIdentity
		if err := rows.Scan(&id.chatID, &id.threadID); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	readErr := rows.Err()
	closeErr := rows.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	// Release RETURNING's cursor before issuing another query on this transaction.
	threads, err := db.threadsByIDs(tx, ids)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return threads, nil
}
