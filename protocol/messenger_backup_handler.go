package protocol

import (
	"database/sql"

	"go.uber.org/zap"

	"github.com/golang/protobuf/proto"

	utils "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/images"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/wakusync"
	ensservice "github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/signal"
)

const (
	SyncWakuSectionKeyProfile           = "profile"
	SyncWakuSectionKeyContacts          = "contacts"
	SyncWakuSectionKeyCommunities       = "communities"
	SyncWakuSectionKeySettings          = "settings"
	SyncWakuSectionKeyKeypairs          = "keypairs"
	SyncWakuSectionKeyWatchOnlyAccounts = "watchOnlyAccounts"
)

func (m *Messenger) handleLocalBackup(state *ReceivedMessageState, backup *protobuf.MessengerLocalBackup) []error {
	var errors []error

	err := m.handleBackedUpProfile(backup.Profile, backup.Clock)
	if err != nil {
		errors = append(errors, err)
	}

	for _, contact := range backup.Contacts {
		err = m.HandleSyncInstallationContactV2(state, contact, nil)
		if err != nil {
			errors = append(errors, err)
		}
	}

	err = m.handleSyncChats(state, backup.Chats)
	if err != nil {
		errors = append(errors, err)
	}

	communityErrors := m.handleLocalBackupCommunities(state, backup.Communities)
	if len(communityErrors) > 0 {
		errors = append(errors, communityErrors...)
	}

	if len(backup.Messages) > 0 {
		err := m.persistence.SaveBackedUpMessages(backup.Messages)
		if err != nil {
			errors = append(errors, err)
		}

		signal.LocalMessageBackupDone()
	}

	return errors
}

func (m *Messenger) handleBackedUpProfile(message *protobuf.BackedUpProfile, backupTime uint64) error {
	if message == nil {
		return nil
	}

	response := wakusync.WakuBackedUpDataResponse{
		Profile: &wakusync.BackedUpProfile{},
	}

	err := utils.ValidateDisplayName(&message.DisplayName)
	if err != nil {
		// Print a warning and set the display name to the account name, but don't stop the recovery
		m.logger.Warn("invalid display name found", zap.Error(err))
		response.SetDisplayName(m.account.Name)
	} else {
		err = m.SaveSyncDisplayName(message.DisplayName, message.DisplayNameClock)
		if err != nil && err != errors.ErrNewClockOlderThanCurrent {
			return err
		}

		response.SetDisplayName(message.DisplayName)

		// if we already have a newer clock, then we don't need to update the display name
		if err == errors.ErrNewClockOlderThanCurrent {
			response.SetDisplayName(m.account.Name)
		}
	}

	syncWithBackedUpImages := false
	dbImages, err := m.multiAccounts.GetIdentityImages(message.KeyUid)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		// if images are deleted and no images were backed up, then we need to delete them on other devices,
		// that's why we don't return in case of `sql.ErrNoRows`
		syncWithBackedUpImages = true
	}
	if len(dbImages) == 0 {
		if len(message.Pictures) > 0 {
			syncWithBackedUpImages = true
		}
	} else {
		// since both images (large and thumbnail) are always stored in the same time, we're free to use either of those two clocks for comparison
		lastImageStoredClock := dbImages[0].Clock
		syncWithBackedUpImages = lastImageStoredClock < backupTime
	}

	if syncWithBackedUpImages {
		if len(message.Pictures) == 0 {
			err = m.multiAccounts.DeleteIdentityImage(message.KeyUid)
			if err != nil {
				return err
			}
			response.SetImages(nil)
		} else {
			idImages := make([]images.IdentityImage, len(message.Pictures))
			for i, pic := range message.Pictures {
				img := images.IdentityImage{
					Name:         pic.Name,
					Payload:      pic.Payload,
					Width:        int(pic.Width),
					Height:       int(pic.Height),
					FileSize:     int(pic.FileSize),
					ResizeTarget: int(pic.ResizeTarget),
					Clock:        pic.Clock,
				}
				idImages[i] = img
			}
			err = m.multiAccounts.StoreIdentityImages(message.KeyUid, idImages, false)
			if err != nil {
				return err
			}

			if m.httpServer != nil {
				for j, image := range idImages {
					idImages[j].LocalURL = m.httpServer.MakeAccountImageURL(message.KeyUid, image.Name, image.Clock)
				}
			}

			response.SetImages(idImages)
		}
	}

	profileShowcasePreferences, err := m.saveProfileShowcasePreferencesProto(message.ProfileShowcasePreferences, false)
	if err != nil {
		return err
	}
	if profileShowcasePreferences != nil {
		response.SetProfileShowcasePreferences(profileShowcasePreferences)
	}

	var ensUsernameDetails []*ensservice.UsernameDetail
	for _, d := range message.EnsUsernameDetails {
		dd, err := m.saveEnsUsernameDetailProto(d)
		if err != nil {
			return err
		}
		ensUsernameDetails = append(ensUsernameDetails, dd)
	}
	response.SetEnsUsernameDetails(ensUsernameDetails)

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.SendWakuBackedUpProfile(&response)
	}

	return err
}

