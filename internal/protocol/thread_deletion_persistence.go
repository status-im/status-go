package protocol

import (
	"database/sql"
	"errors"
)

// softDeleteThread keeps the root and all stored content, while making deletion
// terminal. It runs in the same transaction as reply and notification updates.
func (db sqlitePersistence) softDeleteThread(tx *sql.Tx, chatID, threadID string, clock uint64, actor string, timestamp ...uint64) error {
	var at uint64
	if len(timestamp) > 0 {
		at = timestamp[0]
	}
	_, err := tx.Exec(`UPDATE threads SET deleted = TRUE,
 deleted_at = CASE WHEN NOT deleted OR deleted_clock < ? OR (deleted_clock = ? AND deleted_by < ?) THEN ? ELSE deleted_at END,
 deleted_by = CASE WHEN NOT deleted OR deleted_clock < ? OR (deleted_clock = ? AND deleted_by < ?) THEN ? ELSE deleted_by END,
 deleted_clock = MAX(deleted_clock, ?)
 WHERE chat_id = ? AND thread_id = ?`, clock, clock, actor, at, clock, clock, actor, actor, clock, chatID, threadID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE user_messages SET deleted = TRUE, seen = TRUE,
 deleted_by = (SELECT deleted_by FROM threads WHERE chat_id = ? AND thread_id = ?)
 WHERE local_chat_id = ? AND thread_id = ?`, chatID, threadID, chatID, threadID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE activity_center_notifications SET deleted = TRUE, read = TRUE, updated_at = MAX(updated_at, ?)
 WHERE chat_id = ? AND json_extract(message, '$.threadId') = ?`, clock, chatID, threadID)
	return err
}

func (db sqlitePersistence) SoftDeleteThread(chatID, threadID string, clock uint64, actor string, timestamp ...uint64) (err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	return db.softDeleteThread(tx, chatID, threadID, clock, actor, timestamp...)
}

// A deleted parent is authoritative even when its thread metadata arrives later.
// Reuse the stored actor; the authenticated messenger path refines the deletion
// clock and timestamp when those are available.
func (db sqlitePersistence) cascadeThreadFromDeletedParent(tx *sql.Tx, chatID, parentID string) error {
	var threadID, actor string
	var clock uint64
	err := tx.QueryRow(`SELECT t.thread_id, COALESCE(NULLIF(m.deleted_by, ''), m.source), m.clock_value
 FROM threads t JOIN user_messages m ON m.id = t.parent_message_id AND m.local_chat_id = t.chat_id
 WHERE t.chat_id = ? AND t.parent_message_id = ? AND m.deleted AND NOT t.deleted`, chatID, parentID).Scan(&threadID, &actor, &clock)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return db.softDeleteThread(tx, chatID, threadID, clock, actor)
}
