package protocol

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
)

func (db sqlitePersistence) DeleteActivityCenterNotification(id []byte) error {

	_, err := db.db.Exec(`DELETE FROM activity_center_notifications WHERE id = ?`, id)
	return err
}

func (db sqlitePersistence) DeleteActivityCenterNotificationForMessage(chatID string, messageID string) error {
	var tx *sql.Tx
	var err error

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

	params := activityCenterQueryParams{
		chatID: chatID,
	}

	_, notifications, err := db.buildActivityCenterQuery(tx, params)

	if err != nil {
		return err
	}

	var ids []types.HexBytes

	for _, notification := range notifications {
		if notification.LastMessage != nil && notification.LastMessage.ID == messageID {
			ids = append(ids, notification.ID)
		}

		if notification.Message != nil && notification.Message.ID == messageID {
			ids = append(ids, notification.ID)
		}
	}

	if len(ids) > 0 {
		idsArgs := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			idsArgs = append(idsArgs, id)
		}

		inVector := strings.Repeat("?, ", len(ids)-1) + "?"
		query := "UPDATE activity_center_notifications SET read = 1, dismissed = 1 WHERE id IN (" + inVector + ")" // nolint: gosec
		_, err = tx.Exec(query, idsArgs...)
		return err
	}

	return nil
}

func (db sqlitePersistence) SaveActivityCenterNotification(notification *ActivityCenterNotification) error {
	var tx *sql.Tx
	var err error

	err = notification.Valid()
	if err != nil {
		return err
	}

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

	if notification.Type == ActivityCenterNotificationTypeNewOneToOne ||
		notification.Type == ActivityCenterNotificationTypeNewPrivateGroupChat {
		// Delete other notifications so it pop us again if not currently dismissed
		_, err = tx.Exec(`DELETE FROM activity_center_notifications WHERE id = ? AND (dismissed OR accepted)`, notification.ID)
		if err != nil {
			return err
		}
	}

	// encode message
	var encodedMessage []byte
	if notification.Message != nil {
		encodedMessage, err = json.Marshal(notification.Message)
		if err != nil {
			return err
		}
	}

	// encode message
	var encodedReplyMessage []byte
	if notification.ReplyMessage != nil {
		encodedReplyMessage, err = json.Marshal(notification.ReplyMessage)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO activity_center_notifications (id, timestamp, notification_type, chat_id, community_id, membership_status, message, reply_message, author, contact_verification_status, read, accepted, dismissed) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		notification.ID,
		notification.Timestamp,
		notification.Type,
		notification.ChatID,
		notification.CommunityID,
		notification.MembershipStatus,
		encodedMessage,
		encodedReplyMessage,
		notification.Author,
		notification.ContactVerificationStatus,
		notification.Read,
		notification.Accepted,
		notification.Dismissed,
	)

	// When we have inserted or updated unread notification - mark whole activity_center_settings as unseen
	if err == nil && !notification.Read {
		_, err = tx.Exec(`UPDATE activity_center_states SET has_seen = 0`)
	}

	return err
}

