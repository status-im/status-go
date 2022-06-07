package protocol

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) UnreadActivityCenterNotificationsCount() (uint64, error) {
	return m.persistence.UnreadActivityCenterNotificationsCount()
}

func toHexBytes(b [][]byte) []types.HexBytes {
	hb := make([]types.HexBytes, len(b))

	for i, v := range b {
		hb[i] = types.HexBytes(v)
	}

	return hb
}

func fromHexBytes(hb []types.HexBytes) [][]byte {
	b := make([][]byte, len(hb))

	for i, v := range hb {
		b[i] = v
	}

	return b
}

func (m *Messenger) MarkAllActivityCenterNotificationsRead(ctx context.Context) error {
	if m.hasPairedDevices() {
		ids, err := m.persistence.GetNotReadActivityCenterNotificationIds()
		if err != nil {
			return err
		}

		_, err = m.MarkActivityCenterNotificationsRead(ctx, toHexBytes(ids), true)
		return err
	}

	return m.persistence.MarkAllActivityCenterNotificationsRead()
}

func (m *Messenger) MarkActivityCenterNotificationsRead(ctx context.Context, ids []types.HexBytes, sync bool) (*MessengerResponse, error) {
	err := m.persistence.MarkActivityCenterNotificationsRead(ids)
	if err != nil {
		return nil, err
	}

	if !sync {
		notifications, err := m.persistence.GetActivityCenterNotificationsByID(ids)
		if err != nil {
			return nil, err
		}
		return m.processActivityCenterNotifications(notifications, true)
	}

	syncMessage := &protobuf.SyncActivityCenterRead{
		Clock: m.getTimesource().GetCurrentTime(),
		Ids:   fromHexBytes(ids),
	}

	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return nil, err
	}

	err = m.sendToPairedDevices(ctx, common.RawMessage{
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ACTIVITY_CENTER_READ,
		ResendAutomatically: true,
	})

	return nil, err
}

func (m *Messenger) MarkActivityCenterNotificationsUnread(ids []types.HexBytes) error {
	return m.persistence.MarkActivityCenterNotificationsUnread(ids)
}

func (m *Messenger) processActivityCenterNotifications(notifications []*ActivityCenterNotification, addNotifications bool) (*MessengerResponse, error) {
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

			if chat.PrivateGroupChat() {
				// Send Joined message for backward compatibility
				_, err := m.ConfirmJoiningGroup(context.Background(), chat.ID)
				if err != nil {
					m.logger.Error("failed to join group", zap.Error(err))
					return nil, err
				}
			}

			chats = append(chats, chat)
			response.AddChat(chat)
		}

		if addNotifications {
			response.AddActivityCenterNotification(notification)
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

func (m *Messenger) processAcceptedActivityCenterNotifications(ctx context.Context, notifications []*ActivityCenterNotification, sync bool) (*MessengerResponse, error) {
	ids := make([][]byte, len(notifications))

	for i := range notifications {
		ids[i] = notifications[i].ID
		notifications[i].Accepted = true
		notifications[i].Read = true
	}

	if sync {
		syncMessage := &protobuf.SyncActivityCenterAccepted{
			Clock: m.getTimesource().GetCurrentTime(),
			Ids:   ids,
		}

		encodedMessage, err := proto.Marshal(syncMessage)
		if err != nil {
			return nil, err
		}

		err = m.sendToPairedDevices(ctx, common.RawMessage{
			Payload:             encodedMessage,
			MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ACTIVITY_CENTER_ACCEPTED,
			ResendAutomatically: true,
		})

		if err != nil {
			return nil, err
		}
	}

	return m.processActivityCenterNotifications(notifications, !sync)
}

func (m *Messenger) AcceptAllActivityCenterNotifications(ctx context.Context) (*MessengerResponse, error) {
	notifications, err := m.persistence.AcceptAllActivityCenterNotifications()
	if err != nil {
		return nil, err
	}

	return m.processAcceptedActivityCenterNotifications(ctx, notifications, true)
}

func (m *Messenger) AcceptActivityCenterNotifications(ctx context.Context, ids []types.HexBytes, sync bool) (*MessengerResponse, error) {

	if len(ids) == 0 {
		return nil, errors.New("notifications ids are not provided")
	}

	notifications, err := m.persistence.AcceptActivityCenterNotifications(ids)
	if err != nil {
		return nil, err
	}

	return m.processAcceptedActivityCenterNotifications(ctx, notifications, sync)
}

func (m *Messenger) DismissAllActivityCenterNotifications(ctx context.Context) error {
	if m.hasPairedDevices() {
		ids, err := m.persistence.GetToProcessActivityCenterNotificationIds()
		if err != nil {
			return err
		}

		_, err = m.DismissActivityCenterNotifications(ctx, toHexBytes(ids), true)
		return err
	}

	return m.persistence.DismissAllActivityCenterNotifications()
}

func (m *Messenger) DismissActivityCenterNotifications(ctx context.Context, ids []types.HexBytes, sync bool) (*MessengerResponse, error) {
	err := m.persistence.DismissActivityCenterNotifications(ids)
	if err != nil {
		return nil, err
	}

	if !sync {
		notifications, err := m.persistence.GetActivityCenterNotificationsByID(ids)
		if err != nil {
			return nil, err
		}

		return m.processActivityCenterNotifications(notifications, true)
	}

	syncMessage := &protobuf.SyncActivityCenterDismissed{
		Clock: m.getTimesource().GetCurrentTime(),
		Ids:   fromHexBytes(ids),
	}

	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return nil, err
	}

	err = m.sendToPairedDevices(ctx, common.RawMessage{
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ACTIVITY_CENTER_DISMISSED,
		ResendAutomatically: true,
	})

	return nil, err
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

func (m *Messenger) handleActivityCenterRead(state *ReceivedMessageState, message protobuf.SyncActivityCenterRead) error {
	resp, err := m.MarkActivityCenterNotificationsRead(context.TODO(), toHexBytes(message.Ids), false)

	if err != nil {
		return err
	}

	return state.Response.Merge(resp)
}

func (m *Messenger) handleActivityCenterAccepted(state *ReceivedMessageState, message protobuf.SyncActivityCenterAccepted) error {
	resp, err := m.AcceptActivityCenterNotifications(context.TODO(), toHexBytes(message.Ids), false)

	if err != nil {
		return err
	}

	return state.Response.Merge(resp)
}

func (m *Messenger) handleActivityCenterDismissed(state *ReceivedMessageState, message protobuf.SyncActivityCenterDismissed) error {
	resp, err := m.DismissActivityCenterNotifications(context.TODO(), toHexBytes(message.Ids), false)

	if err != nil {
		return err
	}

	return state.Response.Merge(resp)
}
