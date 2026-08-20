package protocol

import (
	"context"
	"database/sql"

	"go.uber.org/zap"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/internal/db/multiaccounts/errors"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/protocol/backupsync"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/signal"
	ensservice "github.com/status-im/status-go/services/ens"
)

const (
	SyncWakuSectionKeyProfile           = "profile"
	SyncWakuSectionKeyContacts          = "contacts"
	SyncWakuSectionKeyCommunities       = "communities"
	SyncWakuSectionKeySettings          = "settings"
	SyncWakuSectionKeyKeypairs          = "keypairs"
	SyncWakuSectionKeyWatchOnlyAccounts = "watchOnlyAccounts"
)

func (m *Messenger) handleLocalBackup(ctx context.Context, state *ReceivedMessageState, backup *protobuf.MessengerLocalBackup) []error {
	var errors []error

	err := m.handleBackedUpProfile(backup.Profile, backup.Clock)
	if err != nil {
		errors = append(errors, err)
	}

	if len(backup.Messages) > 0 {
		err := m.persistence.SaveBackedUpMessages(backup.Messages)
		if err != nil {
			errors = append(errors, err)
		}

		signal.LocalMessageBackupDone()
	}

	for _, contact := range backup.Contacts {
		err = m.HandleSyncInstallationContactV2(ctx, state, contact, nil)
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

	return errors
}

func (m *Messenger) handleBackedUpProfile(message *protobuf.BackedUpProfile, backupTime uint64) error {
	if message == nil {
		return nil
	}

	response := backupsync.BackedUpDataResponse{
		Profile: &backupsync.BackedUpProfile{},
	}

	err := common.ValidateDisplayName(&message.DisplayName)
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
		m.config.messengerSignalsHandler.SendBackedUpProfile(&response)
	}

	return err
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
		err := m.handleSyncInstallationCommunity(context.Background(), state, syncCommunity)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		err = m.requestCommunityKeysAndSharedAddresses(syncCommunity)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func (m *Messenger) PublishSettingEvent(settingField *settings.SyncSettingField) {
	if m.config.messengerSignalsHandler != nil {
		response := backupsync.BackedUpDataResponse{
			Setting: settingField,
		}
		m.config.messengerSignalsHandler.SendBackedUpSettings(&response)
	}
}

func (m *Messenger) requestCommunityKeysAndSharedAddresses(syncCommunity *protobuf.SyncInstallationCommunity) error {
	ctx, span := m.tracer.Start(context.Background(), "Messenger.requestCommunityKeysAndSharedAddresses")
	defer span.End()

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

	rawMessage := &common.RawMessage{
		Payload:             payload,
		Sender:              m.identity,
		CommunityID:         community.ID(),
		SkipEncryptionLayer: true,
		MessageType:         protobuf.ApplicationMetadataMessage_COMMUNITY_SHARED_ADDRESSES_REQUEST,
	}

	_, err = m.SendMessageToControlNode(ctx, community, rawMessage)

	if err != nil {
		m.logger.Error("failed to request shared addresses", zap.String("communityId", logutils.TruncateWithDot(community.IDString())), zap.Error(err))
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
			m.logger.Error("failed to request community encryption keys", zap.String("communityId", logutils.TruncateWithDot(community.IDString())), zap.Error(err))
			return err
		}
	}

	return nil
}

func (m *Messenger) HandleBackedUpMessageBatch(ctx context.Context, state *ReceivedMessageState, messageBatch *protobuf.BackedUpMessageBatch, msg *common.StatusMessage) error {
	// BackedUpMessages can only be sent in the context of a local backup
	return nil
}
