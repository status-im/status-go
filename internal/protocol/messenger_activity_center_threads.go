package protocol

import (
	"strings"

	"go.uber.org/zap"
)

// populateActivityCenterThreadNames resolves response-only thread names for mentions and replies.
// One bounded lookup per page; no schema changes, persisted names, or shared cache.
func (m *Messenger) populateActivityCenterThreadNames(notifications []*ActivityCenterNotification) {
	if !m.featureFlags.Threads {
		return
	}
	if err := m.persistence.populateActivityCenterThreadNames(notifications); err != nil {
		m.logger.Debug("could not resolve notification threads", zap.Error(err))
	}
}

func (db sqlitePersistence) populateActivityCenterThreadNames(notifications []*ActivityCenterNotification) error {
	targets := make(map[string][]*ActivityCenterNotification)
	var placeholders []string
	var args []interface{}
	for _, notification := range notifications {
		if notification == nil || notification.Message == nil ||
			(notification.Type != ActivityCenterNotificationTypeMention && notification.Type != ActivityCenterNotificationTypeReply) {
			continue
		}
		threadID := notification.Message.GetThreadId()
		if threadID == "" {
			continue
		}
		key := threadKey(notification.ChatID, threadID)
		if _, found := targets[key]; !found {
			placeholders = append(placeholders, "(?, ?)")
			args = append(args, notification.ChatID, threadID)
		}
		targets[key] = append(targets[key], notification)
	}
	if len(args) == 0 {
		return nil
	}
	// #nosec G202 -- Only fixed placeholders are concatenated; chat and thread IDs are bound parameters.
	rows, err := db.db.Query(`SELECT chat_id, thread_id, name FROM threads WHERE (chat_id, thread_id) IN (`+
		strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var chatID, threadID, name string
		if err := rows.Scan(&chatID, &threadID, &name); err != nil {
			return err
		}
		for _, notification := range targets[threadKey(chatID, threadID)] {
			notification.ThreadName = name
		}
	}
	return rows.Err()
}
