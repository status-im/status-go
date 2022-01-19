package protocol

import (
	"context"
	"fmt"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
)

func (m *Messenger) GetCurrentUserStatus() (*UserStatus, error) {

	status := &UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      0,
		CustomText: "",
	}

	err := m.settings.GetCurrentStatus(status)
	if err != nil {
		m.logger.Debug("Error obtaining latest status", zap.Error(err))
		return nil, err
	}

	return status, nil
}

func (m *Messenger) sendUserStatus(ctx context.Context, status UserStatus) error {
	shouldBroadcastUserStatus, err := m.settings.ShouldBroadcastUserStatus()
	if err != nil {
		return err
	}

	if !shouldBroadcastUserStatus {
		m.logger.Debug("user status should not be broadcasted")
		return nil
	}

	status.Clock = uint64(time.Now().Unix())

	err = m.settings.SaveSetting(accounts.CurrentUserStatus.GetReactName(), status)
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

	_, err = m.sender.SendPublic(ctx, contactCodeTopic, rawMessage)
	if err != nil {
		return err
	}

	joinedCommunities, err := m.communitiesManager.Joined()
	if err != nil {
		return err
	}
	for _, community := range joinedCommunities {
		rawMessage.LocalChatID = community.StatusUpdatesChannelID()
		_, err = m.sender.SendPublic(ctx, rawMessage.LocalChatID, rawMessage)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) sendCurrentUserStatus(ctx context.Context) {
	err := m.persistence.CleanOlderStatusUpdates()
	if err != nil {
		m.logger.Debug("Error cleaning status updates", zap.Error(err))
		return
	}

	shouldBroadcastUserStatus, err := m.settings.ShouldBroadcastUserStatus()
	if err != nil {
		m.logger.Debug("Error while getting status broadcast setting", zap.Error(err))
		return
	}

	if !shouldBroadcastUserStatus {
		m.logger.Debug("user status should not be broadcasted")
		return
	}

	currStatus, err := m.GetCurrentUserStatus()
	if err != nil {
		m.logger.Debug("Error obtaining latest status", zap.Error(err))
		return
	}

	if err := m.sendUserStatus(ctx, *currStatus); err != nil {
		m.logger.Debug("Error when sending the latest user status", zap.Error(err))
	}
}

func (m *Messenger) sendCurrentUserStatusToCommunity(ctx context.Context, community *communities.Community) error {
	logger := m.logger.Named("sendCurrentUserStatusToCommunity")

	shouldBroadcastUserStatus, err := m.settings.ShouldBroadcastUserStatus()
	if err != nil {
		logger.Debug("m.settings.ShouldBroadcastUserStatus error", zap.Error(err))
		return err
	}

	if !shouldBroadcastUserStatus {
		logger.Debug("user status should not be broadcasted")
		return nil
	}

	status, err := m.GetCurrentUserStatus()
	if err != nil {
		logger.Debug("Error obtaining latest status", zap.Error(err))
		return err
	}

	status.Clock = uint64(time.Now().Unix())

	err = m.settings.SaveSetting(accounts.CurrentUserStatus.GetReactName(), status)
	if err != nil {
		logger.Debug("m.settings.SaveSetting error",
			zap.Any("current-user-status", status),
			zap.Error(err))
		return err
	}

	statusUpdate := &protobuf.StatusUpdate{
		Clock:      status.Clock,
		StatusType: protobuf.StatusUpdate_StatusType(status.StatusType),
		CustomText: status.CustomText,
	}

	encodedMessage, err := proto.Marshal(statusUpdate)
	if err != nil {
		logger.Debug("proto.Marshal error",
			zap.Any("protobuf.StatusUpdate", statusUpdate),
			zap.Error(err))
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         community.StatusUpdatesChannelID(),
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_STATUS_UPDATE,
		ResendAutomatically: true,
	}

	_, err = m.sender.SendPublic(ctx, rawMessage.LocalChatID, rawMessage)
	if err != nil {
		logger.Debug("m.sender.SendPublic error", zap.Error(err))
		return err
	}

	return nil
}

func (m *Messenger) broadcastLatestUserStatus() {
	m.logger.Debug("broadcasting user status")
	ctx := context.Background()
	go func() {
		// Ensure that we are connected before sending a message
		time.Sleep(5 * time.Second)
		m.sendCurrentUserStatus(ctx)
	}()

	go func() {
		for {
			select {
			case <-time.After(5 * time.Minute):
				m.sendCurrentUserStatus(ctx)
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) SetUserStatus(ctx context.Context, newStatus int, newCustomText string) error {
	if len([]rune(newCustomText)) > maxStatusMessageText {
		return fmt.Errorf("custom text shouldn't be longer than %d", maxStatusMessageText)
	}

	if newStatus != int(protobuf.StatusUpdate_AUTOMATIC) &&
		newStatus != int(protobuf.StatusUpdate_DO_NOT_DISTURB) &&
		newStatus != int(protobuf.StatusUpdate_ALWAYS_ONLINE) &&
		newStatus != int(protobuf.StatusUpdate_INACTIVE) {
		return fmt.Errorf("unknown status type")
	}

	currStatus, err := m.GetCurrentUserStatus()
	if err != nil {
		m.logger.Debug("Error obtaining latest status", zap.Error(err))
		return err
	}

	if newStatus == currStatus.StatusType && newCustomText == currStatus.CustomText {
		m.logger.Debug("Status type did not change")
		return nil
	}

	currStatus.StatusType = newStatus
	currStatus.CustomText = newCustomText

	return m.sendUserStatus(ctx, *currStatus)
}

func (m *Messenger) HandleStatusUpdate(state *ReceivedMessageState, statusMessage protobuf.StatusUpdate) error {
	if err := ValidateStatusUpdate(statusMessage); err != nil {
		return err
	}

	if common.IsPubKeyEqual(state.CurrentMessageState.PublicKey, &m.identity.PublicKey) { // Status message is ours
		currentStatus, err := m.GetCurrentUserStatus()
		if err != nil {
			m.logger.Debug("Error obtaining latest status", zap.Error(err))
			return err
		}

		if currentStatus.Clock >= statusMessage.Clock {
			return nil // older status message, or status does not change ignoring it
		}
		newStatus := ToUserStatus(statusMessage)
		err = m.settings.SaveSetting(accounts.CurrentUserStatus.GetReactName(), newStatus)
		if err != nil {
			return err
		}
		state.Response.SetCurrentStatus(newStatus)
	} else {
		statusUpdate := ToUserStatus(statusMessage)
		statusUpdate.PublicKey = state.CurrentMessageState.Contact.ID

		err := m.persistence.InsertStatusUpdate(statusUpdate)
		if err != nil {
			return err
		}
		state.Response.AddStatusUpdate(statusUpdate)
	}

	return nil
}

func (m *Messenger) StatusUpdates() ([]UserStatus, error) {
	return m.persistence.StatusUpdates()
}
