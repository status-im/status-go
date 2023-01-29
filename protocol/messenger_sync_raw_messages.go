package protocol

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/logutils"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"

	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/signal"

	"github.com/status-im/status-go/protocol/protobuf"
)

type RawMessageHandler func(ctx context.Context, rawMessage common.RawMessage) (common.RawMessage, error)

func (m *Messenger) HandleSyncRawMessages(rawMessages []*protobuf.RawMessage) error {
	state := m.buildMessageState()
	logger := logutils.ZapLogger()
	logger.Info("sync messages processing", zap.Int("count", len(rawMessages)))

	for _, rawMessage := range rawMessages {
		logger.Info("HandleSyncRawMessage", zap.Int("type", int(rawMessage.GetMessageType())))
		switch rawMessage.GetMessageType() {
		case protobuf.ApplicationMetadataMessage_CONTACT_UPDATE:
			var message protobuf.ContactUpdate
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleContactUpdate(state, message)
			if err != nil {
				m.logger.Warn("failed to HandleContactUpdate when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_PUBLIC_CHAT:
			var message protobuf.SyncInstallationPublicChat
			logger.Info("HandleSyncRawMessage SyncInstallationPublicChat")
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			logger.Info("HandleSyncRawMessage SyncInstallationPublicChat Unmarshal", zap.Error(err))
			if err != nil {
				return err
			}
			logger.Info("HandleSyncRawMessage SyncInstallationPublicChat HandleSyncInstallationPublicChat")
			addedChat := m.HandleSyncInstallationPublicChat(state, message)
			if addedChat != nil {
				logger.Info("HandleSyncRawMessage SyncInstallationPublicChat createPublicChat addedChat")
				_, err = m.createPublicChat(addedChat.ID, state.Response)
				logger.Info("done HandleSyncRawMessage SyncInstallationPublicChat createPublicChat addedChat", zap.Error(err))
				if err != nil {
					m.logger.Error("error createPublicChat when HandleSyncRawMessages", zap.Error(err))
					continue
				}
			}
		case protobuf.ApplicationMetadataMessage_SYNC_CHAT_REMOVED:
			var message protobuf.SyncChatRemoved
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncChatRemoved(state, message)
			if err != nil {
				m.logger.Error("failed to HandleSyncChatRemoved when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_CHAT_MESSAGES_READ:
			var message protobuf.SyncChatMessagesRead
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncChatMessagesRead(state, message)
			if err != nil {
				m.logger.Error("failed to HandleSyncChatMessagesRead when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_CLEAR_HISTORY:
			var message protobuf.SyncClearHistory
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncClearHistory(state, message)
			if err != nil {
				m.logger.Error("failed to handleSyncClearHistory when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_CONTACT:
			var message protobuf.SyncInstallationContactV2
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncInstallationContact(state, message)
			if err != nil {
				m.logger.Error("failed to HandleSyncInstallationContact when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_COMMUNITY:
			var message protobuf.SyncCommunity
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncCommunity(state, message)
			if err != nil {
				m.logger.Error("failed to handleSyncCommunity when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_BOOKMARK:
			var message protobuf.SyncBookmark
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncBookmark(state, message)
			if err != nil {
				m.logger.Error("failed to handleSyncBookmark when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_TRUSTED_USER:
			var message protobuf.SyncTrustedUser
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncTrustedUser(state, message)
			if err != nil {
				m.logger.Error("failed to handleSyncTrustedUser when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_VERIFICATION_REQUEST:
			var message protobuf.SyncVerificationRequest
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncVerificationRequest(state, message)
			if err != nil {
				m.logger.Error("failed to handleSyncVerificationRequest when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_SETTING:
			var message protobuf.SyncSetting
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncSetting(state, &message)
			if err != nil {
				m.logger.Error("failed to handleSyncSetting when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_PROFILE_PICTURE:
			var message protobuf.SyncProfilePictures
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncProfilePictures(state, message)
			if err != nil {
				m.logger.Error("failed to HandleSyncProfilePictures when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_CONTACT_REQUEST_DECISION:
			var message protobuf.SyncContactRequestDecision
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncContactRequestDecision(state, message)
			if err != nil {
				m.logger.Error("failed to HandleSyncContactRequestDecision when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_WALLET_ACCOUNT:
			var message protobuf.SyncWalletAccounts
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncWalletAccount(state, message)
			if err != nil {
				m.logger.Error("failed to HandleSyncWalletAccount when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_SAVED_ADDRESS:
			var message protobuf.SyncSavedAddress
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncSavedAddress(state, message)
			if err != nil {
				m.logger.Error("failed to handleSyncSavedAddress when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		}
	}

	response, err := m.saveDataAndPrepareResponse(state)
	logger.Info("saveDataAndPrepareResponse", zap.Error(err))
	if err != nil {
		return err
	}
	publishMessengerResponse(response)
	logger.Info("done publishMessengerResponse")
	return nil
}

// this is a copy implementation of the one in ext/service.go, we should refactor this?
func publishMessengerResponse(response *MessengerResponse) {
	if !response.IsEmpty() {
		logger := logutils.ZapLogger()
		logger.Info("publishing response")
		notifications := response.Notifications()
		logger.Info("publishing response notifications")
		// Clear notifications as not used for now
		response.ClearNotifications()
		logger.Info("publishing response ClearNotifications")
		signal.SendNewMessages(response)
		logger.Info("publishing response SendNewMessages")
		localnotifications.PushMessages(notifications)
		logger.Info("publishing response PushMessages")
	}
}
