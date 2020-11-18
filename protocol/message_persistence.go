package protocol

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (db sqlitePersistence) tableUserMessagesAllFields() string {
	return `id,
    		whisper_timestamp,
    		source,
    		text,
    		content_type,
    		username,
    		timestamp,
    		chat_id,
		local_chat_id,
    		message_type,
    		clock_value,
    		seen,
    		outgoing_status,
		parsed_text,
		sticker_pack,
		sticker_hash,
		image_payload,
		image_type,
		image_base64,
		audio_payload,
		audio_type,
		audio_duration_ms,
		audio_base64,
		community_id,
		mentions,
		links,
		command_id,
		command_value,
		command_from,
		command_address,
		command_contract,
		command_transaction_hash,
		command_state,
		command_signature,
		replace_message,
		rtl,
		line_count,
		response_to`
}

func (db sqlitePersistence) tableUserMessagesAllFieldsJoin() string {
	return `m1.id,
    		m1.whisper_timestamp,
    		m1.source,
    		m1.text,
    		m1.content_type,
    		m1.username,
    		m1.timestamp,
    		m1.chat_id,
		m1.local_chat_id,
    		m1.message_type,
    		m1.clock_value,
    		m1.seen,
    		m1.outgoing_status,
		m1.parsed_text,
		m1.sticker_pack,
		m1.sticker_hash,
		m1.image_base64,
		COALESCE(m1.audio_duration_ms,0),
		m1.audio_base64,
		m1.community_id,
		m1.mentions,
		m1.links,
		m1.command_id,
		m1.command_value,
		m1.command_from,
		m1.command_address,
		m1.command_contract,
		m1.command_transaction_hash,
		m1.command_state,
		m1.command_signature,
		m1.replace_message,
		m1.rtl,
		m1.line_count,
		m1.response_to,
		m2.source,
		m2.text,
		m2.parsed_text,
		m2.image_base64,
		m2.audio_duration_ms,
		m2.audio_base64,
		m2.community_id,
		c.alias,
		c.identicon`
}

func (db sqlitePersistence) tableUserMessagesAllFieldsCount() int {
	return strings.Count(db.tableUserMessagesAllFields(), ",") + 1
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (db sqlitePersistence) tableUserMessagesScanAllFields(row scanner, message *common.Message, others ...interface{}) error {
	var quotedText sql.NullString
	var quotedParsedText []byte
	var quotedFrom sql.NullString
	var quotedImage sql.NullString
	var quotedAudio sql.NullString
	var quotedAudioDuration sql.NullInt64
	var quotedCommunityID sql.NullString
	var serializedMentions []byte
	var serializedLinks []byte
	var alias sql.NullString
	var identicon sql.NullString

	sticker := &protobuf.StickerMessage{}
	command := &common.CommandParameters{}
	audio := &protobuf.AudioMessage{}

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
		&message.MessageType,
		&message.Clock,
		&message.Seen,
		&message.OutgoingStatus,
		&message.ParsedText,
		&sticker.Pack,
		&sticker.Hash,
		&message.Base64Image,
		&audio.DurationMs,
		&message.Base64Audio,
		&message.CommunityID,
		&serializedMentions,
		&serializedLinks,
		&command.ID,
		&command.Value,
		&command.From,
		&command.Address,
		&command.Contract,
		&command.TransactionHash,
		&command.CommandState,
		&command.Signature,
		&message.Replace,
		&message.RTL,
		&message.LineCount,
		&message.ResponseTo,
		&quotedFrom,
		&quotedText,
		&quotedParsedText,
		&quotedImage,
		&quotedAudioDuration,
		&quotedAudio,
		&quotedCommunityID,
		&alias,
		&identicon,
	}
	err := row.Scan(append(args, others...)...)
	if err != nil {
		return err
	}

	if quotedText.Valid {
		message.QuotedMessage = &common.QuotedMessage{
			From:            quotedFrom.String,
			Text:            quotedText.String,
			ParsedText:      quotedParsedText,
			Base64Image:     quotedImage.String,
			AudioDurationMs: uint64(quotedAudioDuration.Int64),
			Base64Audio:     quotedAudio.String,
			CommunityID:     quotedCommunityID.String,
		}
	}
	message.Alias = alias.String
	message.Identicon = identicon.String

	if serializedMentions != nil {
		err := json.Unmarshal(serializedMentions, &message.Mentions)
		if err != nil {
			return err
		}
	}

	if serializedLinks != nil {
		err := json.Unmarshal(serializedLinks, &message.Links)
		if err != nil {
			return err
		}
	}

	switch message.ContentType {
	case protobuf.ChatMessage_STICKER:
		message.Payload = &protobuf.ChatMessage_Sticker{Sticker: sticker}

	case protobuf.ChatMessage_AUDIO:
		message.Payload = &protobuf.ChatMessage_Audio{Audio: audio}

	case protobuf.ChatMessage_TRANSACTION_COMMAND:
		message.CommandParameters = command
	}

	return nil
}

