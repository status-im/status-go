package protocol

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

var msgNotHidden = "NOT(m1.hide) AND NOT(m1.deleted) AND NOT(m1.deleted_for_me)"

func (db sqlitePersistence) executeTx(tx *sql.Tx, fn func(tx *sql.Tx) error) (err error) {
	if tx == nil {
		tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				// don't shadow original error
				_ = tx.Rollback()
			} else {
				err = tx.Commit()
			}
		}()
	}

	return fn(tx)
}

func (db sqlitePersistence) updateAllChatsUnviewedCounts(tx *sql.Tx) error {
	return db.executeTx(tx, func(tx *sql.Tx) error {
		// Recalculate denormalized fields
		_, err := tx.Exec(
			fmt.Sprintf(`
		UPDATE chats
		SET
			unviewed_message_count =
				(SELECT COUNT(1)
				FROM user_messages m1
				WHERE local_chat_id = chats.id
				AND NOT(seen) AND %s),
			unviewed_mentions_count =
				(SELECT COUNT(1)
				FROM user_messages m1
				WHERE local_chat_id = chats.id
				AND NOT(seen) AND %s AND (mentioned OR replied)),
			first_unviewed_message_id =
				(SELECT id
				FROM user_messages m1
				WHERE local_chat_id = chats.id
				AND NOT(seen) AND %s
				ORDER BY clock_value ASC
				LIMIT 1),
			highlight = 0`, msgNotHidden, msgNotHidden, msgNotHidden))

		return err
	})
}

func (db sqlitePersistence) updateChatUnviewedCounts(chatID string, tx *sql.Tx) error {
	return db.executeTx(tx, func(tx *sql.Tx) error {
		// Recalculate denormalized fields
		_, err := tx.Exec(
			fmt.Sprintf(`
		UPDATE chats
		SET
			unviewed_message_count =
				(SELECT COUNT(1)
				FROM user_messages m1
				WHERE local_chat_id = ?
				AND NOT(seen) AND %s),
			unviewed_mentions_count =
				(SELECT COUNT(1)
				FROM user_messages m1
				WHERE local_chat_id = ?
				AND NOT(seen) AND %s AND (mentioned OR replied)),
			first_unviewed_message_id =
				(SELECT id
				FROM user_messages m1
				WHERE local_chat_id = ?
				AND NOT(seen) AND %s
				ORDER BY clock_value ASC
				LIMIT 1),
			highlight = 0
		WHERE id = ?`, msgNotHidden, msgNotHidden, msgNotHidden),
			chatID, chatID, chatID, chatID)

		return err
	})
}

func (db sqlitePersistence) clearChatsUnviewedCounts(chatIDs []string, tx *sql.Tx) error {
	return db.executeTx(tx, func(tx *sql.Tx) error {
		idsArgs := make([]interface{}, 0, len(chatIDs))
		for _, id := range chatIDs {
			idsArgs = append(idsArgs, id)
		}

		inVector := strings.Repeat("?, ", len(chatIDs)-1) + "?"

		_, err := tx.Exec(
			fmt.Sprintf(`
		UPDATE chats
		SET
			unviewed_message_count = 0,
			unviewed_mentions_count = 0,
			first_unviewed_message_id = NULL,
			highlight = 0
		WHERE id IN (%s)`, inVector),
			idsArgs...)

		return err
	})
}

func (db sqlitePersistence) getChatUnviewedCounts(chatID string, tx *sql.Tx) (unviewedMessages, unviewedMentions uint, firstUnviewedMessageID string, err error) {
	err = db.executeTx(tx, func(tx *sql.Tx) error {
		var sqlFirstUnviewedMessageID sql.NullString

		err := tx.QueryRow(`
		SELECT unviewed_message_count, unviewed_mentions_count, first_unviewed_message_id
		FROM chats
		WHERE id = ?`, chatID).Scan(&unviewedMessages, &unviewedMentions, &sqlFirstUnviewedMessageID)
		if err != nil {
			return err
		}

		if sqlFirstUnviewedMessageID.Valid {
			firstUnviewedMessageID = sqlFirstUnviewedMessageID.String
		}

		return nil
	})
	return
}
