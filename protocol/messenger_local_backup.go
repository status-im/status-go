package protocol

import (
	"context"
	crand "crypto/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

// localBackupInterval is the duration we should allow between backups
var localBackupInterval = 1800 * time.Second // 30 minutes

func (m *Messenger) startLocalBackupLoop() {
	ticker := time.NewTicker(localBackupInterval)
	go func() {
		defer gocommon.LogOnPanic()
		for {
			select {
			case <-ticker.C:
				enabled, err := m.backupEnabled()
				if err != nil {
					m.logger.Error("failed to fetch backup enabled")
					continue
				}
				if !enabled || !m.config.featureFlags.EnableLocalBackup {
					m.logger.Debug("backup not enabled, skipping")
					continue
				}

				m.logger.Debug("backing up data")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				err = m.BackupDataLocally(ctx)
				if err != nil {
					m.logger.Error("failed to backup data", zap.Error(err))
				}
			case <-m.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (m *Messenger) BackupDataLocally(ctx context.Context) error {
	if !m.config.featureFlags.EnableLocalBackup {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()
	contactsToBackup := m.backupContacts(ctx)
	communitiesToBackup, err := m.backupCommunities(ctx, clock)
	if err != nil {
		return err
	}
	chatsToBackup := m.backupChats(ctx, clock)
	if err != nil {
		return err
	}
	profileToBackup, err := m.backupProfile(ctx, clock)
	if err != nil {
		return err
	}
	// _, settings, errors := m.prepareSyncSettingsMessages(clock, true)
	// if len(errors) != 0 {
	// 	// return just the first error, the others have been logged
	// 	return errors[0]
	// }
	// woAccountsToBackup, err := m.backupWatchOnlyAccounts()
	// if err != nil {
	// 	return 0, err
	// }

	fullBackup := &protobuf.Backup{}

	for _, d := range contactsToBackup {
		fullBackup.Contacts = append(fullBackup.Contacts, d.Contacts...)
	}
	for _, d := range communitiesToBackup {
		fullBackup.Communities = append(fullBackup.Communities, d.Communities...)
	}
	fullBackup.Profile = profileToBackup.Profile
	for _, d := range chatsToBackup {
		fullBackup.Chats = append(fullBackup.Chats, d.Chats...)
	}
	// for i, d := range settings {
	// 	// TODO find a way to get all settings
	// 	fullBackup.Setting = append(fullBackup.Setting, d)
	// }
	// for i, d := range woAccountsToBackup {
	// 	// TODO find a way to get all watchonlyaccounts
	// 	fullBackup.WatchOnlyAccount = append(fullBackup.WatchOnlyAccount, d.Keypair)
	// }

	// TODO put file in a constant
	path := filepath.Join(m.config.backupConfig.DataDir, "user_data.bkp")

	if err := os.MkdirAll(m.config.backupConfig.DataDir, 0700); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	mashalledMessage, err := proto.Marshal(fullBackup)
	if err != nil {
		return err
	}

	key, err := common.MakeECDHSharedKey(m.identity, &m.identity.PublicKey)
	if err != nil {
		return err
	}
	encryptedMessage, err := common.Encrypt(mashalledMessage, key, crand.Reader)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, encryptedMessage, 0600)
	if err != nil {
		m.logger.Error("failed to write backup message to file", zap.Error(err), zap.String("path", path))
		return err
	}

	chat.LastClockValue = clock
	err = m.saveChat(chat)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) importLocalBackupFile(filePath string) (*MessengerResponse, error) {
	if !m.config.featureFlags.EnableLocalBackup {
		return nil, nil
	}

	// Make sure the backup file exists
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Decrypt the backup file
	key, err := common.MakeECDHSharedKey(m.identity, &m.identity.PublicKey)
	if err != nil {
		return nil, err
	}
	decryptedPayload, err := common.Decrypt(content, key)
	if err != nil {
		return nil, err
	}

	// Unmarshal the decrypted payload to get the backup message
	var backupMessage protobuf.Backup
	err = proto.Unmarshal(decryptedPayload, &backupMessage)
	if err != nil {
		return nil, err
	}

	// Handle the backup
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
	err = m.HandleBackup(
		&state,
		&backupMessage,
		&v1protocol.StatusMessage{},
	)
	if err != nil {
		return nil, err
	}

	return m.saveDataAndPrepareResponse(&state)
}
