package protocol

import (
	"database/sql"

	ensservice "github.com/status-im/status-go/services/ens"

	"github.com/status-im/status-go/protocol/identity"

	"go.uber.org/zap"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/wakusync"
)

const (
	SyncWakuSectionKeyProfile           = "profile"
	SyncWakuSectionKeyContacts          = "contacts"
	SyncWakuSectionKeyCommunities       = "communities"
	SyncWakuSectionKeySettings          = "settings"
	SyncWakuSectionKeyKeypairs          = "keypairs"
	SyncWakuSectionKeyWatchOnlyAccounts = "watchOnlyAccounts"
)

func (m *Messenger) HandleBackup(state *ReceivedMessageState, message protobuf.Backup) []error {
	var errors []error

	err := m.handleBackedUpProfile(message.Profile, message.Clock)
	if err != nil {
		errors = append(errors, err)
	}

	for _, contact := range message.Contacts {
		err = m.HandleSyncInstallationContact(state, *contact)
		if err != nil {
			errors = append(errors, err)
		}
	}

	for _, community := range message.Communities {
		err = m.handleSyncCommunity(state, *community)
		if err != nil {
			errors = append(errors, err)
		}
	}
	err = m.handleBackedUpSettings(message.Setting)
	if err != nil {
		errors = append(errors, err)
	}

	err = m.handleKeypair(message.Keypair)
	if err != nil {
		errors = append(errors, err)
	}

	err = m.handleWatchOnlyAccount(message.WatchOnlyAccount)
	if err != nil {
		errors = append(errors, err)
	}

	// Send signal about applied backup progress
	if m.config.messengerSignalsHandler != nil {
		response := wakusync.WakuBackedUpDataResponse{
			Clock: message.Clock,
		}

		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeyProfile, message.ProfileDetails)
		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeyContacts, message.ContactsDetails)
		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeyCommunities, message.CommunitiesDetails)
		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeySettings, message.SettingsDetails)
		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeyKeypairs, message.KeypairDetails)
		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeyWatchOnlyAccounts, message.WatchOnlyAccountDetails)

		m.config.messengerSignalsHandler.SendWakuFetchingBackupProgress(&response)
	}

	state.Response.BackupHandled = true

	return errors
}

func (m *Messenger) handleBackedUpProfile(message *protobuf.BackedUpProfile, backupTime uint64) error {
	if message == nil {
		return nil
	}

	response := wakusync.WakuBackedUpDataResponse{
		Profile: &wakusync.BackedUpProfile{},
	}

	err := m.SaveSyncDisplayName(message.DisplayName, message.DisplayNameClock)
	if err != nil {
		return err
	}

	response.SetDisplayName(message.DisplayName)

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
			response.SetImages(idImages)
		}
	}

	err = m.handleSyncSocialLinks(message.SocialLinks, func(links identity.SocialLinks) {
		response.SetSocialLinks(links)
	})
	if err != nil {
		return err
	}

	var ensUsernameDetails []*ensservice.UsernameDetail
	for _, d := range message.EnsUsernameDetails {
		dd, err := m.saveEnsUsernameDetailProto(*d)
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

func (m *Messenger) handleBackedUpSettings(message *protobuf.SyncSetting) error {
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

	if settingField != nil && m.config.messengerSignalsHandler != nil {
		response := wakusync.WakuBackedUpDataResponse{
			Setting: settingField,
		}

		m.config.messengerSignalsHandler.SendWakuBackedUpSettings(&response)
	}

	return nil
}

func (m *Messenger) handleKeypair(message *protobuf.SyncKeypair) error {
	if message == nil {
		return nil
	}

	keypair, err := m.handleSyncKeypair(message)
	if err != nil {
		if err == ErrTryingToStoreOldKeypair {
			return nil
		}
		return err
	}

	if m.config.messengerSignalsHandler != nil {
		kpResponse := wakusync.WakuBackedUpDataResponse{
			Keypair: keypair.CopyKeypair(),
		}

		m.config.messengerSignalsHandler.SendWakuBackedUpKeypair(&kpResponse)
	}

	return nil
}

func (m *Messenger) handleWatchOnlyAccount(message *protobuf.SyncAccount) error {
	if message == nil {
		return nil
	}

	acc, err := m.handleSyncWatchOnlyAccount(message)
	if err != nil {
		if err == ErrTryingToStoreOldWalletAccount {
			return nil
		}
		return err
	}

	if m.config.messengerSignalsHandler != nil {
		response := wakusync.WakuBackedUpDataResponse{
			WatchOnlyAccount: acc,
		}

		m.config.messengerSignalsHandler.SendWakuBackedUpWatchOnlyAccount(&response)
	}

	return nil
}
