package protocol

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/status-im/status-go/protocol/protobuf"
	"strings"

	"github.com/pkg/errors"
)

var (
	errRecordNotFound = errors.New("record not found")
)

func (db sqlitePersistence) tableUserMessagesLegacyAllFields() string {
	return `id,
    		whisper_timestamp,
    		source,
    		text,
    		content_type,
    		username,
    		timestamp,
    		chat_id,
		local_chat_id,
    		retry_count,
    		message_type,
    		clock_value,
    		seen,
    		outgoing_status,
		parsed_text,
		raw_payload,
		sticker_pack,
		sticker_hash,
		response_to`
}

func (db sqlitePersistence) tableUserMessagesLegacyAllFieldsJoin() string {
	return `m1.id,
    		m1.whisper_timestamp,
    		m1.source,
    		m1.text,
    		m1.content_type,
    		m1.username,
    		m1.timestamp,
    		m1.chat_id,
		m1.local_chat_id,
    		m1.retry_count,
    		m1.message_type,
    		m1.clock_value,
    		m1.seen,
    		m1.outgoing_status,
		m1.parsed_text,
		m1.raw_payload,
		m1.sticker_pack,
		m1.sticker_hash,
		m1.response_to,
		m2.source,
		m2.text,
		c.alias,
		c.identicon`
}

func (db sqlitePersistence) tableUserMessagesLegacyAllFieldsCount() int {
	return strings.Count(db.tableUserMessagesLegacyAllFields(), ",") + 1
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (db sqlitePersistence) tableUserMessagesLegacyScanAllFields(row scanner, message *Message, others ...interface{}) error {
	var quotedText sql.NullString
	var quotedFrom sql.NullString
	var alias sql.NullString
	var identicon sql.NullString

	sticker := &protobuf.StickerMessage{}

	args := []interface{}{
		&message.ID,
		&message.WhisperTimestamp,
		&message.From, // source in table
		&message.Text,
		&message.ContentType,
		&message.Alias,
		&message.Timestamp,
		&message.ChatId,
		&message.LocalChatID,
		&message.RetryCount,
		&message.MessageType,
		&message.Clock,
		&message.Seen,
		&message.OutgoingStatus,
		&message.ParsedText,
		&message.RawPayload,
		&sticker.Pack,
		&sticker.Hash,
		&message.ResponseTo,
		&quotedFrom,
		&quotedText,
		&alias,
		&identicon,
	}
	err := row.Scan(append(args, others...)...)
	if err != nil {
		return err
	}

	if quotedText.Valid {
		message.QuotedMessage = &QuotedMessage{
			From: quotedFrom.String,
			Text: quotedText.String,
		}
	}
	message.Alias = alias.String
	message.Identicon = identicon.String
	if message.ContentType == protobuf.ChatMessage_STICKER {
		message.Payload = &protobuf.ChatMessage_Sticker{Sticker: sticker}
	}

	return nil
}

func (db sqlitePersistence) tableUserMessagesLegacyAllValues(message *Message) ([]interface{}, error) {
	sticker := message.GetSticker()
	if sticker == nil {
		sticker = &protobuf.StickerMessage{}
	}
	return []interface{}{
		message.ID,
		message.WhisperTimestamp,
		message.From, // source in table
		message.Text,
		message.ContentType,
		message.Alias,
		message.Timestamp,
		message.ChatId,
		message.LocalChatID,
		message.RetryCount,
		message.MessageType,
		message.Clock,
		message.Seen,
		message.OutgoingStatus,
		message.ParsedText,
		message.RawPayload,
		sticker.Pack,
		sticker.Hash,
		message.ResponseTo,
	}, nil
}

func (db sqlitePersistence) messageByID(tx *sql.Tx, id string) (*Message, error) {
	var err error
	if tx == nil {
		tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
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
	}

	var message Message

	allFields := db.tableUserMessagesLegacyAllFieldsJoin()
	row := tx.QueryRow(
		fmt.Sprintf(`
			SELECT
				%s
			FROM
				user_messages_legacy m1
			LEFT JOIN
				user_messages_legacy m2
			ON
			m1.response_to = m2.id

			LEFT JOIN
			        contacts c
		        ON
			m1.source = c.id
			WHERE
				m1.id = ?
		`, allFields),
		id,
	)
	err = db.tableUserMessagesLegacyScanAllFields(row, &message)
	switch err {
	case sql.ErrNoRows:
		return nil, errRecordNotFound
	case nil:
		return &message, nil
	default:
		return nil, err
	}
}

func (db sqlitePersistence) MessageByID(id string) (*Message, error) {
	return db.messageByID(nil, id)
}

func (db sqlitePersistence) MessagesExist(ids []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(ids) == 0 {
		return result, nil
	}

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := fmt.Sprintf(`SELECT id FROM user_messages_legacy WHERE id IN (%s)`, inVector)
	rows, err := db.db.Query(query, idsArgs...)
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
		result[id] = true
	}

	return result, nil
}