func (db sqlitePersistence) unmarshalActivityCenterNotificationRow(row *sql.Row) (*ActivityCenterNotification, error) {
	var chatID sql.NullString
	var communityID sql.NullString
	var lastMessageBytes []byte
	var messageBytes []byte
	var replyMessageBytes []byte
	var name sql.NullString
	var author sql.NullString
	notification := &ActivityCenterNotification{}
	err := row.Scan(
		&notification.ID,
		&notification.Timestamp,
		&notification.Type,
		&chatID,
		&communityID,
		&notification.MembershipStatus,
		&notification.Read,
		&notification.Accepted,
		&notification.Dismissed,
		&messageBytes,
		&lastMessageBytes,
		&replyMessageBytes,
		&notification.ContactVerificationStatus,
		&name,
		&author)

	if err != nil {
		return nil, err
	}

	if chatID.Valid {
		notification.ChatID = chatID.String
	}

	if communityID.Valid {
		notification.CommunityID = communityID.String
	}

	if name.Valid {
		notification.Name = name.String
	}

	if author.Valid {
		notification.Author = author.String
	}

	// Restore last message
	if lastMessageBytes != nil {
		lastMessage := &common.Message{}
		if err = json.Unmarshal(lastMessageBytes, lastMessage); err != nil {
			return nil, err
		}
		notification.LastMessage = lastMessage
	}

	// Restore message
	if messageBytes != nil {
		message := &common.Message{}
		if err = json.Unmarshal(messageBytes, message); err != nil {
			return nil, err
		}
		notification.Message = message
	}

	// Restore reply message
	if replyMessageBytes != nil {
		replyMessage := &common.Message{}
		if err = json.Unmarshal(replyMessageBytes, replyMessage); err != nil {
			return nil, err
		}
		notification.ReplyMessage = replyMessage
	}

	return notification, nil
}

func (db sqlitePersistence) unmarshalActivityCenterNotificationRows(rows *sql.Rows) (string, []*ActivityCenterNotification, error) {
	var notifications []*ActivityCenterNotification
	latestCursor := ""
	for rows.Next() {
		var chatID sql.NullString
		var communityID sql.NullString
		var lastMessageBytes []byte
		var messageBytes []byte
		var replyMessageBytes []byte
		var name sql.NullString
		var author sql.NullString
		notification := &ActivityCenterNotification{}
		err := rows.Scan(
			&notification.ID,
			&notification.Timestamp,
			&notification.Type,
			&chatID,
			&communityID,
			&notification.MembershipStatus,
			&notification.Read,
			&notification.Accepted,
			&notification.Dismissed,
			&messageBytes,
			&lastMessageBytes,
			&replyMessageBytes,
			&notification.ContactVerificationStatus,
			&name,
			&author,
			&latestCursor)
		if err != nil {
			return "", nil, err
		}

		if chatID.Valid {
			notification.ChatID = chatID.String
		}

		if communityID.Valid {
			notification.CommunityID = communityID.String
		}

		if name.Valid {
			notification.Name = name.String
		}

		if author.Valid {
			notification.Author = author.String
		}

		// Restore last message
		if lastMessageBytes != nil {
			lastMessage := &common.Message{}
			if err = json.Unmarshal(lastMessageBytes, lastMessage); err != nil {
				return "", nil, err
			}
			notification.LastMessage = lastMessage
		}

		// Restore message
		if messageBytes != nil {
			message := &common.Message{}
			if err = json.Unmarshal(messageBytes, message); err != nil {
				return "", nil, err
			}
			notification.Message = message
		}

		// Restore reply message
		if replyMessageBytes != nil {
			replyMessage := &common.Message{}
			if err = json.Unmarshal(replyMessageBytes, replyMessage); err != nil {
				return "", nil, err
			}
			notification.ReplyMessage = replyMessage
		}

		notifications = append(notifications, notification)
	}

	return latestCursor, notifications, nil

}

type ActivityCenterQueryParamsRead uint

const (
	ActivityCenterQueryParamsReadRead = iota + 1
	ActivityCenterQueryParamsReadUnread
	ActivityCenterQueryParamsReadAll
)

type activityCenterQueryParams struct {
	cursor              string
	limit               uint64
	ids                 []types.HexBytes
	chatID              string
	author              string
	read                ActivityCenterQueryParamsRead
	accepted            bool
	activityCenterTypes []ActivityCenterType
}

