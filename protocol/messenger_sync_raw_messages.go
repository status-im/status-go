package protocol

import (
	"context"
	"errors"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/signal"
)

type RawMessageHandler func(ctx context.Context, rawMessage common.RawMessage) (common.RawMessage, error)

func (m *Messenger) HandleSyncRawMessages(rawMessages []*protobuf.RawMessage) error {
	state := m.buildMessageState()
	for _, rawMessage := range rawMessages {
		switch rawMessage.GetMessageType() {
		case protobuf.ApplicationMetadataMessage_CONTACT_UPDATE:
			var message protobuf.ContactUpdate
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			publicKey, err := common.HexToPubkey(message.PublicKey)
			if err != nil {
				return err
			}
			var contact *Contact
			if c, ok := state.AllContacts.Load(message.PublicKey); ok {
				contact = c
			} else {
				c, err := buildContact(message.PublicKey, publicKey)
				if err != nil {
					m.logger.Info("failed to build contact", zap.Error(err))
					continue
				}
				contact = c
				state.AllContacts.Store(message.PublicKey, contact)
			}
			currentMessageState := &CurrentMessageState{
				Message: protobuf.ChatMessage{
					Clock: message.Clock,
				},
				MessageID:        " ", // make it not empty to bypass this validation: https://github.com/status-im/status-go/blob/7cd7430d3141b08f7c455d7918f4160ea8fd0559/protocol/messenger_handler.go#L325
				WhisperTimestamp: message.Clock,
				PublicKey:        publicKey,
				Contact:          contact,
			}
			state.CurrentMessageState = currentMessageState
			err = m.HandleContactUpdate(state, message)
			if err != nil {
				m.logger.Warn("failed to HandleContactUpdate when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_PUBLIC_CHAT:
			var message protobuf.SyncInstallationPublicChat
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			addedChat := m.HandleSyncInstallationPublicChat(state, message)
			if addedChat != nil {
				_, err = m.createPublicChat(addedChat.ID, state.Response)
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
			err = m.HandleSyncWalletAccount(state, message, true)
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
		case protobuf.ApplicationMetadataMessage_SYNC_ALL_KEYCARDS:
			var message protobuf.SyncAllKeycards
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncKeycards(state, message)
			if err != nil {
				m.logger.Error("failed to handleSyncKeycards when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_SOCIAL_LINK_SETTING:
			var message protobuf.SyncSocialLinkSetting
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncSocialLinkSetting(state, message)
			if err != nil {
				m.logger.Error("failed to HandleSyncSocialLinkSetting when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_ENS_USERNAME_DETAIL:
			var message protobuf.SyncEnsUsernameDetail
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncEnsUsernameDetail(state, message)
			if err != nil {
				m.logger.Error("failed to handleSyncEnsUsernameDetail when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_PAIR_INSTALLATION:
			var message protobuf.PairInstallation
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			identity := m.myHexIdentity()
			installations := []*multidevice.Installation{
				{
					Identity:  identity,
					ID:        message.InstallationId,
					Version:   message.Version,
					Enabled:   true,
					Timestamp: int64(message.Clock),
					InstallationMetadata: &multidevice.InstallationMetadata{
						DeviceType: message.DeviceType,
						Name:       message.Name,
					},
				}}
			m.handleInstallations(installations)
			// set WhisperTimestamp to pass the validation in HandlePairInstallation
			state.CurrentMessageState = &CurrentMessageState{WhisperTimestamp: message.Clock}
			err = m.HandlePairInstallation(state, message)
			if err != nil {
				return err
			}

			multidevice := m.encryptor.GetMultiDevice()
			if multidevice == nil {
				return errors.New("multidevice is nil")
			}
			_, err = multidevice.AddInstallations(m.IdentityPublicKeyCompressed(), int64(message.GetClock()), installations, true)
			if err != nil {
				return err
			}
			// if receiver already logged in before local pairing, we need force enable the installation,
			// AddInstallations won't make sure enable it, e.g. installation maybe already exist in db but not enabled yet
			err = m.EnableInstallation(message.InstallationId)
			if err != nil {
				return err
			}
		}
	}
	response, err := m.saveDataAndPrepareResponse(state)
	if err != nil {
		return err
	}
	publishMessengerResponse(response)
	return nil
}

// this is a copy implementation of the one in ext/service.go, we should refactor this?
func publishMessengerResponse(response *MessengerResponse) {
	if !response.IsEmpty() {
		notifications := response.Notifications()
		// Clear notifications as not used for now
		response.ClearNotifications()
		signal.SendNewMessages(response)
		localnotifications.PushMessages(notifications)
	}
}