func (db sqlitePersistence) tableUserMessagesAllValues(message *common.Message) ([]interface{}, error) {
	sticker := message.GetSticker()
	if sticker == nil {
		sticker = &protobuf.StickerMessage{}
	}

	image := message.GetImage()
	if image == nil {
		image = &protobuf.ImageMessage{}
	}

	audio := message.GetAudio()
	if audio == nil {
		audio = &protobuf.AudioMessage{}
	}

	command := message.CommandParameters
	if command == nil {
		command = &common.CommandParameters{}
	}

	var serializedMentions []byte
	var err error
	if len(message.Mentions) != 0 {
		serializedMentions, err = json.Marshal(message.Mentions)
		if err != nil {
			return nil, err
		}
	}

	var serializedLinks []byte
	if len(message.Links) != 0 {
		serializedLinks, err = json.Marshal(message.Links)
		if err != nil {
			return nil, err
		}
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
		message.MessageType,
		message.Clock,
		message.Seen,
		message.OutgoingStatus,
		message.ParsedText,
		sticker.Pack,
		sticker.Hash,
		image.Payload,
		image.Type,
		message.Base64Image,
		audio.Payload,
		audio.Type,
		audio.DurationMs,
		message.Base64Audio,
		message.CommunityID,
		serializedMentions,
		serializedLinks,
		command.ID,
		command.Value,
		command.From,
		command.Address,
		command.Contract,
		command.TransactionHash,
		command.CommandState,
		command.Signature,
		message.Replace,
		message.RTL,
		message.LineCount,
		message.ResponseTo,
	}, nil
}