func (db sqlitePersistence) buildActivityCenterQuery(tx *sql.Tx, params activityCenterQueryParams) (string, []*ActivityCenterNotification, error) {
	var args []interface{}
	var conditions []string

	cursor := params.cursor
	ids := params.ids
	author := params.author
	activityCenterTypes := params.activityCenterTypes
	limit := params.limit
	chatID := params.chatID
	read := params.read
	accepted := params.accepted

	if cursor != "" {
		conditions = append(conditions, "cursor <= ?")
		args = append(args, cursor)
	}

	if len(ids) != 0 {
		inVector := strings.Repeat("?, ", len(ids)-1) + "?"
		conditions = append(conditions, fmt.Sprintf("a.id IN (%s)", inVector))
		for _, id := range ids {
			args = append(args, id)
		}
	}

	switch read {
	case ActivityCenterQueryParamsReadRead:
		conditions = append(conditions, "a.read = 1")
	case ActivityCenterQueryParamsReadUnread:
		conditions = append(conditions, "NOT a.read")
	}

	if !accepted {
		conditions = append(conditions, "NOT a.accepted")
	}

	if chatID != "" {
		conditions = append(conditions, "a.chat_id = ?")
		args = append(args, chatID)
	}

	if author != "" {
		conditions = append(conditions, "a.author = ?")
		args = append(args, author)
	}

	if len(activityCenterTypes) > 0 {
		inVector := strings.Repeat("?, ", len(activityCenterTypes)-1) + "?"
		conditions = append(conditions, fmt.Sprintf("a.notification_type IN (%s)", inVector))
		for _, activityCenterType := range activityCenterTypes {
			args = append(args, activityCenterType)
		}
	}

	var conditionsString string
	if len(conditions) > 0 {
		conditionsString = " WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf( // nolint: gosec
		`
	SELECT
	a.id,
	a.timestamp,
	a.notification_type,
	a.chat_id,
	a.community_id,
	a.membership_status,
	a.read,
	a.accepted,
	a.dismissed,
	a.message,
	c.last_message,
	a.reply_message,
	a.contact_verification_status,
	c.name,
	a.author,
	substr('0000000000000000000000000000000000000000000000000000000000000000' || a.timestamp, -64, 64) || hex(a.id) as cursor
	FROM activity_center_notifications a
	LEFT JOIN chats c
	ON
	c.id = a.chat_id
	%s
	ORDER BY cursor DESC`, conditionsString)

	if limit != 0 {
		args = append(args, limit)
		query += ` LIMIT ?`
	}

	rows, err := tx.Query(query, args...)
	if err != nil {
		return "", nil, err
	}

	return db.unmarshalActivityCenterNotificationRows(rows)
}

