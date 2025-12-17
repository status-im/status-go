package protocol

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/contacts"
	"github.com/status-im/status-go/protocol/protobuf"
)

type RawMessageHandler func(ctx context.Context, rawMessage common.RawMessage) (common.RawMessage, error)

func (m *Messenger) HandleSyncRawMessages(rawMessages []*protobuf.RawMessage) error {
	ctx, span := m.tracer.Start(context.Background(), "Messenger.HandleSyncRawMessages")
	defer span.End()

	state := m.buildMessageState()
	for _, rawMessage := range rawMessages {
		switch rawMessage.GetMessageType() {
		case protobuf.ApplicationMetadataMessage_CONTACT_UPDATE:
			var message protobuf.ContactUpdate
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			publicKey, err := crypto.HexToPubkey(message.PublicKey)
			if err != nil {
				return err
			}
			var contact *contacts.Contact
			if c, ok := state.AllContacts.Load(message.PublicKey); ok {
				contact = c
			} else {
				c, err := contacts.BuildContact(message.PublicKey, publicKey)
				if err != nil {
					m.logger.Info("failed to build contact", zap.Error(err))
					continue
				}
				contact = c
				state.AllContacts.Store(message.PublicKey, contact)
			}
			currentMessageState := &CurrentMessageState{
				Message: &protobuf.ChatMessage{
					Clock: message.Clock,
				},
				MessageID:        " ", // make it not empty to bypass this validation: https://github.com/status-im/status-go/blob/7cd7430d3141b08f7c455d7918f4160ea8fd0559/protocol/messenger_handler.go#L325
				WhisperTimestamp: message.Clock,
				PublicKey:        publicKey,
				Contact:          contact,
			}
			state.CurrentMessageState = currentMessageState
			err = m.HandleContactUpdate(ctx, state, &message, nil)
			if err != nil {
				m.logger.Warn("failed to HandleContactUpdate when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_CHAT:
			var message protobuf.SyncChat
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncChat(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("error createChat when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_CHAT_REMOVED:
			var message protobuf.SyncChatRemoved
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncChatRemoved(ctx, state, &message, nil)
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
			err = m.HandleSyncChatMessagesRead(ctx, state, &message, nil)
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
			err = m.HandleSyncClearHistory(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to handleSyncClearHistory when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_CONTACT_V2:
			var message protobuf.SyncInstallationContactV2
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncInstallationContactV2(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to HandleSyncInstallationContact when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_COMMUNITY:
			var message protobuf.SyncInstallationCommunity
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncInstallationCommunity(ctx, state, &message)
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
			err = m.HandleSyncBookmark(ctx, state, &message, nil)
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
			err = m.HandleSyncTrustedUser(ctx, state, &message, nil)
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
			err = m.HandleSyncVerificationRequest(ctx, state, &message, nil)
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
			err = m.HandleSyncSetting(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to handleSyncSetting when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_PROFILE_PICTURES:
			var message protobuf.SyncProfilePictures
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncProfilePictures(ctx, state, &message, nil)
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
			err = m.HandleSyncContactRequestDecision(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to HandleSyncContactRequestDecision when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_ACCOUNT:
			var message protobuf.SyncAccount
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncAccount(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to HandleSyncWatchOnlyAccount when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_KEYPAIR:
			var message protobuf.SyncKeypair
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.handleSyncKeypairInternal(state, &message, true)
			if err != nil {
				m.logger.Error("failed to HandleSyncKeypair when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_ACCOUNTS_POSITIONS:
			var message protobuf.SyncAccountsPositions
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncAccountsPositions(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to HandleSyncAccountsPositions when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_TOKEN_PREFERENCES:
			var message protobuf.SyncTokenPreferences
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncTokenPreferences(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to HandleSyncTokenPreferences when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_COLLECTIBLE_PREFERENCES:
			var message protobuf.SyncCollectiblePreferences
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncCollectiblePreferences(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to HandleSyncCollectiblePreferences when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_SAVED_ADDRESS:
			var message protobuf.SyncSavedAddress
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncSavedAddress(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to handleSyncSavedAddress when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_ENS_USERNAME_DETAIL:
			var message protobuf.SyncEnsUsernameDetail
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncEnsUsernameDetail(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to handleSyncEnsUsernameDetail when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_DELETE_FOR_ME_MESSAGE:
			var message protobuf.SyncDeleteForMeMessage
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			err = m.HandleSyncDeleteForMeMessage(ctx, state, &message, nil)
			if err != nil {
				m.logger.Error("failed to HandleDeleteForMeMessage when HandleSyncRawMessages", zap.Error(err))
				continue
			}
		case protobuf.ApplicationMetadataMessage_SYNC_PAIR_INSTALLATION:
			var message protobuf.SyncPairInstallation
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			identity := m.myHexIdentity()
			installations := []*messagingtypes.Installation{
				{
					Identity:  identity,
					ID:        message.InstallationId,
					Version:   message.Version,
					Enabled:   true,
					Timestamp: int64(message.Clock),
					InstallationMetadata: &messagingtypes.InstallationMetadata{
						DeviceType: message.DeviceType,
						Name:       message.Name,
					},
				}}
			m.handleInstallations(installations)
			// set WhisperTimestamp to pass the validation in HandleSyncPairInstallation
			state.CurrentMessageState = &CurrentMessageState{WhisperTimestamp: message.Clock}
			err = m.HandleSyncPairInstallation(ctx, state, &message, nil)
			if err != nil {
				return err
			}

			_, err = m.messaging.AddInstallations(m.IdentityPublicKeyCompressed(), int64(message.GetClock()), installations, true)
			if err != nil {
				return err
			}
			// if receiver already logged in before local pairing, we need force enable the installation,
			// AddInstallations won't make sure enable it, e.g. installation maybe already exist in db but not enabled yet
			_, err = m.EnableInstallation(message.InstallationId)
			if err != nil {
				return err
			}
		case protobuf.ApplicationMetadataMessage_SYNC_PROFILE_SHOWCASE_PREFERENCES:
			var message protobuf.SyncProfileShowcasePreferences
			err := proto.Unmarshal(rawMessage.GetPayload(), &message)
			if err != nil {
				return err
			}
			_, err = m.saveProfileShowcasePreferencesProto(&message, false)
			if err != nil {
				return err
			}
		case protobuf.ApplicationMetadataMessage_BACKED_UP_MESSAGE_BATCH:
			var messageBatch protobuf.BackedUpMessageBatch
			err := proto.Unmarshal(rawMessage.GetPayload(), &messageBatch)
			if err != nil {
				return err
			}
			err = m.persistence.SaveBackedUpMessages(messageBatch.Messages)
			if err != nil {
				return err
			}
		}
	}
	response, err := m.saveDataAndPrepareResponse(state)
	if err != nil {
		return err
	}
	m.PublishMessengerResponse(response)
	return nil
}