func (db sqlitePersistence) messageByID(tx *sql.Tx, id string) (*common.Message, error) {
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

	var message common.Message

	allFields := db.tableUserMessagesAllFieldsJoin()
	row := tx.QueryRow(
		fmt.Sprintf(`
			SELECT
				%s
			FROM
				user_messages m1
			LEFT JOIN
				user_messages m2
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
	err = db.tableUserMessagesScanAllFields(row, &message)
	switch err {
	case sql.ErrNoRows:
		return nil, common.ErrRecordNotFound
	case nil:
		return &message, nil
	default:
		return nil, err
	}
}

func (db sqlitePersistence) MessageByCommandID(chatID, id string) (*common.Message, error) {

	var message common.Message

	allFields := db.tableUserMessagesAllFieldsJoin()
	row := db.db.QueryRow(
		fmt.Sprintf(`
			SELECT
				%s
			FROM
				user_messages m1
			LEFT JOIN
				user_messages m2
			ON
			m1.response_to = m2.id

			LEFT JOIN
			        contacts c
		        ON
			m1.source = c.id
			WHERE
				m1.command_id = ?
				AND
				m1.local_chat_id = ?
				ORDER BY m1.clock_value DESC
				LIMIT 1
		`, allFields),
		id,
		chatID,
	)
	err := db.tableUserMessagesScanAllFields(row, &message)
	switch err {
	case sql.ErrNoRows:
		return nil, common.ErrRecordNotFound
	case nil:
		return &message, nil
	default:
		return nil, err
	}
}

func (db sqlitePersistence) MessageByID(id string) (*common.Message, error) {
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
	query := "SELECT id FROM user_messages WHERE id IN (" + inVector + ")" // nolint: gosec
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

func (db sqlitePersistence) MessagesByIDs(ids []string) ([]*common.Message, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	allFields := db.tableUserMessagesAllFieldsJoin()
	inVector := strings.Repeat("?, ", len(ids)-1) + "?"

	// nolint: gosec
	rows, err := db.db.Query(fmt.Sprintf(`
			SELECT
				%s
			FROM
				user_messages m1
			LEFT JOIN
				user_messages m2
			ON
			m1.response_to = m2.id

			LEFT JOIN
			      contacts c
			ON

			m1.source = c.id
			WHERE NOT(m1.hide) AND m1.id IN (%s)`, allFields, inVector), idsArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*common.Message
	for rows.Next() {
		var message common.Message
		if err := db.tableUserMessagesScanAllFields(rows, &message); err != nil {
			return nil, err
		}
		result = append(result, &message)
	}

	return result, nil
}

// MessageByChatID returns all messages for a given chatID in descending order.
// Ordering is accomplished using two concatenated values: ClockValue and ID.
// These two values are also used to compose a cursor which is returned to the result.
func (db sqlitePersistence) MessageByChatID(chatID string, currCursor string, limit int) ([]*common.Message, string, error) {
	cursorWhere := ""
	if currCursor != "" {
		cursorWhere = "AND cursor <= ?" //nolint: goconst
	}
	allFields := db.tableUserMessagesAllFieldsJoin()
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
				user_messages m1
			LEFT JOIN
				user_messages m2
			ON
			m1.response_to = m2.id

			LEFT JOIN
			      contacts c
			ON

			m1.source = c.id
			WHERE
				NOT(m1.hide) AND m1.local_chat_id = ? %s
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
		result  []*common.Message
		cursors []string
	)
	for rows.Next() {
		var (
			message common.Message
			cursor  string
		)
		if err := db.tableUserMessagesScanAllFields(rows, &message, &cursor); err != nil {
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

// MessageByChatIDs returns all messages for a given chatIDs in descending order.
// Ordering is accomplished using two concatenated values: ClockValue and ID.
// These two values are also used to compose a cursor which is returned to the result.
func (db sqlitePersistence) MessageByChatIDs(chatIDs []string, currCursor string, limit int) ([]*common.Message, string, error) {
	cursorWhere := ""
	if currCursor != "" {
		cursorWhere = "AND cursor <= ?" //nolint: goconst
	}
	allFields := db.tableUserMessagesAllFieldsJoin()
	args := make([]interface{}, len(chatIDs))
	for i, v := range chatIDs {
		args[i] = v
	}
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
				user_messages m1
			LEFT JOIN
				user_messages m2
			ON
			m1.response_to = m2.id

			LEFT JOIN
			      contacts c
			ON

			m1.source = c.id
			WHERE
				NOT(m1.hide) AND m1.local_chat_id IN %s %s
			ORDER BY cursor DESC
			LIMIT ?
		`, allFields, "(?"+strings.Repeat(",?", len(chatIDs)-1)+")", cursorWhere),
		append(args, limit+1)..., // take one more to figure our whether a cursor should be returned
	)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var (
		result  []*common.Message
		cursors []string
	)
	for rows.Next() {
		var (
			message common.Message
			cursor  string
		)
		if err := db.tableUserMessagesScanAllFields(rows, &message, &cursor); err != nil {
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

// EmojiReactionsByChatID returns the emoji reactions for the queried messages, up to a maximum of 100, as it's a potentially unbound number.
// NOTE: This is not completely accurate, as the messages in the database might have change since the last call to `MessageByChatID`.
func (db sqlitePersistence) EmojiReactionsByChatID(chatID string, currCursor string, limit int) ([]*EmojiReaction, error) {
	cursorWhere := ""
	if currCursor != "" {
		cursorWhere = "AND substr('0000000000000000000000000000000000000000000000000000000000000000' || m.clock_value, -64, 64) || m.id <= ?" //nolint: goconst

	}
	args := []interface{}{chatID, chatID}
	if currCursor != "" {
		args = append(args, currCursor)
	}
	args = append(args, limit)
	// NOTE: We match against local_chat_id for security reasons.
	// As a user could potentially send an emoji reaction for a one to
	// one/group chat that has no access to.
	// We also limit the number of emoji to a reasonable number (1000)
	// for now, as we don't want the client to choke on this.
	// The issue is that your own emoji might not be returned in such cases,
	// allowing the user to react to a post multiple times.
	// Jakubgs: Returning the whole list seems like a real overkill.
	// This will get very heavy in threads that have loads of reactions on loads of messages.
	// A more sensible response would just include a count and a bool telling you if you are in the list.
	// nolint: gosec
	query := fmt.Sprintf(`
			SELECT
			    e.clock_value,
			    e.source,
			    e.emoji_id,
			    e.message_id,
			    e.chat_id,
			    e.local_chat_id,
			    e.retracted
			FROM
				emoji_reactions e
			WHERE NOT(e.retracted)
			AND
			e.local_chat_id = ?
			AND
			e.message_id IN
			(SELECT id FROM user_messages m WHERE NOT(m.hide) AND m.local_chat_id = ? %s
			ORDER BY substr('0000000000000000000000000000000000000000000000000000000000000000' || m.clock_value, -64, 64) || m.id DESC LIMIT ?)
			LIMIT 1000
		`, cursorWhere)

	rows, err := db.db.Query(
		query,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*EmojiReaction
	for rows.Next() {
		var emojiReaction EmojiReaction
		err := rows.Scan(&emojiReaction.Clock,
			&emojiReaction.From,
			&emojiReaction.Type,
			&emojiReaction.MessageId,
			&emojiReaction.ChatId,
			&emojiReaction.LocalChatID,
			&emojiReaction.Retracted)
		if err != nil {
			return nil, err
		}

		result = append(result, &emojiReaction)
	}

	return result, nil
}

// EmojiReactionsByChatIDs returns the emoji reactions for the queried messages, up to a maximum of 100, as it's a potentially unbound number.
// NOTE: This is not completely accurate, as the messages in the database might have change since the last call to `MessageByChatID`.
func (db sqlitePersistence) EmojiReactionsByChatIDs(chatIDs []string, currCursor string, limit int) ([]*EmojiReaction, error) {
	cursorWhere := ""
	if currCursor != "" {
		cursorWhere = "AND substr('0000000000000000000000000000000000000000000000000000000000000000' || m.clock_value, -64, 64) || m.id <= ?" //nolint: goconst

	}
	chatsLen := len(chatIDs)
	args := make([]interface{}, chatsLen*2)
	for i, v := range chatIDs {
		args[i] = v
	}
	for i, v := range chatIDs {
		args[chatsLen+i] = v
	}
	if currCursor != "" {
		args = append(args, currCursor)
	}
	args = append(args, limit)
	// NOTE: We match against local_chat_id for security reasons.
	// As a user could potentially send an emoji reaction for a one to
	// one/group chat that has no access to.
	// We also limit the number of emoji to a reasonable number (1000)
	// for now, as we don't want the client to choke on this.
	// The issue is that your own emoji might not be returned in such cases,
	// allowing the user to react to a post multiple times.
	// Jakubgs: Returning the whole list seems like a real overkill.
	// This will get very heavy in threads that have loads of reactions on loads of messages.
	// A more sensible response would just include a count and a bool telling you if you are in the list.
	// nolint: gosec
	query := fmt.Sprintf(`
			SELECT
			    e.clock_value,
			    e.source,
			    e.emoji_id,
			    e.message_id,
			    e.chat_id,
			    e.local_chat_id,
			    e.retracted
			FROM
				emoji_reactions e
			WHERE NOT(e.retracted)
			AND
			e.local_chat_id IN %s
			AND
			e.message_id IN
			(SELECT id FROM user_messages m WHERE NOT(m.hide) AND m.local_chat_id IN %s %s
			ORDER BY substr('0000000000000000000000000000000000000000000000000000000000000000' || m.clock_value, -64, 64) || m.id DESC LIMIT ?)
			LIMIT 1000
		`, "(?"+strings.Repeat(",?", chatsLen-1)+")", "(?"+strings.Repeat(",?", chatsLen-1)+")", cursorWhere)

	rows, err := db.db.Query(
		query,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*EmojiReaction
	for rows.Next() {
		var emojiReaction EmojiReaction
		err := rows.Scan(&emojiReaction.Clock,
			&emojiReaction.From,
			&emojiReaction.Type,
			&emojiReaction.MessageId,
			&emojiReaction.ChatId,
			&emojiReaction.LocalChatID,
			&emojiReaction.Retracted)
		if err != nil {
			return nil, err
		}

		result = append(result, &emojiReaction)
	}

	return result, nil
}

func (db sqlitePersistence) SaveMessages(messages []*common.Message) (err error) {
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

	allFields := db.tableUserMessagesAllFields()
	valuesVector := strings.Repeat("?, ", db.tableUserMessagesAllFieldsCount()-1) + "?"
	query := "INSERT INTO user_messages(" + allFields + ") VALUES (" + valuesVector + ")" // nolint: gosec
	stmt, err := tx.Prepare(query)
	if err != nil {
		return
	}

	for _, msg := range messages {
		var allValues []interface{}
		allValues, err = db.tableUserMessagesAllValues(msg)
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
	_, err := db.db.Exec(`DELETE FROM user_messages WHERE id = ?`, id)
	return err
}

func (db sqlitePersistence) HideMessage(id string) error {
	_, err := db.db.Exec(`UPDATE user_messages SET hide = 1, seen = 1 WHERE id = ?`, id)
	return err
}

func (db sqlitePersistence) DeleteMessagesByChatID(id string) error {
	_, err := db.db.Exec(`DELETE FROM user_messages WHERE local_chat_id = ?`, id)
	return err
}

func (db sqlitePersistence) MarkAllRead(chatID string) error {
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

	_, err = tx.Exec(`UPDATE user_messages SET seen = 1 WHERE local_chat_id = ? AND seen != 1`, chatID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE chats SET unviewed_message_count = 0 WHERE id = ?`, chatID)
	return err
}

func (db sqlitePersistence) MarkMessagesSeen(chatID string, ids []string) (uint64, error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return 0, err
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
	q := "UPDATE user_messages SET seen = 1 WHERE NOT(seen) AND id IN (" + inVector + ")" // nolint: gosec
	_, err = tx.Exec(q, idsArgs...)
	if err != nil {
		return 0, err
	}

	var count uint64
	row := tx.QueryRow("SELECT changes();")
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	// Update denormalized count
	_, err = tx.Exec(
		`UPDATE chats
              	SET unviewed_message_count =
		   (SELECT COUNT(1)
		   FROM user_messages
		   WHERE local_chat_id = ? AND seen = 0)
		WHERE id = ?`, chatID, chatID)
	return count, err
}

func (db sqlitePersistence) UpdateMessageOutgoingStatus(id string, newOutgoingStatus string) error {
	_, err := db.db.Exec(`
		UPDATE user_messages
		SET outgoing_status = ?
		WHERE id = ?
	`, newOutgoingStatus, id)
	return err
}

// BlockContact updates a contact, deletes all the messages and 1-to-1 chat, updates the unread messages count and returns a map with the new count
func (db sqlitePersistence) BlockContact(contact *Contact) ([]*Chat, error) {
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
		 FROM user_messages
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
			unviewed_message_count = (SELECT COUNT(1) FROM user_messages WHERE seen = 0 AND local_chat_id = chats.id)`)
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
		row := tx.QueryRow(`SELECT id FROM user_messages WHERE local_chat_id = ? ORDER BY clock_value DESC LIMIT 1`, c.ID)
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
				c.LastMessage = message

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

func (db sqlitePersistence) SaveEmojiReaction(emojiReaction *EmojiReaction) (err error) {
	query := "INSERT INTO emoji_reactions(id,clock_value,source,emoji_id,message_id,chat_id,local_chat_id,retracted) VALUES (?,?,?,?,?,?,?,?)"
	stmt, err := db.db.Prepare(query)
	if err != nil {
		return
	}

	_, err = stmt.Exec(
		emojiReaction.ID(),
		emojiReaction.Clock,
		emojiReaction.From,
		emojiReaction.Type,
		emojiReaction.MessageId,
		emojiReaction.ChatId,
		emojiReaction.LocalChatID,
		emojiReaction.Retracted,
	)

	return
}

func (db sqlitePersistence) EmojiReactionByID(id string) (*EmojiReaction, error) {
	row := db.db.QueryRow(
		`SELECT
			    clock_value,
			    source,
			    emoji_id,
			    message_id,
			    chat_id,
			    local_chat_id,
			    retracted
			FROM
				emoji_reactions
			WHERE
				emoji_reactions.id = ?
		`, id)

	emojiReaction := new(EmojiReaction)
	err := row.Scan(&emojiReaction.Clock,
		&emojiReaction.From,
		&emojiReaction.Type,
		&emojiReaction.MessageId,
		&emojiReaction.ChatId,
		&emojiReaction.LocalChatID,
		&emojiReaction.Retracted,
	)

	switch err {
	case sql.ErrNoRows:
		return nil, common.ErrRecordNotFound
	case nil:
		return emojiReaction, nil
	default:
		return nil, err
	}
}

func (db sqlitePersistence) SaveInvitation(invitation *GroupChatInvitation) (err error) {
	query := "INSERT INTO group_chat_invitations(id,source,chat_id,message,state,clock) VALUES (?,?,?,?,?,?)"
	stmt, err := db.db.Prepare(query)
	if err != nil {
		return
	}
	_, err = stmt.Exec(
		invitation.ID(),
		invitation.From,
		invitation.ChatId,
		invitation.IntroductionMessage,
		invitation.State,
		invitation.Clock,
	)

	return
}

func (db sqlitePersistence) GetGroupChatInvitations() (rst []*GroupChatInvitation, err error) {

	tx, err := db.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	bRows, err := tx.Query(`SELECT
			    source,
			    chat_id,
			    message,
			    state,
			    clock
			FROM
				group_chat_invitations`)
	if err != nil {
		return
	}
	defer bRows.Close()
	for bRows.Next() {
		invitation := GroupChatInvitation{}
		err = bRows.Scan(
			&invitation.From,
			&invitation.ChatId,
			&invitation.IntroductionMessage,
			&invitation.State,
			&invitation.Clock)
		if err != nil {
			return nil, err
		}
		rst = append(rst, &invitation)
	}

	return rst, nil
}

func (db sqlitePersistence) InvitationByID(id string) (*GroupChatInvitation, error) {
	row := db.db.QueryRow(
		`SELECT
			    source,
			    chat_id,
			    message,
			    state,
			    clock
			FROM
				group_chat_invitations
			WHERE
				group_chat_invitations.id = ?
		`, id)

	chatInvitations := new(GroupChatInvitation)
	err := row.Scan(&chatInvitations.From,
		&chatInvitations.ChatId,
		&chatInvitations.IntroductionMessage,
		&chatInvitations.State,
		&chatInvitations.Clock,
	)

	switch err {
	case sql.ErrNoRows:
		return nil, common.ErrRecordNotFound
	case nil:
		return chatInvitations, nil
	default:
		return nil, err
	}
}
