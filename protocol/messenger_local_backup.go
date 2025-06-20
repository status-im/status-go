package protocol

import (
	"context"
	crand "crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

// localBackupInterval is the duration we should allow between backups
var localBackupInterval = 30 * time.Minute

func (m *Messenger) startLocalBackupLoop() {
	ticker := time.NewTicker(localBackupInterval)
	defer ticker.Stop()
	m.shutdownWaitGroup.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
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

				ctx, cancel := context.WithTimeout(m.ctx, 5*time.Minute)
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
	profileToBackup, err := m.backupProfile(ctx, clock)
	if err != nil {
		return err
	}
	_, settings, err := m.prepareSyncSettingsMessages(clock, true)
	if err != nil {
		return err
	}
	woAccountsToBackup, err := m.backupWatchOnlyAccounts()
	if err != nil {
		return err
	}

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
	fullBackup.Settings = append(fullBackup.Settings, settings...)
	for _, d := range woAccountsToBackup {
		fullBackup.WatchOnlyAccounts = append(fullBackup.WatchOnlyAccounts, d.WatchOnlyAccount)
	}

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

func (m *Messenger) ImportLocalBackupFile(filePath string) (*MessengerResponse, error) {
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
	errs := m.handleBackup(
		&state,
		&backupMessage,
	)
	if err != nil {
		return nil, errors.Join(errs...)
	}

	return m.saveDataAndPrepareResponse(&state)
}
