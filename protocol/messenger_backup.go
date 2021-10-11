package protocol

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

const (
	BackupContactsPerBatch = 20
)

// backupTickerInterval is how often we should check for backups
var backupTickerInterval = 120 * time.Second

// backupIntervalSeconds is the amount of seconds we should allow between
// backups
var backupIntervalSeconds uint64 = 28800

func (m *Messenger) backupEnabled() (bool, error) {
	return m.settings.BackupEnabled()
}

func (m *Messenger) lastBackup() (uint64, error) {
	return m.settings.LastBackup()
}

func (m *Messenger) startBackupLoop() {

	ticker := time.NewTicker(backupTickerInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if !m.online() {
					continue
				}

				enabled, err := m.backupEnabled()
				if err != nil {
					m.logger.Error("failed to fetch backup enabled")
					continue
				}
				if !enabled {
					m.logger.Debug("backup not enabled, skipping")
					continue
				}

				lastBackup, err := m.lastBackup()
				if err != nil {
					m.logger.Error("failed to fetch last backup time")
				}

				now := time.Now().Unix()
				if uint64(now) <= backupIntervalSeconds+lastBackup {
					m.logger.Debug("not backing up")
					continue
				}
				m.logger.Debug("backing up data")

				_, err = m.BackupData(context.Background())
				if err != nil {
					m.logger.Error("failed to backup data", zap.Error(err))
				}
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) BackupData(ctx context.Context) (uint64, error) {
	var contacts []*protobuf.SyncInstallationContactV2
	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		syncContact := m.syncBackupContact(ctx, contact)
		if syncContact != nil {
			contacts = append(contacts, syncContact)
		}
		return true
	})

	clock, chat := m.getLastClockWithRelatedChat()

	for i := 0; i < len(contacts); i += BackupContactsPerBatch {
		j := i + BackupContactsPerBatch
		if j > len(contacts) {
			j = len(contacts)
		}

		contactsToAdd := contacts[i:j]

		backupMessage := &protobuf.Backup{
			Contacts: contactsToAdd,
		}

		encodedMessage, err := proto.Marshal(backupMessage)
		if err != nil {
			return 0, err
		}

		_, err = m.dispatchMessage(ctx, common.RawMessage{
			LocalChatID:    chat.ID,
			Payload:        encodedMessage,
			SkipEncryption: true,
			MessageType:    protobuf.ApplicationMetadataMessage_BACKUP,
		})
		if err != nil {
			return 0, err
		}

	}
	chat.LastClockValue = clock
	err := m.saveChat(chat)
	if err != nil {
		return 0, err
	}
	err = m.settings.SetLastBackup(clock)
	if err != nil {
		return 0, err
	}
	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.BackupPerformed(clock)
	}

	return clock, nil
}

// syncContact sync as contact with paired devices
func (m *Messenger) syncBackupContact(ctx context.Context, contact *Contact) *protobuf.SyncInstallationContactV2 {
	if contact.IsSyncing {
		return nil
	}
	clock, _ := m.getLastClockWithRelatedChat()

	var ensName string
	if contact.ENSVerified {
		ensName = contact.Name
	}

	oneToOneChat, ok := m.allChats.Load(contact.ID)
	muted := false
	if ok {
		muted = oneToOneChat.Muted
	}

	return &protobuf.SyncInstallationContactV2{
		Clock:         clock,
		Id:            contact.ID,
		EnsName:       ensName,
		LocalNickname: contact.LocalNickname,
		Added:         contact.Added,
		Blocked:       contact.Blocked,
		Muted:         muted,
		Removed:       contact.Removed,
	}
}
