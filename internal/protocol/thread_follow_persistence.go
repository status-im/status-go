package protocol

import (
	"database/sql"
)

// SetThreadFollowed records an explicit follow choice. The newest clock wins;
// it returns whether the stored state was updated.
func (db sqlitePersistence) SetThreadFollowed(chatID, threadID string, followed bool, clock uint64) (bool, error) {
	result, err := db.db.Exec(`UPDATE threads SET followed = ?, followed_clock = ?
		WHERE thread_id = ? AND chat_id = ? AND followed_clock < ?`,
		followed, clock, threadID, chatID, clock)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// AutoFollowOwnedThread follows a thread created by ownID or started on one of
// ownID's messages, unless the user has already made an explicit choice.
func (db sqlitePersistence) AutoFollowOwnedThread(chatID, threadID, ownID string) (bool, error) {
	result, err := db.db.Exec(`UPDATE threads SET followed = TRUE
		WHERE thread_id = ? AND chat_id = ? AND followed_clock = 0 AND NOT followed AND NOT deleted
			AND (creator_id = ? OR EXISTS (
				SELECT 1 FROM user_messages
				WHERE user_messages.id = threads.parent_message_id
					AND user_messages.local_chat_id = threads.chat_id
					AND user_messages.source = ?))`,
		threadID, chatID, ownID, ownID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// mergeThreadFollowed applies backed up follow state. Explicit choices merge by
// clock; automatic follows (clock 0) only apply while no explicit choice exists.
func (db sqlitePersistence) mergeThreadFollowed(tx *sql.Tx, chatID, threadID string, followed bool, clock uint64) error {
	if clock == 0 {
		_, err := tx.Exec(`UPDATE threads SET followed = followed OR ?
			WHERE thread_id = ? AND chat_id = ? AND followed_clock = 0`, followed, threadID, chatID)
		return err
	}
	_, err := tx.Exec(`UPDATE threads SET followed = ?, followed_clock = ?
		WHERE thread_id = ? AND chat_id = ? AND followed_clock < ?`,
		followed, clock, threadID, chatID, clock)
	return err
}
