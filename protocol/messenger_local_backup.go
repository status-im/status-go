package protocol

import (
	"context"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/communities"
	contacts2 "github.com/status-im/status-go/protocol/contacts"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/signal"
)

const (
	BackupContactsPerBatch = 20
)

type CommunitySet struct {
	Joined    []*communities.Community
	Deleted   []*communities.Community
	Spectated []*communities.Community
}

func (m *Messenger) backupContacts() []*protobuf.Backup {
	var contacts []*protobuf.SyncInstallationContactV2
	myID := contacts2.ContactIDFromPublicKey(&m.identity.PublicKey)
	m.allContacts.Range(func(contactID string, contact *contacts2.Contact) (shouldContinue bool) {
		syncContact := m.buildSyncContactMessage(contact)
		if syncContact == nil {
			return true
		}

		// Only backup contacts that aren't ourselves
		if contact.ID == myID {
			return true
		}
		canonicalID, err := contacts2.ContactIDFromPublicKeyString(contact.ID)
		if err == nil && canonicalID == myID {
			return true
		}

		contacts = append(contacts, syncContact)

		return true
	})

	var backupMessages []*protobuf.Backup
	for i := 0; i < len(contacts); i += BackupContactsPerBatch {
		j := min(i+BackupContactsPerBatch, len(contacts))

		contactsToAdd := contacts[i:j]

		backupMessage := &protobuf.Backup{
			Contacts: contactsToAdd,
		}
		backupMessages = append(backupMessages, backupMessage)
	}

	return backupMessages
}

func (m *Messenger) retrieveAllCommunities() (*CommunitySet, error) {
	joinedCs, err := m.communitiesManager.JoinedAndPendingCommunitiesWithRequests()
	if err != nil {
		return nil, err
	}

	deletedCs, err := m.communitiesManager.DeletedCommunities()
	if err != nil {
		return nil, err
	}

	spectatedCs, err := m.communitiesManager.Spectated()
	if err != nil {
		return nil, err
	}

	return &CommunitySet{
		Joined:    joinedCs,
		Deleted:   deletedCs,
		Spectated: spectatedCs,
	}, nil
}

func (m *Messenger) backupCommunity(community *communities.Community, clock uint64) (*protobuf.Backup, error) {
	communityId := community.ID()
	settings, err := m.communitiesManager.GetCommunitySettingsByID(communityId)
	if err != nil {
		return nil, err
	}

	syncControlNode, err := m.communitiesManager.GetSyncControlNode(communityId)
	if err != nil {
		return nil, err
	}

	syncMessage, err := community.ToSyncInstallationCommunityProtobuf(clock, settings, syncControlNode)
	if err != nil {
		return nil, err
	}

	err = m.propagateSyncInstallationCommunityWithHRKeys(syncMessage, community)
	if err != nil {
		return nil, err
	}

	return &protobuf.Backup{
		Communities: []*protobuf.SyncInstallationCommunity{syncMessage},
	}, nil
}

func (m *Messenger) backupCommunities(clock uint64) ([]*protobuf.Backup, error) {
	communitySet, err := m.retrieveAllCommunities()
	if err != nil {
		return nil, err
	}

	var backupMessages []*protobuf.Backup
	combinedCs := append(communitySet.Joined, communitySet.Deleted...)
	combinedCs = append(combinedCs, communitySet.Spectated...)

	for _, c := range combinedCs {
		_, beingImported := m.importingCommunities[c.IDString()]
		if !beingImported {
			backupMessage, err := m.backupCommunity(c, clock)
			if err != nil {
				return nil, err
			}

			backupMessages = append(backupMessages, backupMessage)
		}
	}

	return backupMessages, nil
}

func (m *Messenger) backupChats(clock uint64) []*protobuf.Backup {
	var oneToOneAndGroupChats []*protobuf.SyncChat
	m.allChats.Range(func(chatID string, chat *Chat) bool {
		if !chat.OneToOne() && !chat.PrivateGroupChat() {
			return true
		}
		syncChat := protobuf.SyncChat{
			Clock:    clock,
			Id:       chatID,
			ChatType: uint32(chat.ChatType),
			Active:   chat.Active,
		}
		chatMuteTill, _ := time.Parse(time.RFC3339, chat.MuteTill.Format(time.RFC3339))
		if chat.Muted && chatMuteTill.Equal(time.Time{}) {
			// Only set Muted if it is "permanently" muted
			syncChat.Muted = true
		}
		if chat.PrivateGroupChat() {
			syncChat.Name = chat.Name // The Name is only useful in the case of a group chat

			syncChat.MembershipUpdateEvents = make([]*protobuf.MembershipUpdateEvents, len(chat.MembershipUpdates))
			for i, membershipUpdate := range chat.MembershipUpdates {
				syncChat.MembershipUpdateEvents[i] = &protobuf.MembershipUpdateEvents{
					Clock:      membershipUpdate.ClockValue,
					Type:       uint32(membershipUpdate.Type),
					Members:    membershipUpdate.Members,
					Name:       membershipUpdate.Name,
					Signature:  membershipUpdate.Signature,
					ChatId:     membershipUpdate.ChatID,
					From:       membershipUpdate.From,
					RawPayload: membershipUpdate.RawPayload,
					Color:      membershipUpdate.Color,
					Image:      membershipUpdate.Image,
				}
			}
		}
		oneToOneAndGroupChats = append(oneToOneAndGroupChats, &syncChat)
		return true
	})

	var backupMessages []*protobuf.Backup
	backupMessage := &protobuf.Backup{
		Chats: oneToOneAndGroupChats,
	}
	backupMessages = append(backupMessages, backupMessage)
	return backupMessages
}

