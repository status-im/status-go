package protocol

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
)

func (db sqlitePersistence) DismissActivityCenterNotification(notificationID types.HexBytes) error {
	return nil
}

func (db sqlitePersistence) MarkActivityCenterNotificationsRead() error {
	return nil
}

func (db sqlitePersistence) SaveActivityCenterNotification(notification *ActivityCenterNotification) error {
	err := notification.Valid()
	if err != nil {
		return err
	}
	_, err = db.db.Exec(`INSERT INTO activity_center_notifications (id,timestamp,notification_type, chat_id) VALUES (?,?,?,?)`, notification.ID, notification.Timestamp, notification.Type, notification.ChatID)
	return err
}

func (db sqlitePersistence) ActivityCenterNotifications(currCursor string, limit uint64) (string, []*ActivityCenterNotification, error) {
	// We fetch limit + 1 to check for pagination
	incrementedLimit := int(limit) + 1
	cursorWhere := ""
	if currCursor != "" {
		cursorWhere = "AND cursor <= ?" //nolint: goconst
	}

	query := fmt.Sprintf(`
  SELECT
  a.id,
  a.timestamp,
  a.notification_type,
  a.chat_id,
  a.read,
  a.accepted,
  a.dismissed,
  c.last_message,
  substr('0000000000000000000000000000000000000000000000000000000000000000' || a.timestamp, -64, 64) || a.id as cursor
  FROM activity_center_notifications a
  LEFT JOIN chats c
  ON
  c.id = a.chat_id
  WHERE NOT a.dismissed AND NOT a.accepted
  %s
  ORDER BY cursor DESC
  LIMIT ?`, cursorWhere)
	args := []interface{}{}
	if currCursor != "" {
		args = append(args, currCursor)
	}
	args = append(args, incrementedLimit)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return "", nil, err
	}

	latestCursor := ""

	var notifications []*ActivityCenterNotification
	for rows.Next() {
		var chatID sql.NullString
		var lastMessageBytes []byte
		notification := &ActivityCenterNotification{}
		err := rows.Scan(
			&notification.ID,
			&notification.Timestamp,
			&notification.Type,
			&chatID,
			&notification.Read,
			&notification.Accepted,
			&notification.Dismissed,
			&lastMessageBytes,
			&latestCursor)
		if err != nil {
			return "", nil, err
		}
		if chatID.Valid {
			notification.ChatID = chatID.String

		}

		// Restore last message
		if lastMessageBytes != nil {
			message := &common.Message{}
			if err = json.Unmarshal(lastMessageBytes, message); err != nil {
				return "", nil, err
			}
			notification.LastMessage = message
		}

		notifications = append(notifications, notification)
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

func (db sqlitePersistence) AcceptAllActivityCenterNotifications() error {
	_, err := db.db.Exec(`UPDATE activity_center_notifications SET accepted = 1 WHERE NOT accepted AND NOT dismissed`)
	return err
}

func (db sqlitePersistence) AcceptActivityCenterNotifications(ids []types.HexBytes) error {

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := "UPDATE activity_center_notifications SET accepted = 1 WHERE id IN (" + inVector + ")" // nolint: gosec
	_, err := db.db.Exec(query, idsArgs...)
	return err
}

func (db sqlitePersistence) MarkAllActivityCenterNotificationsRead() error {
	_, err := db.db.Exec(`UPDATE activity_center_notifications SET read = 1 WHERE NOT read`)
	return err
}

func (db sqlitePersistence) UnreadActivityCenterNotificationsCount() (uint64, error) {
	var count uint64
	err := db.db.QueryRow(`SELECT COUNT(1) FROM activity_center_notifications WHERE NOT read`).Scan(&count)
	return count, err
}
