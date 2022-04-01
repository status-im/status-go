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
		edited_at,
		deleted,
		rtl,
		line_count,
		response_to,
		gap_from,
		gap_to,
		contact_request_state,
		mentioned`
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
		m1.image_payload,
		m1.image_type,
		COALESCE(m1.audio_duration_ms,0),
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
		m1.edited_at,
		m1.deleted,
		m1.rtl,
		m1.line_count,
		m1.response_to,
		m1.gap_from,
		m1.gap_to,
		m1.contact_request_state,
		m1.mentioned,
		m2.source,
		m2.text,
		m2.parsed_text,
		m2.audio_duration_ms,
		m2.community_id,
		m2.id,
        m2.content_type,
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
	var quotedID sql.NullString
	var ContentType sql.NullInt64
	var quotedText sql.NullString
	var quotedParsedText []byte
	var quotedFrom sql.NullString
	var quotedAudioDuration sql.NullInt64
	var quotedCommunityID sql.NullString
	var serializedMentions []byte
	var serializedLinks []byte
	var alias sql.NullString
	var identicon sql.NullString
	var communityID sql.NullString
	var gapFrom sql.NullInt64
	var gapTo sql.NullInt64
	var editedAt sql.NullInt64
	var deleted sql.NullBool
	var contactRequestState sql.NullInt64

	sticker := &protobuf.StickerMessage{}
	command := &common.CommandParameters{}
	audio := &protobuf.AudioMessage{}
	image := &protobuf.ImageMessage{}

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
		&image.Payload,
		&image.Type,
		&audio.DurationMs,
		&communityID,
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
		&editedAt,
		&deleted,
		&message.RTL,
		&message.LineCount,
		&message.ResponseTo,
		&gapFrom,
		&gapTo,
		&contactRequestState,
		&message.Mentioned,
		&quotedFrom,
		&quotedText,
		&quotedParsedText,
		&quotedAudioDuration,
		&quotedCommunityID,
		&quotedID,
		&ContentType,
		&alias,
		&identicon,
	}
	err := row.Scan(append(args, others...)...)
	if err != nil {
		return err
	}

	if editedAt.Valid {
		message.EditedAt = uint64(editedAt.Int64)
	}

	if deleted.Valid {
		message.Deleted = deleted.Bool
	}

        if contactRequestState.Valid {
          message.ContactRequestState = common.ContactRequestState(contactRequestState.Int64)
        }

	if quotedText.Valid {
		message.QuotedMessage = &common.QuotedMessage{
			ID:          quotedID.String,
			ContentType: ContentType.Int64,
			From:        quotedFrom.String,
			Text:        quotedText.String,
			ParsedText:  quotedParsedText,
			CommunityID: quotedCommunityID.String,
		}
	}
	message.Alias = alias.String
	message.Identicon = identicon.String

	if gapFrom.Valid && gapTo.Valid {
		message.GapParameters = &common.GapParameters{
			From: uint32(gapFrom.Int64),
			To:   uint32(gapTo.Int64),
		}
	}
	if communityID.Valid {
		message.CommunityID = communityID.String
	}

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

	case protobuf.ChatMessage_IMAGE:
		img := protobuf.ImageMessage{
			Payload: image.Payload,
			Type:    image.Type,
		}
		message.Payload = &protobuf.ChatMessage_Image{Image: &img}
	}

	return nil
}

func (db sqlitePersistence) tableUserMessagesAllValues(message *common.Message) ([]interface{}, error) {
	var gapFrom, gapTo uint32

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

	if message.GapParameters != nil {
		gapFrom = message.GapParameters.From
		gapTo = message.GapParameters.To
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
		int64(message.EditedAt),
		message.Deleted,
		message.RTL,
		message.LineCount,
		message.ResponseTo,
		gapFrom,
		gapTo,
		message.ContactRequestState,
		message.Mentioned,
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

func (db sqlitePersistence) PendingContactRequests(currCursor string, limit int) ([]*common.Message, string, error) {
	cursorWhere := ""
	if currCursor != "" {
		cursorWhere = "AND cursor <= ?" //nolint: goconst
	}
	allFields := db.tableUserMessagesAllFieldsJoin()
	args := []interface{}{protobuf.ChatMessage_CONTACT_REQUEST}
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
				NOT(m1.hide) AND NOT(m1.seen) AND m1.content_type = ? %s
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

// AllMessageByChatIDWhichMatchPattern returns all messages which match the search
// term, for a given chatID in descending order.
// Ordering is accomplished using two concatenated values: ClockValue and ID.
// These two values are also used to compose a cursor which is returned to the result.
func (db sqlitePersistence) AllMessageByChatIDWhichMatchTerm(chatID string, searchTerm string, caseSensitive bool) ([]*common.Message, error) {
	if searchTerm == "" {
		return nil, fmt.Errorf("empty search term")
	}

	searchCond := ""
	if caseSensitive {
		searchCond = "AND m1.text LIKE '%' || ? || '%'"
	} else {
		searchCond = "AND LOWER(m1.text) LIKE LOWER('%' || ? || '%')"
	}

	allFields := db.tableUserMessagesAllFieldsJoin()

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
		`, allFields, searchCond),
		chatID, searchTerm,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		result []*common.Message
	)
	for rows.Next() {
		var (
			message common.Message
			cursor  string
		)
		if err := db.tableUserMessagesScanAllFields(rows, &message, &cursor); err != nil {
			return nil, err
		}
		result = append(result, &message)
	}

	return result, nil
}

