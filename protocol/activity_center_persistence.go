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

	if notification.Type == ActivityCenterNotificationTypeNewOneToOne {
		// Delete other notifications so it pop us again if not currently dismissed
		_, err = tx.Exec(`DELETE FROM activity_center_notifications WHERE id = ? AND dismissed`, notification.ID)
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

	_, err = tx.Exec(`INSERT INTO activity_center_notifications (id, timestamp, notification_type, chat_id, message, author) VALUES (?,?,?,?,?,?)`, notification.ID, notification.Timestamp, notification.Type, notification.ChatID, encodedMessage, notification.Author)
	return err
}

func (db sqlitePersistence) unmarshalActivityCenterNotificationRows(rows *sql.Rows) (string, []*ActivityCenterNotification, error) {
	var notifications []*ActivityCenterNotification
	latestCursor := ""
	for rows.Next() {
		var chatID sql.NullString
		var lastMessageBytes []byte
		var messageBytes []byte
		var name sql.NullString
		var author sql.NullString
		notification := &ActivityCenterNotification{}
		err := rows.Scan(
			&notification.ID,
			&notification.Timestamp,
			&notification.Type,
			&chatID,
			&notification.Read,
			&notification.Accepted,
			&notification.Dismissed,
			&messageBytes,
			&lastMessageBytes,
			&name,
			&author,
			&latestCursor)
		if err != nil {
			return "", nil, err
		}
		if chatID.Valid {
			notification.ChatID = chatID.String

		}

		if name.Valid {
			notification.Name = name.String
		}

		if author.Valid {
			notification.Author = name.String
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

		notifications = append(notifications, notification)
	}

	return latestCursor, notifications, nil

}
func (db sqlitePersistence) buildActivityCenterQuery(tx *sql.Tx, cursor string, limit int, ids []types.HexBytes) (string, []*ActivityCenterNotification, error) {
	var args []interface{}
	cursorWhere := ""
	inQueryWhere := ""
	if cursor != "" {
		cursorWhere = "AND cursor <= ?" //nolint: goconst
		args = append(args, cursor)
	}

	if len(ids) != 0 {

		inVector := strings.Repeat("?, ", len(ids)-1) + "?"
		inQueryWhere = fmt.Sprintf(" AND a.id IN (%s)", inVector)
		for _, id := range ids {
			args = append(args, id)
		}

	}

	query := fmt.Sprintf( // nolint: gosec
		`
  SELECT
  a.id,
  a.timestamp,
  a.notification_type,
  a.chat_id,
  a.read,
  a.accepted,
  a.dismissed,
  a.message,
  c.last_message,
  c.name,
  a.author,
  substr('0000000000000000000000000000000000000000000000000000000000000000' || a.timestamp, -64, 64) || a.id as cursor
  FROM activity_center_notifications a
  LEFT JOIN chats c
  ON
  c.id = a.chat_id
  WHERE NOT a.dismissed AND NOT a.accepted
  %s
  %s
  ORDER BY cursor DESC`, cursorWhere, inQueryWhere)
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

func (db sqlitePersistence) ActivityCenterNotifications(currCursor string, limit uint64) (string, []*ActivityCenterNotification, error) {
	var tx *sql.Tx
	var err error
	// We fetch limit + 1 to check for pagination
	incrementedLimit := int(limit) + 1
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

	latestCursor, notifications, err := db.buildActivityCenterQuery(tx, currCursor, incrementedLimit, nil)
	if err != nil {
		return "", nil, err
	}

	if len(notifications) == incrementedLimit {
		notifications = notifications[0:limit]
	} else {
		latestCursor = ""
	}

	return latestCursor, notifications, nil
}

func (db sqlitePersistence) DismissAllActivityCenterNotifications() error {
	_, err := db.db.Exec(`UPDATE activity_center_notifications SET dismissed = 1 WHERE NOT dismissed AND NOT accepted`)
	return err
}

func (db sqlitePersistence) DismissActivityCenterNotifications(ids []types.HexBytes) error {

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := "UPDATE activity_center_notifications SET dismissed = 1 WHERE id IN (" + inVector + ")" // nolint: gosec
	_, err := db.db.Exec(query, idsArgs...)
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

	_, notifications, err := db.buildActivityCenterQuery(tx, "", 0, nil)

	_, err = tx.Exec(`UPDATE activity_center_notifications SET accepted = 1 WHERE NOT accepted AND NOT dismissed`)
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

	_, notifications, err := db.buildActivityCenterQuery(tx, "", 0, ids)

	if err != nil {
		return nil, err
	}

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := "UPDATE activity_center_notifications SET accepted = 1 WHERE id IN (" + inVector + ")" // nolint: gosec
	_, err = tx.Exec(query, idsArgs...)
	return notifications, err
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

func (db sqlitePersistence) UnreadActivityCenterNotificationsCount() (uint64, error) {
	var count uint64
	err := db.db.QueryRow(`SELECT COUNT(1) FROM activity_center_notifications WHERE NOT read`).Scan(&count)
	return count, err
}
