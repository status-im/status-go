package protocol

import (
	"context"
	"encoding/json"

	"github.com/status-im/status-go/protocol/verification"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

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

func (m *Messenger) ActivityCenterNotifications(request ActivityCenterNotificationsRequest) (*ActivityCenterPaginationResponse, error) {
	cursor, notifications, err := m.persistence.ActivityCenterNotifications(request.Cursor, request.Limit, request.ActivityTypes, request.ReadType, true)
	if err != nil {
		return nil, err
	}

	return &ActivityCenterPaginationResponse{
		Cursor:        cursor,
		Notifications: notifications,
	}, nil
}

func (m *Messenger) ActivityCenterNotificationsCount(request ActivityCenterCountRequest) (*ActivityCenterCountResponse, error) {
	response := make(ActivityCenterCountResponse)

	for _, activityType := range request.ActivityTypes {
		count, err := m.persistence.ActivityCenterNotificationsCount([]ActivityCenterType{activityType}, request.ReadType, true)
		if err != nil {
			return nil, err
		}

		response[activityType] = count
	}

	return &response, nil
}

func (m *Messenger) HasUnseenActivityCenterNotifications() (bool, error) {
	seen, _, err := m.persistence.HasUnseenActivityCenterNotifications()
	return seen, err
}

func (m *Messenger) GetActivityCenterState() (*ActivityCenterState, error) {
	return m.persistence.GetActivityCenterState()
}

func (m *Messenger) syncActivityCenterNotificationState(state *ActivityCenterState) error {
	if state == nil {
		return nil
	}
	syncStateMessage := &protobuf.SyncActivityCenterNotificationState{
		UpdatedAt: state.UpdatedAt,
		HasSeen:   state.HasSeen,
	}
	encodedMessage, err := proto.Marshal(syncStateMessage)
	if err != nil {
		return err
	}
	return m.sendToPairedDevices(context.TODO(), common.RawMessage{
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ACTIVITY_CENTER_NOTIFICATION_STATE,
		ResendAutomatically: true,
	})
}

func (m *Messenger) syncActivityCenterNotifications(notifications []*ActivityCenterNotification) (err error) {
	if notifications == nil {
		return nil
	}
	var s []*protobuf.SyncActivityCenterNotification
	for _, n := range notifications {
		if n == nil {
			return errors.New("SyncActivityCenterNotifications with nil notification")
		}
		var p *protobuf.SyncActivityCenterNotification
		p, err = convertActivityCenterNotificationToProtobuf(n)
		if err != nil {
			return
		}
		s = append(s, p)
	}
	var encodedMessage []byte
	encodedMessage, err = proto.Marshal(&protobuf.SyncActivityCenterNotifications{
		ActivityCenterNotifications: s,
	})
	if err != nil {
		return
	}
	return m.sendToPairedDevices(context.TODO(), common.RawMessage{
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ACTIVITY_CENTER_NOTIFICATION,
		ResendAutomatically: true,
	})
}

func (m *Messenger) MarkAsSeenActivityCenterNotifications() (*MessengerResponse, error) {
	response := &MessengerResponse{}
	s := &ActivityCenterState{
		UpdatedAt: m.getCurrentTimeInMillis(),
		HasSeen:   true,
	}
	n, err := m.persistence.UpdateActivityCenterNotificationState(s)
	if err != nil {
		return nil, err
	}

	state, err := m.persistence.GetActivityCenterState()
	if err != nil {
		return nil, err
	}

	response.SetActivityCenterState(state)
	if n > 0 {
		return response, m.syncActivityCenterNotificationState(state)
	}
	return response, nil
}

func (m *Messenger) MarkAllActivityCenterNotificationsRead(ctx context.Context) (*MessengerResponse, error) {
	response := &MessengerResponse{}
	updateAt := m.getCurrentTimeInMillis()
	if m.hasPairedDevices() {
		ids, err := m.persistence.GetNotReadActivityCenterNotificationIds()
		if err != nil {
			return nil, err
		}

		_, err = m.MarkActivityCenterNotificationsRead(ctx, toHexBytes(ids), updateAt, true)
		return nil, err
	}

	err := m.persistence.MarkAllActivityCenterNotificationsRead(updateAt)
	if err != nil {
		return nil, err
	}

	state, err := m.persistence.GetActivityCenterState()
	if err != nil {
		return nil, err
	}

	response.SetActivityCenterState(state)
	return response, nil
}

func (m *Messenger) MarkActivityCenterNotificationsRead(ctx context.Context, ids []types.HexBytes, updatedAt uint64, sync bool) (*MessengerResponse, error) {
	response := &MessengerResponse{}
	if updatedAt == 0 {
		updatedAt = m.getCurrentTimeInMillis()
	}
	err := m.persistence.MarkActivityCenterNotificationsRead(ids, updatedAt)
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
		Clock: updatedAt,
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
	if err != nil {
		return nil, err
	}

	state, err := m.persistence.GetActivityCenterState()
	if err != nil {
		return nil, err
	}

	response.SetActivityCenterState(state)
	return response, nil
}

func (m *Messenger) MarkActivityCenterNotificationsUnread(ids []types.HexBytes) (*MessengerResponse, error) {
	response := &MessengerResponse{}
	notifications, err := m.persistence.MarkActivityCenterNotificationsUnread(ids, m.getCurrentTimeInMillis())
	if err != nil {
		return nil, err
	}
	err = m.syncActivityCenterNotifications(notifications)
	if err != nil {
		m.logger.Error("MarkActivityCenterNotificationsUnread, failed to sync activity center notifications", zap.Error(err))
		return nil, err
	}

	state, err := m.persistence.GetActivityCenterState()
	if err != nil {
		return nil, err
	}

	response.SetActivityCenterState(state)
	return response, nil
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
	}

	if sync {
		syncMessage := &protobuf.SyncActivityCenterAccepted{
			Clock: m.getCurrentTimeInMillis(),
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

func (m *Messenger) AcceptActivityCenterNotifications(ctx context.Context, ids []types.HexBytes, updatedAt uint64, sync bool) (*MessengerResponse, error) {
	if len(ids) == 0 {
		return nil, errors.New("notifications ids are not provided")
	}

	if updatedAt == 0 {
		updatedAt = m.getCurrentTimeInMillis()
	}

	notifications, err := m.persistence.AcceptActivityCenterNotifications(ids, updatedAt)
	if err != nil {
		return nil, err
	}

	return m.processAcceptedActivityCenterNotifications(ctx, notifications, sync)
}

func (m *Messenger) DismissActivityCenterNotifications(ctx context.Context, ids []types.HexBytes, updatedAt uint64, sync bool) (*MessengerResponse, error) {
	if updatedAt == 0 {
		updatedAt = m.getCurrentTimeInMillis()
	}
	err := m.persistence.DismissActivityCenterNotifications(ids, updatedAt)
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
		Clock: updatedAt,
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

func (m *Messenger) DeleteActivityCenterNotifications(ctx context.Context, ids []types.HexBytes, sync bool) error {
	notifications, err := m.persistence.DeleteActivityCenterNotifications(ids, m.getCurrentTimeInMillis())
	if err != nil {
		return err
	}
	err = m.syncActivityCenterNotifications(notifications)
	if err != nil {
		m.logger.Error("DeleteActivityCenterNotifications, failed to sync activity center notifications", zap.Error(err))
	}
	return err
}

func (m *Messenger) ActivityCenterNotification(id types.HexBytes) (*ActivityCenterNotification, error) {
	return m.persistence.GetActivityCenterNotificationByID(id)
}

func (m *Messenger) handleActivityCenterRead(state *ReceivedMessageState, message protobuf.SyncActivityCenterRead) error {
	resp, err := m.MarkActivityCenterNotificationsRead(context.TODO(), toHexBytes(message.Ids), message.Clock, false)

	if err != nil {
		return err
	}

	return state.Response.Merge(resp)
}

func (m *Messenger) handleActivityCenterAccepted(state *ReceivedMessageState, message protobuf.SyncActivityCenterAccepted) error {
	resp, err := m.AcceptActivityCenterNotifications(context.TODO(), toHexBytes(message.Ids), message.Clock, false)

	if err != nil {
		return err
	}

	return state.Response.Merge(resp)
}

func (m *Messenger) handleActivityCenterDismissed(state *ReceivedMessageState, message protobuf.SyncActivityCenterDismissed) error {
	resp, err := m.DismissActivityCenterNotifications(context.TODO(), toHexBytes(message.Ids), message.Clock, false)

	if err != nil {
		return err
	}

	return state.Response.Merge(resp)
}

func (m *Messenger) handleSyncActivityCenterNotificationState(state *ReceivedMessageState, a *protobuf.SyncActivityCenterNotificationState) error {
	s := &ActivityCenterState{
		HasSeen:   a.HasSeen,
		UpdatedAt: a.UpdatedAt,
	}
	_, err := m.persistence.UpdateActivityCenterNotificationState(s)
	if err != nil {
		return err
	}
	state.Response.SetActivityCenterState(s)
	return nil
}

func (m *Messenger) handleSyncActivityCenterNotifications(state *ReceivedMessageState, a *protobuf.SyncActivityCenterNotifications) error {
	var notifications []*ActivityCenterNotification
	for _, n := range a.ActivityCenterNotifications {
		notification, err := convertActivityCenterNotificationFromProtobuf(n)
		if err != nil {
			return err
		}
		err = m.persistence.SaveActivityCenterNotification(notification, false)
		if err != nil {
			return err
		}
		notifications = append(notifications, notification)
	}
	response, err := m.processActivityCenterNotifications(notifications, true)
	if err != nil {
		return err
	}
	return state.Response.Merge(response)
}

func convertActivityCenterNotificationToProtobuf(n *ActivityCenterNotification) (p *protobuf.SyncActivityCenterNotification, err error) {
	if n == nil {
		return nil, errors.New("convertActivityCenterNotificationToProtobuf, n is nil")
	}

	var (
		message      []byte
		replyMessage []byte
	)
	if n.Message != nil {
		message, err = json.Marshal(n.Message)
		if err != nil {
			return
		}
	}
	if n.ReplyMessage != nil {
		replyMessage, err = json.Marshal(n.ReplyMessage)
		if err != nil {
			return
		}
	}
	p = &protobuf.SyncActivityCenterNotification{
		Id:                        n.ID,
		Timestamp:                 n.Timestamp,
		NotificationType:          protobuf.SyncActivityCenterNotification_NotificationType(n.Type),
		ChatId:                    n.ChatID,
		Read:                      n.Read,
		Dismissed:                 n.Dismissed,
		Accepted:                  n.Accepted,
		Message:                   message,
		Author:                    n.Author,
		ReplyMessage:              replyMessage,
		CommunityId:               n.CommunityID,
		MembershipStatus:          protobuf.SyncActivityCenterNotification_MembershipStatus(n.MembershipStatus),
		ContactVerificationStatus: protobuf.SyncActivityCenterNotification_ContactVerificationStatus(n.ContactVerificationStatus),
		Deleted:                   n.Deleted,
		UpdatedAt:                 n.UpdatedAt,
	}
	return
}

func convertActivityCenterNotificationFromProtobuf(proto *protobuf.SyncActivityCenterNotification) (*ActivityCenterNotification, error) {
	if proto == nil {
		return nil, errors.New("convertActivityCenterNotificationFromProtobuf, proto is nil")
	}

	a := &ActivityCenterNotification{
		ID:                        proto.Id,
		ChatID:                    proto.ChatId,
		CommunityID:               proto.CommunityId,
		MembershipStatus:          ActivityCenterMembershipStatus(proto.MembershipStatus),
		Author:                    proto.Author,
		Type:                      ActivityCenterType(proto.NotificationType),
		Timestamp:                 proto.Timestamp,
		Read:                      proto.Read,
		Accepted:                  proto.Accepted,
		Dismissed:                 proto.Dismissed,
		Deleted:                   proto.Deleted,
		ContactVerificationStatus: verification.RequestStatus(proto.ContactVerificationStatus),
		UpdatedAt:                 proto.UpdatedAt,
	}

	if len(proto.Message) > 0 {
		message := &common.Message{}
		err := json.Unmarshal(proto.Message, &message)
		if err != nil {
			return nil, err
		}
		a.Message = message
	}
	if len(proto.ReplyMessage) > 0 {
		replyMessage := &common.Message{}
		err := json.Unmarshal(proto.ReplyMessage, &replyMessage)
		if err != nil {
			return nil, err
		}
		a.ReplyMessage = replyMessage
	}

	return a, nil
}