// AllMessagesFromChatsAndCommunitiesWhichMatchTerm returns all messages which match the search
// term, if they belong to either any chat from the chatIds array or any channel of any community
// from communityIds array.
// Ordering is accomplished using two concatenated values: ClockValue and ID.
// These two values are also used to compose a cursor which is returned to the result.
func (db sqlitePersistence) AllMessagesFromChatsAndCommunitiesWhichMatchTerm(communityIds []string, chatIds []string, searchTerm string, caseSensitive bool) ([]*common.Message, error) {
	if searchTerm == "" {
		return nil, fmt.Errorf("empty search term")
	}

	chatsCond := ""
	if len(chatIds) > 0 {
		inVector := strings.Repeat("?, ", len(chatIds)-1) + "?"
		chatsCond = `m1.local_chat_id IN (%s)`
		chatsCond = fmt.Sprintf(chatsCond, inVector)
	}

	communitiesCond := ""
	if len(communityIds) > 0 {
		inVector := strings.Repeat("?, ", len(communityIds)-1) + "?"
		communitiesCond = `m1.local_chat_id IN (SELECT id FROM chats WHERE community_id IN (%s))`
		communitiesCond = fmt.Sprintf(communitiesCond, inVector)
	}

	searchCond := ""
	if caseSensitive {
		searchCond = "m1.text LIKE '%' || ? || '%'"
	} else {
		searchCond = "LOWER(m1.text) LIKE LOWER('%' || ? || '%')"
	}

	finalCond := "AND %s AND %s"
	if len(communityIds) > 0 && len(chatIds) > 0 {
		finalCond = "AND (%s OR %s) AND %s"
		finalCond = fmt.Sprintf(finalCond, chatsCond, communitiesCond, searchCond)
	} else if len(chatIds) > 0 {
		finalCond = fmt.Sprintf(finalCond, chatsCond, searchCond)
	} else if len(communityIds) > 0 {
		finalCond = fmt.Sprintf(finalCond, communitiesCond, searchCond)
	} else {
		return nil, fmt.Errorf("you must specify either community ids or chat ids or both")
	}

	var parameters []string
	parameters = append(parameters, chatIds...)
	parameters = append(parameters, communityIds...)
	parameters = append(parameters, searchTerm)

	idsArgs := make([]interface{}, 0, len(parameters))
	for _, param := range parameters {
		idsArgs = append(idsArgs, param)
	}

	allFields := db.tableUserMessagesAllFieldsJoin()

	finalQuery := fmt.Sprintf( // nolint: gosec
		`
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
			NOT(m1.hide) %s
		ORDER BY cursor DESC
	`, allFields, finalCond)

	rows, err := db.db.Query(finalQuery, idsArgs...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		result []*common.Message
	)
	for rows.Next() {
		var (
			message common.Message
			cursor  string
		)
		if err := db.tableUserMessagesScanAllFields(rows, &message, &cursor); err != nil {
			return nil, err
		}
		result = append(result, &message)
	}

	return result, nil
}

