package statusproto

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

var (
	errRecordNotFound = errors.New("record not found")
)

func (db sqlitePersistence) tableUserMessagesLegacyAllFields() string {
	return `id,
    		raw_payload_hash,
    		whisper_timestamp,
    		source,
    		destination,
    		content,
    		content_type,
    		username,
    		timestamp,
    		chat_id,
    		retry_count,
    		message_type,
    		message_status,
    		clock_value,
    		show,
    		seen,
    		outgoing_status`
}

func (db sqlitePersistence) tableUserMessagesLegacyAllFieldsCount() int {
	return strings.Count(db.tableUserMessagesLegacyAllFields(), ",") + 1
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (db sqlitePersistence) tableUserMessagesLegacyScanAllFields(row scanner, message *Message, others ...interface{}) error {
	args := []interface{}{
		&message.ID,
		&message.RawPayloadHash,
		&message.WhisperTimestamp,
		&message.From, // source in table
		&message.To,   // destination in table
		&message.Content,
		&message.ContentType,
		&message.Username,
		&message.Timestamp,
		&message.ChatID,
		&message.RetryCount,
		&message.MessageType,
		&message.MessageStatus,
		&message.ClockValue,
		&message.Show,
		&message.Seen,
		&message.OutgoingStatus,
	}
	return row.Scan(append(args, others...)...)
}

func (db sqlitePersistence) tableUserMessagesLegacyAllValues(message *Message) []interface{} {
	return []interface{}{
		message.ID,
		message.RawPayloadHash,
		message.WhisperTimestamp,
		message.From, // source in table
		message.To,   // destination in table
		message.Content,
		message.ContentType,
		message.Username,
		message.Timestamp,
		message.ChatID,
		message.RetryCount,
		message.MessageType,
		message.MessageStatus,
		message.ClockValue,
		message.Show,
		message.Seen,
		message.OutgoingStatus,
	}
}

func (db sqlitePersistence) MessageByID(id string) (*Message, error) {
	var message Message

	allFields := db.tableUserMessagesLegacyAllFields()
	row := db.db.QueryRow(
		fmt.Sprintf(`
			SELECT
				%s
			FROM
				user_messages_legacy
			WHERE
				id = ?
		`, allFields),
		id,
	)
	err := db.tableUserMessagesLegacyScanAllFields(row, &message)
	switch err {
	case sql.ErrNoRows:
		return nil, errRecordNotFound
	case nil:
		return &message, nil
	default:
		return nil, err
	}
}

func (db sqlitePersistence) MessageExists(id string) (bool, error) {
	var result bool
	err := db.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM user_messages_legacy WHERE id = ?)`, id).Scan(&result)
	switch err {
	case sql.ErrNoRows:
		return false, errRecordNotFound
	case nil:
		return result, nil
	default:
		return false, err
	}
}

// MessageByChatID returns all messages for a given chatID in descending order.
// Ordering is accomplished using two concatenated values: ClockValue and ID.
// These two values are also used to compose a cursor which is returned to the result.
func (db sqlitePersistence) MessageByChatID(chatID string, currCursor string, limit int) ([]*Message, string, error) {
	cursorWhere := ""
	if currCursor != "" {
		cursorWhere = "AND cursor <= ?"
	}
	allFields := db.tableUserMessagesLegacyAllFields()
	args := []interface{}{chatID}
	if currCursor != "" {
		args = append(args, currCursor)
	}
	// Build a new column `cursor` at the query time by having a fixed-sized clock value at the beginning
	// concatenated with rowid. Results are sorted using this new column.
	// This new column values can also be returned as a cursor for subsequent requests.
	rows, err := db.db.Query(
		fmt.Sprintf(`
			SELECT
				%s,
				substr('0000000000000000000000000000000000000000000000000000000000000000' || clock_value, -64, 64) || id as cursor
			FROM
				user_messages_legacy
			WHERE
				chat_id = ? %s
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

func (db sqlitePersistence) MessagesFrom(from []byte) ([]*Message, error) {
	allFields := db.tableUserMessagesLegacyAllFields()
	rows, err := db.db.Query(
		fmt.Sprintf(`
			SELECT
				%s
			FROM
				user_messages_legacy
			WHERE
				source = ?
		`, allFields),
		from,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Message
	for rows.Next() {
		var message Message
		if err := db.tableUserMessagesLegacyScanAllFields(rows, &message); err != nil {
			return nil, err
		}
		result = append(result, &message)
	}
	return result, nil
}

func (db sqlitePersistence) UnseenMessageIDs() ([][]byte, error) {
	rows, err := db.db.Query(`SELECT id FROM user_messages_legacy WHERE seen = 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result [][]byte
	for rows.Next() {
		var id []byte
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

func (db sqlitePersistence) SaveMessage(m *Message) error {
	allFields := db.tableUserMessagesLegacyAllFields()
	valuesVector := strings.Repeat("?, ", db.tableUserMessagesLegacyAllFieldsCount()-1) + "?"
	query := fmt.Sprintf(`INSERT INTO user_messages_legacy(%s) VALUES (%s)`, allFields, valuesVector)
	_, err := db.db.Exec(
		query,
		db.tableUserMessagesLegacyAllValues(m)...,
	)
	return err
}

func (db sqlitePersistence) DeleteMessage(id string) error {
	_, err := db.db.Exec(`DELETE FROM user_messages_legacy WHERE id = ?`, id)
	return err
}

func (db sqlitePersistence) MarkMessagesSeen(ids ...string) error {
	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	_, err := db.db.Exec(
		fmt.Sprintf(`
			UPDATE user_messages_legacy
			SET seen = 1
			WHERE id IN (%s)
		`, inVector),
		idsArgs...)
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
