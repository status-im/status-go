package protocol

import (
	"github.com/status-im/status-go/eth-node/types"
)

func (m *Messenger) UnreadActivityCenterNotificationsCount() (uint64, error) {
	return m.persistence.UnreadActivityCenterNotificationsCount()
}

func (m *Messenger) MarkAllActivityCenterNotificationsRead() error {
	return m.persistence.MarkAllActivityCenterNotificationsRead()
}

func (m *Messenger) processAcceptedActivityCenterNotifications(notifications []*ActivityCenterNotification) (*MessengerResponse, error) {
	response := &MessengerResponse{}
	var chats []*Chat
	for _, notification := range notifications {
		if notification.ChatID != "" {
			chat, ok := m.allChats.Load(notification.ChatID)
			if !ok {
				// This should not really happen, but ignore just in case it was deleted in the meantime
				m.logger.Warn("chat not found")
				continue
			}
			chat.Active = true

			chats = append(chats, chat)
			response.AddChat(chat)
		}
	}
	if len(chats) != 0 {
		err := m.saveChats(chats)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

func (m *Messenger) AcceptAllActivityCenterNotifications() (*MessengerResponse, error) {
	notifications, err := m.persistence.AcceptAllActivityCenterNotifications()
	if err != nil {
		return nil, err
	}
	return m.processAcceptedActivityCenterNotifications(notifications)
}

func (m *Messenger) AcceptActivityCenterNotifications(ids []types.HexBytes) (*MessengerResponse, error) {
	notifications, err := m.persistence.AcceptActivityCenterNotifications(ids)
	if err != nil {
		return nil, err
	}
	return m.processAcceptedActivityCenterNotifications(notifications)
}

func (m *Messenger) DismissAllActivityCenterNotifications() error {
	return m.persistence.DismissAllActivityCenterNotifications()
}

func (m *Messenger) DismissActivityCenterNotifications(ids []types.HexBytes) error {
	return m.persistence.DismissActivityCenterNotifications(ids)
}

func (m *Messenger) ActivityCenterNotifications(cursor string, limit uint64) (*ActivityCenterPaginationResponse, error) {
	cursor, notifications, err := m.persistence.ActivityCenterNotifications(cursor, limit)
	if err != nil {
		return nil, err
	}

	return &ActivityCenterPaginationResponse{
		Cursor:        cursor,
		Notifications: notifications,
	}, nil
}