func (db sqlitePersistence) AllChatIDsByCommunity(communityID string) ([]string, error) {
	rows, err := db.db.Query("SELECT id FROM chats WHERE community_id = ?", communityID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []string

	for rows.Next() {
		var chatID string
		err = rows.Scan(&chatID)
		if err != nil {
			return nil, err
		}
		rst = append(rst, chatID)
	}

	return rst, nil
}

// PinnedMessageByChatID returns all pinned messages for a given chatID in descending order.
// Ordering is accomplished using two concatenated values: ClockValue and ID.
// These two values are also used to compose a cursor which is returned to the result.
func (db sqlitePersistence) PinnedMessageByChatIDs(chatIDs []string, currCursor string, limit int) ([]*common.PinnedMessage, string, error) {
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

	limitStr := ""
	if limit > -1 {
		args = append(args, limit+1) // take one more to figure our whether a cursor should be returned
	}
	// Build a new column `cursor` at the query time by having a fixed-sized clock value at the beginning
	// concatenated with message ID. Results are sorted using this new column.
	// This new column values can also be returned as a cursor for subsequent requests.
	rows, err := db.db.Query(
		fmt.Sprintf(`
			SELECT
				%s,
				pm.clock_value as pinnedAt,
				pm.pinned_by as pinnedBy,
				substr('0000000000000000000000000000000000000000000000000000000000000000' || m1.clock_value, -64, 64) || m1.id as cursor
			FROM
				pin_messages pm
			JOIN
				user_messages m1
			ON
				pm.message_id = m1.id
			LEFT JOIN
				user_messages m2
			ON
				m1.response_to = m2.id
			LEFT JOIN
				contacts c
			ON
				m1.source = c.id
			WHERE
				pm.pinned = 1
				AND NOT(m1.hide) AND m1.local_chat_id IN %s %s
			ORDER BY cursor DESC
			%s
		`, allFields, "(?"+strings.Repeat(",?", len(chatIDs)-1)+")", cursorWhere, limitStr),
		args..., // take one more to figure our whether a cursor should be returned
	)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var (
		result  []*common.PinnedMessage
		cursors []string
	)
	for rows.Next() {
		var (
			message  common.Message
			pinnedAt uint64
			pinnedBy string
			cursor   string
		)
		if err := db.tableUserMessagesScanAllFields(rows, &message, &pinnedAt, &pinnedBy, &cursor); err != nil {
			return nil, "", err
		}
		pinnedMessage := &common.PinnedMessage{
			Message:  &message,
			PinnedAt: pinnedAt,
			PinnedBy: pinnedBy,
		}
		result = append(result, pinnedMessage)
		cursors = append(cursors, cursor)
	}

	var newCursor string

	if limit > -1 && len(result) > limit && cursors != nil {
		newCursor = cursors[limit]
		result = result[:limit]
	}
	return result, newCursor, nil
}

func (db sqlitePersistence) PinnedMessageByChatID(chatID string, currCursor string, limit int) ([]*common.PinnedMessage, string, error) {
	return db.PinnedMessageByChatIDs([]string{chatID}, currCursor, limit)
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

// EmojiReactionsByChatIDMessageID returns the emoji reactions for the queried message.
func (db sqlitePersistence) EmojiReactionsByChatIDMessageID(chatID string, messageID string) ([]*EmojiReaction, error) {

	args := []interface{}{chatID, messageID}
	query := `SELECT
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
			e.message_id = ?
			LIMIT 1000`

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

func (db sqlitePersistence) SavePinMessages(messages []*common.PinMessage) (err error) {
	tx, err := db.db.BeginTx(context.Background(), nil)
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

	// select
	selectQuery := "SELECT clock_value FROM pin_messages WHERE id = ?"

	// insert
	allInsertFields := `id, message_id, whisper_timestamp, chat_id, local_chat_id, clock_value, pinned, pinned_by`
	insertValues := strings.Repeat("?, ", strings.Count(allInsertFields, ",")) + "?"
	insertQuery := "INSERT INTO pin_messages(" + allInsertFields + ") VALUES (" + insertValues + ")" // nolint: gosec
	insertStmt, err := tx.Prepare(insertQuery)
	if err != nil {
		return
	}

	// update
	updateQuery := "UPDATE pin_messages SET pinned = ?, clock_value = ?, pinned_by = ? WHERE id = ?"
	updateStmt, err := tx.Prepare(updateQuery)
	if err != nil {
		return
	}

	for _, message := range messages {
		row := tx.QueryRow(selectQuery, message.ID)
		var existingClock uint64
		switch err = row.Scan(&existingClock); err {
		case sql.ErrNoRows:
			// not found, insert new record
			allValues := []interface{}{
				message.ID,
				message.MessageId,
				message.WhisperTimestamp,
				message.ChatId,
				message.LocalChatID,
				message.Clock,
				message.Pinned,
				message.From,
			}
			_, err = insertStmt.Exec(allValues...)
			if err != nil {
				return
			}
		case nil:
			// found, update if current message is more recent, otherwise skip
			if existingClock < message.Clock {
				// update
				_, err = updateStmt.Exec(message.Pinned, message.Clock, message.From, message.ID)
				if err != nil {
					return
				}
			}

		default:
			return
		}
	}

	return
}

func (db sqlitePersistence) DeleteMessage(id string) error {
	_, err := db.db.Exec(`DELETE FROM user_messages WHERE id = ?`, id)
	return err
}

func (db sqlitePersistence) DeleteMessages(ids []string) error {
	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}
	inVector := strings.Repeat("?, ", len(ids)-1) + "?"

	_, err := db.db.Exec("DELETE FROM user_messages WHERE id IN ("+inVector+")", idsArgs...) // nolint: gosec

	return err
}

func (db sqlitePersistence) HideMessage(id string) error {
	_, err := db.db.Exec(`UPDATE user_messages SET hide = 1, seen = 1 WHERE id = ?`, id)
	return err
}

// SetHideOnMessage set the hide flag, but not the seen flag, as it's needed by the client to understand whether the count should be updated
func (db sqlitePersistence) SetHideOnMessage(id string) error {
	_, err := db.db.Exec(`UPDATE user_messages SET hide = 1 WHERE id = ?`, id)
	return err
}

func (db sqlitePersistence) DeleteMessagesByChatID(id string) error {
	return db.deleteMessagesByChatID(id, nil)
}

func (db sqlitePersistence) deleteMessagesByChatID(id string, tx *sql.Tx) (err error) {
	if tx == nil {
		tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
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
	}

	_, err = tx.Exec(`DELETE FROM user_messages WHERE local_chat_id = ?`, id)
	if err != nil {
		return
	}

	_, err = tx.Exec(`DELETE FROM pin_messages WHERE local_chat_id = ?`, id)

	return
}

func (db sqlitePersistence) deleteMessagesByChatIDAndClockValueLessThanOrEqual(id string, clock uint64, tx *sql.Tx) (unViewedMessages, unViewedMentions uint, err error) {
	if tx == nil {
		tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
		if err != nil {
			return 0, 0, err
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

	_, err = tx.Exec(`DELETE FROM user_messages WHERE local_chat_id = ? AND clock_value <= ?`, id, clock)
	if err != nil {
		return
	}

	_, err = tx.Exec(`DELETE FROM pin_messages WHERE local_chat_id = ? AND clock_value <= ?`, id, clock)
	if err != nil {
		return
	}

	_, err = tx.Exec(
		`UPDATE chats
		   SET unviewed_message_count =
		   (SELECT COUNT(1)
		   FROM user_messages
		   WHERE local_chat_id = ? AND seen = 0),
		   unviewed_mentions_count =
		   (SELECT COUNT(1)
		   FROM user_messages
		   WHERE local_chat_id = ? AND seen = 0 AND mentioned),
                   highlight = 0
		WHERE id = ?`, id, id, id)

	if err != nil {
		return 0, 0, err
	}

	err = tx.QueryRow(`SELECT unviewed_message_count, unviewed_mentions_count FROM chats 
				WHERE id = ?`, id).Scan(&unViewedMessages, &unViewedMentions)

	return
}

func (db sqlitePersistence) MarkAllRead(chatID string, clock uint64) (int64, int64, error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	seenResult, err := tx.Exec(`UPDATE user_messages SET seen = 1 WHERE local_chat_id = ? AND not(seen) AND clock_value <= ? AND not(mentioned)`, chatID, clock)
	if err != nil {
		return 0, 0, err
	}

	seen, err := seenResult.RowsAffected()
	if err != nil {
		return 0, 0, err
	}

	mentionedResult, err := tx.Exec(`UPDATE user_messages SET seen = 1 WHERE local_chat_id = ? AND not(seen) AND clock_value <= ? AND mentioned`, chatID, clock)
	if err != nil {
		return 0, 0, err
	}

	mentioned, err := mentionedResult.RowsAffected()
	if err != nil {
		return 0, 0, err
	}

	_, err = tx.Exec(
		`UPDATE chats
		   SET unviewed_message_count =
		   (SELECT COUNT(1)
		   FROM user_messages
		   WHERE local_chat_id = ? AND seen = 0),
		   unviewed_mentions_count =
		   (SELECT COUNT(1)
		   FROM user_messages
		   WHERE local_chat_id = ? AND seen = 0 AND mentioned),
                   highlight = 0
		WHERE id = ?`, chatID, chatID, chatID)

	if err != nil {
		return 0, 0, err
	}

	return (seen + mentioned), mentioned, nil
}

func (db sqlitePersistence) MarkAllReadMultiple(chatIDs []string) error {
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

	idsArgs := make([]interface{}, 0, len(chatIDs))
	for _, id := range chatIDs {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(chatIDs)-1) + "?"

	q := "UPDATE user_messages SET seen = 1 WHERE local_chat_id IN (%s) AND seen != 1"
	q = fmt.Sprintf(q, inVector)
	_, err = tx.Exec(q, idsArgs...)
	if err != nil {
		return err
	}

	q = "UPDATE chats SET unviewed_mentions_count = 0, unviewed_message_count = 0, highlight = 0 WHERE id IN (%s)"
	q = fmt.Sprintf(q, inVector)
	_, err = tx.Exec(q, idsArgs...)
	return err
}

func (db sqlitePersistence) MarkMessagesSeen(chatID string, ids []string) (uint64, uint64, error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return 0, 0, err
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
	q := "UPDATE user_messages SET seen = 1 WHERE NOT(seen) AND mentioned AND id IN (" + inVector + ")" // nolint: gosec
	_, err = tx.Exec(q, idsArgs...)
	if err != nil {
		return 0, 0, err
	}

	var countWithMentions uint64
	row := tx.QueryRow("SELECT changes();")
	if err := row.Scan(&countWithMentions); err != nil {
		return 0, 0, err
	}

	q = "UPDATE user_messages SET seen = 1 WHERE NOT(seen) AND NOT(mentioned) AND id IN (" + inVector + ")" // nolint: gosec
	_, err = tx.Exec(q, idsArgs...)
	if err != nil {
		return 0, 0, err
	}

	var countNoMentions uint64
	row = tx.QueryRow("SELECT changes();")
	if err := row.Scan(&countNoMentions); err != nil {
		return 0, 0, err
	}

	// Update denormalized count
	_, err = tx.Exec(
		`UPDATE chats
              	SET unviewed_message_count =
		   (SELECT COUNT(1)
		   FROM user_messages
		   WHERE local_chat_id = ? AND seen = 0),
		   unviewed_mentions_count =
		   (SELECT COUNT(1)
		   FROM user_messages
		   WHERE local_chat_id = ? AND seen = 0 AND mentioned),
                   highlight = 0
		WHERE id = ?`, chatID, chatID, chatID)
	return countWithMentions + countNoMentions, countWithMentions, err
}

func (db sqlitePersistence) UpdateMessageOutgoingStatus(id string, newOutgoingStatus string) error {
	_, err := db.db.Exec(`
		UPDATE user_messages
		SET outgoing_status = ?
		WHERE id = ? AND outgoing_status != ?
	`, newOutgoingStatus, id, common.OutgoingStatusDelivered)
	return err
}

// BlockContact updates a contact, deletes all the messages and 1-to-1 chat, updates the unread messages count and returns a map with the new count
func (db sqlitePersistence) BlockContact(contact *Contact, isDesktopFunc bool) ([]*Chat, error) {
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

	if !isDesktopFunc {
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
	}

	// Update contact
	err = db.SaveContact(contact, tx)
	if err != nil {
		return nil, err
	}

	if !isDesktopFunc {
		// Delete one-to-one chat
		_, err = tx.Exec("DELETE FROM chats WHERE id = ?", contact.ID)
		if err != nil {
			return nil, err
		}
	}

	// Recalculate denormalized fields
	_, err = tx.Exec(`
		UPDATE chats
		SET
			unviewed_message_count = (SELECT COUNT(1) FROM user_messages WHERE seen = 0 AND local_chat_id = chats.id),
			unviewed_mentions_count = (SELECT COUNT(1) FROM user_messages WHERE seen = 0 AND local_chat_id = chats.id AND mentioned)`)
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

// ClearHistory deletes all the messages for a chat and updates it's values
func (db sqlitePersistence) ClearHistory(chat *Chat, currentClockValue uint64) (err error) {
	var tx *sql.Tx

	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
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
	err = db.clearHistory(chat, currentClockValue, tx, false)

	return
}

func (db sqlitePersistence) ClearHistoryFromSyncMessage(chat *Chat, currentClockValue uint64) (err error) {
	var tx *sql.Tx

	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
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
	err = db.clearHistoryFromSyncMessage(chat, currentClockValue, tx)

	return
}

// Deactivate chat sets a chat as inactive and clear its history
func (db sqlitePersistence) DeactivateChat(chat *Chat, currentClockValue uint64) (err error) {
	var tx *sql.Tx

	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
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
	err = db.deactivateChat(chat, currentClockValue, tx)

	return
}

func (db sqlitePersistence) deactivateChat(chat *Chat, currentClockValue uint64, tx *sql.Tx) error {
	chat.Active = false
	err := db.saveChat(tx, *chat)
	if err != nil {
		return err
	}

	return db.clearHistory(chat, currentClockValue, tx, true)
}

func (db sqlitePersistence) SaveDelete(deleteMessage DeleteMessage) error {
	_, err := db.db.Exec(`INSERT INTO user_messages_deletes (clock, chat_id, message_id, source, id) VALUES(?,?,?,?,?)`, deleteMessage.Clock, deleteMessage.ChatId, deleteMessage.MessageId, deleteMessage.From, deleteMessage.ID)
	return err
}

func (db sqlitePersistence) GetDeletes(messageID string, from string) ([]*DeleteMessage, error) {
	rows, err := db.db.Query(`SELECT clock, chat_id, message_id, source, id FROM user_messages_deletes WHERE message_id = ? AND source = ? ORDER BY CLOCK DESC`, messageID, from)
	if err != nil {
		return nil, err
	}

	var messages []*DeleteMessage

	for rows.Next() {
		d := &DeleteMessage{}
		err := rows.Scan(&d.Clock, &d.ChatId, &d.MessageId, &d.From, &d.ID)
		if err != nil {
			return nil, err
		}
		messages = append(messages, d)

	}
	return messages, nil
}

func (db sqlitePersistence) SaveEdit(editMessage EditMessage) error {
	_, err := db.db.Exec(`INSERT INTO user_messages_edits (clock, chat_id, message_id, text, source, id) VALUES(?,?,?,?,?,?)`, editMessage.Clock, editMessage.ChatId, editMessage.MessageId, editMessage.Text, editMessage.From, editMessage.ID)
	return err
}

func (db sqlitePersistence) GetEdits(messageID string, from string) ([]*EditMessage, error) {
	rows, err := db.db.Query(`SELECT clock, chat_id, message_id, source, text, id FROM user_messages_edits WHERE message_id = ? AND source = ? ORDER BY CLOCK DESC`, messageID, from)
	if err != nil {
		return nil, err
	}

	var messages []*EditMessage

	for rows.Next() {
		e := &EditMessage{}
		err := rows.Scan(&e.Clock, &e.ChatId, &e.MessageId, &e.From, &e.Text, &e.ID)
		if err != nil {
			return nil, err
		}
		messages = append(messages, e)

	}
	return messages, nil
}

func (db sqlitePersistence) clearHistory(chat *Chat, currentClockValue uint64, tx *sql.Tx, deactivate bool) error {
	// Set deleted at clock value if it's not a public chat so that
	// old messages will be discarded, or if it's a straight clear history
	if !deactivate || (!chat.Public() && !chat.ProfileUpdates() && !chat.Timeline()) {
		if chat.LastMessage != nil && chat.LastMessage.Clock != 0 {
			chat.DeletedAtClockValue = chat.LastMessage.Clock
		}
		chat.DeletedAtClockValue = currentClockValue
	}

	// Reset synced-to/from
	syncedTo := uint32(currentClockValue / 1000)
	chat.SyncedTo = syncedTo
	chat.SyncedFrom = 0

	chat.LastMessage = nil
	chat.UnviewedMessagesCount = 0
	chat.UnviewedMentionsCount = 0
	chat.Highlight = true

	err := db.deleteMessagesByChatID(chat.ID, tx)
	if err != nil {
		return err
	}

	err = db.saveChat(tx, *chat)
	return err
}

func (db sqlitePersistence) clearHistoryFromSyncMessage(chat *Chat, clearedAt uint64, tx *sql.Tx) error {
	chat.DeletedAtClockValue = clearedAt

	// Reset synced-to/from
	syncedTo := uint32(clearedAt / 1000)
	chat.SyncedTo = syncedTo
	chat.SyncedFrom = 0

	unViewedMessagesCount, unViewedMentionsCount, err := db.deleteMessagesByChatIDAndClockValueLessThanOrEqual(chat.ID, clearedAt, tx)
	if err != nil {
		return err
	}

	chat.UnviewedMessagesCount = unViewedMessagesCount
	chat.UnviewedMentionsCount = unViewedMentionsCount

	if chat.LastMessage != nil && chat.LastMessage.Clock <= clearedAt {
		chat.LastMessage = nil
	}

	err = db.saveChat(tx, *chat)
	return err
}

func (db sqlitePersistence) SetContactRequestState(id string, state common.ContactRequestState) error {
  _, err := db.db.Exec(`UPDATE user_messages SET contact_request_state = ? WHERE id = ?`, state, id)
  return err
}