func (db sqlitePersistence) runActivityCenterIDQuery(query string) ([][]byte, error) {
	rows, err := db.db.Query(query)
	if err != nil {
		return nil, err
	}

	var ids [][]byte

	for rows.Next() {
		var id []byte
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (db sqlitePersistence) GetNotReadActivityCenterNotificationIds() ([][]byte, error) {
	return db.runActivityCenterIDQuery("SELECT a.id FROM activity_center_notifications a WHERE NOT a.read")
}

func (db sqlitePersistence) GetToProcessActivityCenterNotificationIds() ([][]byte, error) {
	return db.runActivityCenterIDQuery("SELECT a.id FROM activity_center_notifications a WHERE NOT a.dismissed AND NOT a.accepted")
}

func (db sqlitePersistence) HasPendingNotificationsForChat(chatID string) (bool, error) {
	rows, err := db.db.Query("SELECT 1 FROM activity_center_notifications a WHERE a.chat_id = ? AND NOT a.dismissed AND NOT a.accepted", chatID)
	if err != nil {
		return false, err
	}

	result := false

	if rows.Next() {
		result = true
		rows.Close()
	}

	return result, nil
}

func (db sqlitePersistence) GetActivityCenterNotificationsByID(ids []types.HexBytes) ([]*ActivityCenterNotification, error) {
	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	rows, err := db.db.Query("SELECT a.id, a.read, a.accepted, a.dismissed FROM activity_center_notifications a WHERE a.id IN ("+inVector+")", idsArgs...) // nolint: gosec

	if err != nil {
		return nil, err
	}

	var notifications []*ActivityCenterNotification
	for rows.Next() {
		notification := &ActivityCenterNotification{}
		err := rows.Scan(
			&notification.ID,
			&notification.Read,
			&notification.Accepted,
			&notification.Dismissed)

		if err != nil {
			return nil, err
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

func (db sqlitePersistence) GetActivityCenterNotificationByID(id types.HexBytes) (*ActivityCenterNotification, error) {
	row := db.db.QueryRow(`
		SELECT
		a.id,
		a.timestamp,
		a.notification_type,
		a.chat_id,
		a.community_id,
		a.membership_status,
		a.read,
		a.accepted,
		a.dismissed,
		a.message,
		c.last_message,
		a.reply_message,
		a.contact_verification_status,
		c.name,
		a.author
		FROM activity_center_notifications a
		LEFT JOIN chats c
		ON
		c.id = a.chat_id
		WHERE a.id = ?`, id)

	notification, err := db.unmarshalActivityCenterNotificationRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return notification, err
}

func (db sqlitePersistence) UnreadActivityCenterNotifications(cursor string, limit uint64, activityTypes []ActivityCenterType) (string, []*ActivityCenterNotification, error) {
	params := activityCenterQueryParams{
		activityCenterTypes: activityTypes,
		cursor:              cursor,
		limit:               limit,
		read:                ActivityCenterQueryParamsReadUnread,
	}

	return db.activityCenterNotifications(params)
}

func (db sqlitePersistence) ReadActivityCenterNotifications(cursor string, limit uint64, activityTypes []ActivityCenterType) (string, []*ActivityCenterNotification, error) {
	params := activityCenterQueryParams{
		activityCenterTypes: activityTypes,
		cursor:              cursor,
		limit:               limit,
		read:                ActivityCenterQueryParamsReadRead,
	}

	return db.activityCenterNotifications(params)
}

func (db sqlitePersistence) ActivityCenterNotificationsBy(cursor string, limit uint64, activityTypes []ActivityCenterType, readType ActivityCenterQueryParamsRead, accepted bool) (string, []*ActivityCenterNotification, error) {
	params := activityCenterQueryParams{
		activityCenterTypes: activityTypes,
		cursor:              cursor,
		limit:               limit,
		read:                readType,
		accepted:            accepted,
	}

	return db.activityCenterNotifications(params)
}

func (db sqlitePersistence) activityCenterNotifications(params activityCenterQueryParams) (string, []*ActivityCenterNotification, error) {
	var tx *sql.Tx
	var err error
	// We fetch limit + 1 to check for pagination
	nonIncrementedLimit := params.limit
	incrementedLimit := int(params.limit) + 1
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return "", nil, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	params.limit = uint64(incrementedLimit)
	latestCursor, notifications, err := db.buildActivityCenterQuery(tx, params)
	if err != nil {
		return "", nil, err
	}

	if len(notifications) == incrementedLimit {
		notifications = notifications[0:nonIncrementedLimit]
	} else {
		latestCursor = ""
	}

	return latestCursor, notifications, nil
}

func (db sqlitePersistence) ActivityCenterNotifications(currCursor string, limit uint64) (string, []*ActivityCenterNotification, error) {
	params := activityCenterQueryParams{
		cursor: currCursor,
		limit:  limit,
	}
	return db.activityCenterNotifications(params)
}

func (db sqlitePersistence) DismissAllActivityCenterNotifications() error {
	_, err := db.db.Exec(`UPDATE activity_center_notifications SET read = 1, dismissed = 1 WHERE NOT dismissed AND NOT accepted`)
	return err
}

func (db sqlitePersistence) DismissAllActivityCenterNotificationsFromUser(userPublicKey string) error {
	_, err := db.db.Exec(`UPDATE activity_center_notifications SET read = 1, dismissed = 1 WHERE NOT dismissed AND NOT accepted AND author = ?`, userPublicKey)
	return err
}

func (db sqlitePersistence) DismissActivityCenterNotifications(ids []types.HexBytes) error {

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := "UPDATE activity_center_notifications SET read = 1, dismissed = 1 WHERE id IN (" + inVector + ")" // nolint: gosec
	_, err := db.db.Exec(query, idsArgs...)
	return err

}

func (db sqlitePersistence) DismissAllActivityCenterNotificationsFromCommunity(communityID string) error {

	chatIDs, err := db.AllChatIDsByCommunity(communityID)
	if err != nil {
		return err
	}

	chatIDsCount := len(chatIDs)
	if chatIDsCount == 0 {
		return nil
	}

	chatIDsArgs := make([]interface{}, 0, chatIDsCount)
	for _, chatID := range chatIDs {
		chatIDsArgs = append(chatIDsArgs, chatID)
	}

	inVector := strings.Repeat("?, ", chatIDsCount-1) + "?"
	query := "UPDATE activity_center_notifications SET read = 1, dismissed = 1 WHERE chat_id IN (" + inVector + ")" // nolint: gosec
	_, err = db.db.Exec(query, chatIDsArgs...)
	return err

}

func (db sqlitePersistence) DismissAllActivityCenterNotificationsFromChatID(chatID string) error {
	// We exclude notifications related to contacts, since those we don't want to be cleared
	_, err := db.db.Exec(`
UPDATE activity_center_notifications SET read = 1, dismissed = 1
	WHERE
	NOT dismissed
	AND NOT accepted
	AND chat_id = ?
	AND notification_type != ?
	`, chatID, ActivityCenterNotificationTypeContactRequest)
	return err
}

func (db sqlitePersistence) AcceptAllActivityCenterNotifications() ([]*ActivityCenterNotification, error) {
	var tx *sql.Tx
	var err error

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

	_, notifications, err := db.buildActivityCenterQuery(tx, activityCenterQueryParams{})

	_, err = tx.Exec(`UPDATE activity_center_notifications SET read = 1, accepted = 1 WHERE NOT accepted AND NOT dismissed`)
	if err != nil {
		return nil, err
	}
	return notifications, nil
}

func (db sqlitePersistence) AcceptActivityCenterNotifications(ids []types.HexBytes) ([]*ActivityCenterNotification, error) {

	var tx *sql.Tx
	var err error

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

	params := activityCenterQueryParams{
		ids: ids,
	}

	_, notifications, err := db.buildActivityCenterQuery(tx, params)

	if err != nil {
		return nil, err
	}

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := "UPDATE activity_center_notifications SET read = 1, accepted = 1 WHERE id IN (" + inVector + ")" // nolint: gosec
	_, err = tx.Exec(query, idsArgs...)
	return notifications, err
}

func (db sqlitePersistence) AcceptActivityCenterNotificationsForInvitesFromUser(userPublicKey string) ([]*ActivityCenterNotification, error) {
	var tx *sql.Tx
	var err error

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

	params := activityCenterQueryParams{
		author:              userPublicKey,
		activityCenterTypes: []ActivityCenterType{ActivityCenterNotificationTypeNewPrivateGroupChat},
	}

	_, notifications, err := db.buildActivityCenterQuery(tx, params)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`UPDATE activity_center_notifications SET read = 1, accepted = 1 WHERE NOT accepted AND NOT dismissed AND author = ? AND notification_type = ?`, userPublicKey, ActivityCenterNotificationTypeNewPrivateGroupChat)

	if err != nil {
		return nil, err
	}

	return notifications, nil
}

func (db sqlitePersistence) MarkAllActivityCenterNotificationsRead() error {
	_, err := db.db.Exec(`UPDATE activity_center_notifications SET read = 1 WHERE NOT read`)
	return err
}

func (db sqlitePersistence) MarkActivityCenterNotificationsRead(ids []types.HexBytes) error {

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := "UPDATE activity_center_notifications SET read = 1 WHERE id IN (" + inVector + ")" // nolint: gosec
	_, err := db.db.Exec(query, idsArgs...)
	return err

}

func (db sqlitePersistence) MarkActivityCenterNotificationsUnread(ids []types.HexBytes) error {

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := "UPDATE activity_center_notifications SET read = 0 WHERE id IN (" + inVector + ")" // nolint: gosec
	_, err := db.db.Exec(query, idsArgs...)
	return err

}

func (db sqlitePersistence) buildActivityCenterNotificationsCountQuery(isAccepted bool, activityTypes []ActivityCenterType) *sql.Row {
	var args []interface{}
	var acceptedWhere string

	if !isAccepted {
		acceptedWhere = `AND NOT accepted`
	}

	var inTypeWhere string
	if len(activityTypes) != 0 {
		inVector := strings.Repeat("?, ", len(activityTypes)-1) + "?"
		inTypeWhere = fmt.Sprintf(" AND notification_type IN (%s)", inVector)
		for _, activityCenterType := range activityTypes {
			args = append(args, activityCenterType)
		}
	}

	query := fmt.Sprintf(`
		SELECT COUNT(1) 
		FROM activity_center_notifications 
		WHERE NOT read AND NOT dismissed 
		%s 
		%s
	`, acceptedWhere, inTypeWhere)

	return db.db.QueryRow(query, args...)
}

func (db sqlitePersistence) UnreadActivityCenterNotificationsCount() (uint64, error) {
	var count uint64
	err := db.buildActivityCenterNotificationsCountQuery(false, []ActivityCenterType{}).Scan(&count)
	return count, err
}

func (db sqlitePersistence) UnreadAndAcceptedActivityCenterNotificationsCount(activityTypes []ActivityCenterType) (uint64, error) {
	var count uint64
	err := db.buildActivityCenterNotificationsCountQuery(true, activityTypes).Scan(&count)
	return count, err
}

func (db sqlitePersistence) ActiveContactRequestNotification(contactID string) (*ActivityCenterNotification, error) {
	row := db.db.QueryRow(`
		SELECT
		a.id,
		a.timestamp,
		a.notification_type,
		a.chat_id,
		a.community_id,
		a.membership_status,
		a.read,
		a.accepted,
		a.dismissed,
		a.message,
		c.last_message,
		a.reply_message,
		a.contact_verification_status,
		c.name,
		a.author
		FROM activity_center_notifications a
		LEFT JOIN chats c
		ON
		c.id = a.chat_id
		WHERE NOT dismissed AND NOT a.accepted AND notification_type = ? AND author = ?`, ActivityCenterNotificationTypeContactRequest, contactID)
	notification, err := db.unmarshalActivityCenterNotificationRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return notification, err
}

func (db sqlitePersistence) RemoveAllContactRequestActivityCenterNotifications(chatID string) error {
	_, err := db.db.Exec(`
				DELETE FROM activity_center_notifications
	WHERE
	chat_id = ?
	AND notification_type = ?
	`, chatID, ActivityCenterNotificationTypeContactRequest)
	return err
}

func (db sqlitePersistence) HasUnseenActivityCenterNotifications() (bool, error) {
	row := db.db.QueryRow(`SELECT has_seen FROM activity_center_states`)
	hasSeen := true
	err := row.Scan(&hasSeen)
	return !hasSeen, err
}

func (db sqlitePersistence) MarkAsSeenActivityCenterNotifications() error {
	_, err := db.db.Exec(`UPDATE activity_center_states SET has_seen = 1`)
	return err
}

func (db sqlitePersistence) GetActivityCenterState() (*ActivityCenterState, error) {
	unread, err := db.UnreadActivityCenterNotificationsCount()
	if err != nil {
		return nil, err
	}

	unseen, err := db.HasUnseenActivityCenterNotifications()
	if err != nil {
		return nil, err
	}

	state := &ActivityCenterState{
		Unread:  unread,
		HasSeen: !unseen,
	}
	return state, nil
}
