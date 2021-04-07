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

func (m *Messenger) AcceptAllActivityCenterNotifications() error {
	return m.persistence.AcceptAllActivityCenterNotifications()
}

func (m *Messenger) AcceptActivityCenterNotifications(ids []types.HexBytes) error {
	return m.persistence.AcceptActivityCenterNotifications(ids)
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