// TODO move this to the AccountsService
func (m *Messenger) HandleBackedUpSettings(message *protobuf.SyncSetting) error {
	if message == nil {
		return nil
	}

	// DisplayName is recovered via `protobuf.BackedUpProfile` message
	if message.GetType() == protobuf.SyncSetting_DISPLAY_NAME {
		return nil
	}

	settingField, err := m.extractAndSaveSyncSetting(message)
	if err != nil {
		m.logger.Warn("failed to handle SyncSetting from backed up message", zap.Error(err))
		return nil
	}

	if settingField != nil {
		if message.GetType() == protobuf.SyncSetting_PREFERRED_NAME && message.GetValueString() != "" {
			displayNameClock, err := m.settings.GetSettingLastSynced(settings.DisplayName)
			if err != nil {
				m.logger.Warn("failed to get last synced clock for display name", zap.Error(err))
				return nil
			}
			// there is a race condition between display name and preferred name on updating m.account.Name, so we need to check the clock
			// there is also a similar check within SaveSyncDisplayName
			if displayNameClock < message.GetClock() {
				m.account.Name = message.GetValueString()
				err = m.multiAccounts.SaveAccount(*m.account)
				if err != nil {
					m.logger.Warn("[HandleBackedUpSettings] failed to save account", zap.Error(err))
					return nil
				}
			}
		}

		if m.config.messengerSignalsHandler != nil {
			response := wakusync.WakuBackedUpDataResponse{
				Setting: settingField,
			}
			m.config.messengerSignalsHandler.SendWakuBackedUpSettings(&response)
		}
	}

	return nil
}

func syncInstallationCommunitiesSet(communities []*protobuf.SyncInstallationCommunity) map[string]*protobuf.SyncInstallationCommunity {
	ret := map[string]*protobuf.SyncInstallationCommunity{}
	for _, c := range communities {
		id := string(c.GetId())
		prevC, ok := ret[id]
		if !ok || prevC.Clock < c.Clock {
			ret[id] = c
		}
	}
	return ret
}

func (m *Messenger) handleLocalBackupCommunities(state *ReceivedMessageState, communities []*protobuf.SyncInstallationCommunity) []error {
	var errors []error
	for _, syncCommunity := range syncInstallationCommunitiesSet(communities) {
		err := m.handleSyncInstallationCommunity(state, syncCommunity)
		if err != nil {
			errors = append(errors, err)
		}

		err = m.requestCommunityKeysAndSharedAddresses(syncCommunity)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func (m *Messenger) requestCommunityKeysAndSharedAddresses(syncCommunity *protobuf.SyncInstallationCommunity) error {
	if !syncCommunity.Joined {
		return nil
	}

	community, err := m.GetCommunityByID(syncCommunity.Id)
	if err != nil {
		return err
	}

	if community == nil {
		return communities.ErrOrgNotFound
	}

	// Send a request to get back our previous shared addresses
	request := &protobuf.CommunitySharedAddressesRequest{
		CommunityId: syncCommunity.Id,
	}

	payload, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	rawMessage := &messagingtypes.RawMessage{
		Payload:             payload,
		Sender:              m.identity,
		CommunityID:         community.ID(),
		SkipEncryptionLayer: true,
		MessageType:         protobuf.ApplicationMetadataMessage_COMMUNITY_SHARED_ADDRESSES_REQUEST,
	}

	_, err = m.SendMessageToControlNode(community, rawMessage)

	if err != nil {
		m.logger.Error("failed to request shared addresses", zap.String("communityId", utils.TruncateWithDot(community.IDString())), zap.Error(err))
		return err
	}

	// If the community is encrypted or one channel is, ask for the encryption keys back
	isEncrypted := syncCommunity.Encrypted || len(syncCommunity.EncryptionKeysV2) > 0
	if !isEncrypted {
		// check if we have encrypted channels
		myPk := m.IdentityPublicKeyString()
		for channelID, channel := range community.Chats() {
			_, exists := channel.GetMembers()[myPk]
			if exists && community.ChannelEncrypted(channelID) {
				isEncrypted = true
				break
			}
		}
	}

	if isEncrypted {
		err = m.requestCommunityEncryptionKeys(community, nil)
		if err != nil {
			m.logger.Error("failed to request community encryption keys", zap.String("communityId", utils.TruncateWithDot(community.IDString())), zap.Error(err))
			return err
		}
	}

	return nil
}

func (m *Messenger) HandleBackedUpMessageBatch(state *ReceivedMessageState, messageBatch *protobuf.BackedUpMessageBatch, msg *messagingtypes.Message) error {
	// BackedUpMessages can only be sent in the context of a local backup
	return nil
}
