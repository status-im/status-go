package protocol

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/wakusync"
)

const (
	// timeToPostponeBackedUpMessagesHandling - the idea is to wait for 30 secs in loading state with the hope that at least
	// one message from the set of last backed up messages will arrive, that we can know which set of messages to work with
	timeToPostponeBackedUpMessagesHandling = 30 * time.Second

	SyncWakuSectionKeyProfile     = "profile"
	SyncWakuSectionKeyContacts    = "contacts"
	SyncWakuSectionKeyCommunities = "communities"
	SyncWakuSectionKeySettings    = "settings"
)

type backupHandler struct {
	messagesToProceed      []protobuf.Backup
	lastKnownTime          uint64
	postponeHandling       bool
	postponeTasksWaitGroup sync.WaitGroup
}

func (bh *backupHandler) addMessage(message protobuf.Backup) {
	if message.Clock < bh.lastKnownTime {
		return
	}
	if message.Clock > bh.lastKnownTime {
		bh.messagesToProceed = nil
		bh.lastKnownTime = message.Clock
	}

	bh.messagesToProceed = append(bh.messagesToProceed, message)
}

func (m *Messenger) startWaitingForTheLatestBackedupMessageLoop() {
	ticker := time.NewTicker(timeToPostponeBackedUpMessagesHandling)
	m.backedupMessagesHandler.postponeTasksWaitGroup.Add(1)
	go func() {
		defer m.backedupMessagesHandler.postponeTasksWaitGroup.Done()
		for {
			select {
			case <-ticker.C:
				if !m.backedupMessagesHandler.postponeHandling {
					return
				}
				m.backedupMessagesHandler.postponeHandling = false

				for _, msg := range m.backedupMessagesHandler.messagesToProceed {
					messageState := m.buildMessageState()
					errors := m.HandleBackup(messageState, msg)
					if len(errors) > 0 {
						for _, err := range errors {
							m.logger.Warn("failed to handle postponed Backup messages", zap.Error(err))
						}
					}
				}

				m.backedupMessagesHandler.messagesToProceed = nil
			case <-m.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (m *Messenger) HandleBackup(state *ReceivedMessageState, message protobuf.Backup) []error {
	var errors []error
	if m.backedupMessagesHandler.postponeHandling {
		m.backedupMessagesHandler.addMessage(message)
		return errors
	}

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

	if m.config.messengerSignalsHandler != nil {
		response := wakusync.WakuBackedUpDataResponse{}

		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeyProfile, message.ProfileDetails)
		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeyContacts, message.ContactsDetails)
		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeyCommunities, message.CommunitiesDetails)
		response.AddFetchingBackedUpDataDetails(SyncWakuSectionKeySettings, message.SettingsDetails)

		m.config.messengerSignalsHandler.SendWakuFetchingBackupProgress(&response)
	}

	return errors
}

func (m *Messenger) handleBackedUpProfile(message *protobuf.BackedUpProfile, backupTime uint64) error {
	if message == nil {
		return nil
	}

	dbDisplayNameClock, err := m.settings.GetSettingLastSynced(settings.DisplayName)
	if err != nil {
		return err
	}

	response := wakusync.WakuBackedUpDataResponse{
		Profile: &wakusync.BackedUpProfile{},
	}

	if dbDisplayNameClock < message.DisplayNameClock {
		err := m.SetDisplayName(message.DisplayName, false)
		response.AddDisplayName(message.DisplayName, err == nil)
	}

	syncWithBackedUpImages := false
	dbImages, err := m.multiAccounts.GetIdentityImages(message.KeyUid)
	if err != nil {
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
			response.AddImages(nil, err == nil)
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
			response.AddImages(idImages, err == nil)
		}
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.SendWakuBackedUpProfile(&response)
	}

	return err
}

func (m *Messenger) handleBackedUpSettings(message *protobuf.SyncSetting) error {
	if message == nil {
		return nil
	}

	settingField, err := m.extractSyncSetting(message)
	if err != nil {
		m.logger.Warn("failed to handle SyncSetting from backed up message", zap.Error(err))
		return nil
	}

	response := wakusync.WakuBackedUpDataResponse{
		Setting: settingField,
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.SendWakuBackedUpSettings(&response)
	}

	return nil
}
