package protocol

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
	"go.uber.org/zap"
)

func ToUserStatus(msg protobuf.StatusUpdate) accounts.UserStatus {
	return accounts.UserStatus{
		StatusType: int(msg.StatusType),
		Clock:      msg.Clock,
		CustomText: msg.CustomText,
	}
}

func GetDefaultUserStatus() accounts.UserStatus {
	return accounts.UserStatus{
		StatusType: int(protobuf.StatusUpdate_ONLINE),
		Clock:      0,
		CustomText: "",
	}
}

func (m *Messenger) sendUserStatus(status accounts.UserStatus) error {
	shouldBroadcastUserStatus, err := m.settings.ShouldBroadcastUserStatus()
	if err != nil {
		return err
	}

	if !shouldBroadcastUserStatus {
		m.logger.Debug("user status should not be broadcasted")
		return nil
	}

	status.Clock = uint64(time.Now().Unix())

	err = m.settings.SaveSetting("current-user-status", status)
	if err != nil {
		return err
	}

	statusUpdate := &protobuf.StatusUpdate{
		Clock:      status.Clock,
		StatusType: protobuf.StatusUpdate_StatusType(status.StatusType),
		CustomText: status.CustomText,
	}

	encodedMessage, err := proto.Marshal(statusUpdate)
	if err != nil {
		return err
	}

	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)

	rawMessage := common.RawMessage{
		LocalChatID:         contactCodeTopic,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_STATUS_UPDATE,
		ResendAutomatically: true,
	}

	_, err = m.sender.SendPublic(context.Background(), contactCodeTopic, rawMessage)
	if err != nil {
		return err
	}

	joinedCommunities, err := m.communitiesManager.Joined()
	if err != nil {
		return err
	}
	for _, community := range joinedCommunities {
		rawMessage.LocalChatID = community.StatusUpdatesChannelID()
		_, err = m.sender.SendPublic(context.Background(), rawMessage.LocalChatID, rawMessage)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) sendCurrentUserStatus() {
	currStatus, err := m.settings.GetCurrentStatus()
	if err != nil {
		m.logger.Debug("Error obtaining latest status", zap.Error(err))
		return
	}
	if currStatus == nil {
		defaultStatus := GetDefaultUserStatus()
		currStatus = &defaultStatus
	}

	if err := m.sendUserStatus(*currStatus); err != nil {
		m.logger.Debug("Error when sending the latest user status", zap.Error(err))
	}
}

func (m *Messenger) sendCurrentUserStatusToCommunity(community *communities.Community) error {
	shouldBroadcastUserStatus, err := m.settings.ShouldBroadcastUserStatus()
	if err != nil {
		return err
	}

	if !shouldBroadcastUserStatus {
		m.logger.Debug("user status should not be broadcasted")
		return nil
	}

	status, err := m.settings.GetCurrentStatus()
	if err != nil {
		m.logger.Debug("Error obtaining latest status", zap.Error(err))
		return err
	}

	if status == nil {
		defaultStatus := GetDefaultUserStatus()
		status = &defaultStatus
	}

	status.Clock = uint64(time.Now().Unix())

	err = m.settings.SaveSetting("current-user-status", status)
	if err != nil {
		return err
	}

	statusUpdate := &protobuf.StatusUpdate{
		Clock:      status.Clock,
		StatusType: protobuf.StatusUpdate_StatusType(status.StatusType),
		CustomText: status.CustomText,
	}

	encodedMessage, err := proto.Marshal(statusUpdate)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         community.StatusUpdatesChannelID(),
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_STATUS_UPDATE,
		ResendAutomatically: true,
	}

	_, err = m.sender.SendPublic(context.Background(), rawMessage.LocalChatID, rawMessage)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) broadcastLatestUserStatus() {
	m.logger.Debug("broadcasting user status")
	m.sendCurrentUserStatus()
	go func() {
		for {
			select {
			case <-time.After(5 * time.Minute):
				m.sendCurrentUserStatus()
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) SetUserStatus(newStatus int, newCustomText string) error {
	if len([]rune(newCustomText)) > maxStatusMessageText {
		return fmt.Errorf("custom text shouldn't be longer than %d", maxStatusMessageText)
	}

	if newStatus != int(protobuf.StatusUpdate_ONLINE) && newStatus != int(protobuf.StatusUpdate_DO_NOT_DISTURB) {
		return fmt.Errorf("unknown status type")
	}

	currStatus, err := m.settings.GetCurrentStatus()
	if err != nil {
		return err
	}

	if currStatus == nil {
		c := GetDefaultUserStatus()
		currStatus = &c
	}

	if newStatus == currStatus.StatusType && newCustomText == currStatus.CustomText {
		m.logger.Debug("Status type did not change")
		return nil
	}

	currStatus.StatusType = newStatus
	currStatus.CustomText = newCustomText

	return m.sendUserStatus(*currStatus)
}

func (m *Messenger) HandleStatusUpdate(state *ReceivedMessageState, statusMessage protobuf.StatusUpdate) error {
	if err := ValidateStatusUpdate(statusMessage); err != nil {
		return err
	}

	currentStatus, err := m.settings.GetCurrentStatus()
	if err != nil {
		return err
	}

	if currentStatus == nil {
		c := GetDefaultUserStatus()
		currentStatus = &c
	}

	if common.IsPubKeyEqual(state.CurrentMessageState.PublicKey, &m.identity.PublicKey) { // Status message is ours
		if currentStatus.Clock >= statusMessage.Clock || (currentStatus.StatusType == int(statusMessage.StatusType) && currentStatus.CustomText == statusMessage.CustomText) {
			return nil // older status message, or status does not change ignoring it
		}
		newStatus := ToUserStatus(statusMessage)
		err = m.settings.SaveSetting("current-user-status", newStatus)
		if err != nil {
			return err
		}
		state.Response.SetCurrentStatus(newStatus)
	} else {
		allowed, err := m.isMessageAllowedFrom(state.CurrentMessageState.Contact.ID, nil)
		if err != nil {
			return err
		}

		if !allowed {
			return ErrMessageNotAllowed
		}

		statusUpdate := ToUserStatus(statusMessage)
		statusUpdate.PublicKey = state.CurrentMessageState.Contact.ID
		state.Response.AddStatusUpdate(statusUpdate)
	}
	return nil
}