// MessageByChatID returns all messages for a given chatID in descending order.
// Ordering is accomplished using two concatenated values: ClockValue and ID.
// These two values are also used to compose a cursor which is returned to the result.
func (db sqlitePersistence) MessageByChatID(chatID string, currCursor string, limit int) ([]*Message, string, error) {
	cursorWhere := ""
	if currCursor != "" {
		cursorWhere = "AND cursor <= ?"
	}
	allFields := db.tableUserMessagesLegacyAllFieldsJoin()
	args := []interface{}{chatID}
	if currCursor != "" {
		args = append(args, currCursor)
	}
	// Build a new column `cursor` at the query time by having a fixed-sized clock value at the beginning
	// concatenated with message ID. Results are sorted using this new column.
	// This new column values can also be returned as a cursor for subsequent requests.
	rows, err := db.db.Query(
		fmt.Sprintf(`
			SELECT
				%s,
				substr('0000000000000000000000000000000000000000000000000000000000000000' || m1.clock_value, -64, 64) || m1.id as cursor
			FROM
				user_messages_legacy m1
			LEFT JOIN
				user_messages_legacy m2
			ON
			m1.response_to = m2.id

			LEFT JOIN
			      contacts c
			ON

			m1.source = c.id
			WHERE
				m1.local_chat_id = ? %s
			ORDER BY cursor DESC
			LIMIT ?
		`, allFields, cursorWhere),
		append(args, limit+1)..., // take one more to figure our whether a cursor should be returned
	)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var (
		result  []*Message
		cursors []string
	)
	for rows.Next() {
		var (
			message Message
			cursor  string
		)
		if err := db.tableUserMessagesLegacyScanAllFields(rows, &message, &cursor); err != nil {
			return nil, "", err
		}
		result = append(result, &message)
		cursors = append(cursors, cursor)
	}

	var newCursor string
	if len(result) > limit {
		newCursor = cursors[limit]
		result = result[:limit]
	}
	return result, newCursor, nil
}

func (db sqlitePersistence) SaveMessagesLegacy(messages []*Message) (err error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
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

	allFields := db.tableUserMessagesLegacyAllFields()
	valuesVector := strings.Repeat("?, ", db.tableUserMessagesLegacyAllFieldsCount()-1) + "?"
	query := fmt.Sprintf(`INSERT INTO user_messages_legacy(%s) VALUES (%s)`, allFields, valuesVector)
	stmt, err := tx.Prepare(query)
	if err != nil {
		return
	}

	for _, msg := range messages {
		var allValues []interface{}
		allValues, err = db.tableUserMessagesLegacyAllValues(msg)
		if err != nil {
			return
		}

		_, err = stmt.Exec(allValues...)
		if err != nil {
			return
		}
	}
	return
}

func (db sqlitePersistence) DeleteMessage(id string) error {
	_, err := db.db.Exec(`DELETE FROM user_messages_legacy WHERE id = ?`, id)
	return err
}

func (db sqlitePersistence) DeleteMessagesByChatID(id string) error {
	_, err := db.db.Exec(`DELETE FROM user_messages_legacy WHERE local_chat_id = ?`, id)
	return err
}

func (db sqlitePersistence) MarkMessagesSeen(chatID string, ids []string) error {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
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

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	_, err = tx.Exec(
		fmt.Sprintf(`
			UPDATE user_messages_legacy
			SET seen = 1
			WHERE id IN (%s)
		`, inVector),
		idsArgs...)
	if err != nil {
		return err
	}

	// Update denormalized count
	_, err = tx.Exec(
		`UPDATE chats
              	SET unviewed_message_count =
		   (SELECT COUNT(1)
		   FROM user_messages_legacy
		   WHERE local_chat_id = ? AND seen = 0)
		WHERE id = ?`, chatID, chatID)
	return err
}

func (db sqlitePersistence) UpdateMessageOutgoingStatus(id string, newOutgoingStatus string) error {
	_, err := db.db.Exec(`
		UPDATE user_messages_legacy
		SET outgoing_status = ?
		WHERE id = ?
	`, newOutgoingStatus, id)
	return err
}

// BlockContact updates a contact, deletes all the messages and 1-to-1 chat, updates the unread messages count and returns a map with the new count
func (db sqlitePersistence) BlockContact(contact Contact) ([]*Chat, error) {
	var chats []*Chat
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
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

	// Delete messages
	_, err = tx.Exec(
		`DELETE
		 FROM user_messages_legacy
		 WHERE source = ?`,
		contact.ID,
	)
	if err != nil {
		return nil, err
	}

	// Update contact
	err = db.SaveContact(contact, tx)
	if err != nil {
		return nil, err
	}

	// Delete one-to-one chat
	_, err = tx.Exec("DELETE FROM chats WHERE id = ?", contact.ID)
	if err != nil {
		return nil, err
	}

	// Recalculate denormalized fields
	_, err = tx.Exec(`
		UPDATE chats
		SET
			unviewed_message_count = (SELECT COUNT(1) FROM user_messages_legacy WHERE seen = 0 AND local_chat_id = chats.id)`)
	if err != nil {
		return nil, err
	}

	// return the updated chats
	chats, err = db.chats(tx)
	if err != nil {
		return nil, err
	}
	for _, c := range chats {
		var lastMessageID string
		row := tx.QueryRow(`SELECT id FROM user_messages_legacy WHERE local_chat_id = ? ORDER BY clock_value DESC LIMIT 1`, c.ID)
		switch err := row.Scan(&lastMessageID); err {

		case nil:
			message, err := db.messageByID(tx, lastMessageID)
			if err != nil {
				return nil, err
			}
			if message != nil {
				encodedMessage, err := json.Marshal(message)
				if err != nil {
					return nil, err
				}
				_, err = tx.Exec(`UPDATE chats SET last_message = ? WHERE id = ?`, encodedMessage, c.ID)
				if err != nil {
					return nil, err
				}
				c.LastMessage = encodedMessage

			}

		case sql.ErrNoRows:
			// Reset LastMessage
			_, err = tx.Exec(`UPDATE chats SET last_message = NULL WHERE id = ?`, c.ID)
			if err != nil {
				return nil, err
			}
			c.LastMessage = nil
		default:
			return nil, err
		}
	}

	return chats, err
}