func (m *Messenger) backupProfile(clock uint64) (*protobuf.Backup, error) {
	displayName, err := m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	displayNameClock, err := m.settings.GetSettingLastSynced(settings.DisplayName)
	if err != nil {
		return nil, err
	}

	if m.account == nil {
		return nil, nil
	}

	keyUID := m.account.KeyUID
	images, err := m.multiAccounts.GetIdentityImages(keyUID)
	if err != nil {
		return nil, err
	}

	pictureProtos := make([]*protobuf.SyncProfilePicture, len(images))
	for i, image := range images {
		p := &protobuf.SyncProfilePicture{}
		p.Name = image.Name
		p.Payload = image.Payload
		p.Width = uint32(image.Width)
		p.Height = uint32(image.Height)
		p.FileSize = uint32(image.FileSize)
		p.ResizeTarget = uint32(image.ResizeTarget)
		if image.Clock == 0 {
			p.Clock = clock
		} else {
			p.Clock = image.Clock
		}
		pictureProtos[i] = p
	}

	ensUsernameDetails, err := m.getEnsUsernameDetails()
	if err != nil {
		return nil, err
	}
	ensUsernameDetailProtos := make([]*protobuf.SyncEnsUsernameDetail, len(ensUsernameDetails))
	for i, ensUsernameDetail := range ensUsernameDetails {
		ensUsernameDetailProtos[i] = &protobuf.SyncEnsUsernameDetail{
			Username: ensUsernameDetail.Username,
			Clock:    ensUsernameDetail.Clock,
			Removed:  ensUsernameDetail.Removed,
			ChainId:  ensUsernameDetail.ChainID,
		}
	}

	profileShowcasePreferences, err := m.GetProfileShowcasePreferences()
	if err != nil {
		return nil, err
	}

	backupMessage := &protobuf.Backup{
		Profile: &protobuf.BackedUpProfile{
			KeyUid:                     keyUID,
			DisplayName:                displayName,
			Pictures:                   pictureProtos,
			DisplayNameClock:           displayNameClock,
			EnsUsernameDetails:         ensUsernameDetailProtos,
			ProfileShowcasePreferences: ToProfileShowcasePreferencesProto(profileShowcasePreferences),
		},
	}

	return backupMessage, nil
}

func (m *Messenger) backupMessages() ([]*protobuf.BackedUpMessage, error) {
	messagesBackupEnabled, err := m.settings.MessagesBackupEnabled()
	if err != nil {
		return nil, err
	}

	if !messagesBackupEnabled {
		return nil, nil
	}

	return m.persistence.AllMessagesForBackup()
}

func (m *Messenger) ExportBackup() ([]byte, error) {
	backup := &protobuf.MessengerLocalBackup{}

	clock, _ := m.getLastClockWithRelatedChat()
	contactsToBackup := m.backupContacts()
	communitiesToBackup, err := m.backupCommunities(clock)
	if err != nil {
		return nil, err
	}
	chatsToBackup := m.backupChats(clock)
	profileToBackup, err := m.backupProfile(clock)
	if err != nil {
		return nil, err
	}

	for _, d := range contactsToBackup {
		backup.Contacts = append(backup.Contacts, d.Contacts...)
	}
	for _, d := range communitiesToBackup {
		backup.Communities = append(backup.Communities, d.Communities...)
	}
	backup.Profile = profileToBackup.Profile
	for _, d := range chatsToBackup {
		backup.Chats = append(backup.Chats, d.Chats...)
	}

	backupMessages, err := m.backupMessages()
	if err != nil {
		return nil, err
	}
	backup.Messages = backupMessages

	return proto.Marshal(backup)
}

func (m *Messenger) ImportBackup(data []byte) error {
	var backup protobuf.MessengerLocalBackup
	err := proto.Unmarshal(data, &backup)
	if err != nil {
		return err
	}

	state := ReceivedMessageState{
		Response: &MessengerResponse{},
		AllChats: &chatMap{},
		AllContacts: &contactMap{
			me: m.selfContact,
		},
		Timesource:            m.getTimesource(),
		ModifiedContacts:      &stringBoolMap{},
		ModifiedInstallations: &stringBoolMap{},
	}
	errs := m.handleLocalBackup(
		context.Background(),
		&state,
		&backup,
	)
	// Proceed to save data even if there were errors during handling because some data might have been handled successfully
	response, err := m.saveDataAndPrepareResponse(&state)
	if response != nil {
		// Send response even if there were errors as some data might have been saved successfully
		signal.SendNewMessages(response)
	}
	if err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
