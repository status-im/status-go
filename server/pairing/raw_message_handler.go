package pairing

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/status-im/status-go/logutils"
	"go.uber.org/zap"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/status-im/status-go/multiaccounts/settings"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/protobuf"
)

type SyncRawMessageHandler struct {
	backend *api.GethStatusBackend
}

func NewSyncRawMessageHandler(backend *api.GethStatusBackend) *SyncRawMessageHandler {
	return &SyncRawMessageHandler{backend: backend}
}

func (s *SyncRawMessageHandler) PrepareRawMessage(keyUID string) ([]byte, error) {
	messenger := s.backend.Messenger()
	if messenger == nil {
		return nil, fmt.Errorf("messenger is nil when PrepareRawMessage")
	}

	currentAccount, err := s.backend.GetActiveAccount()
	if err != nil {
		return nil, err
	}
	if keyUID != currentAccount.KeyUID {
		return nil, fmt.Errorf("keyUID not equal")
	}

	messenger.SetLocalPairing(true)
	defer func() {
		messenger.SetLocalPairing(false)
	}()
	rawMessageCollector := new(RawMessageCollector)
	err = messenger.SyncDevices(context.TODO(), currentAccount.Name, currentAccount.Identicon, rawMessageCollector.dispatchMessage)
	if err != nil {
		return nil, err
	}
	syncRawMessage := new(protobuf.SyncRawMessage)
	for _, m := range rawMessageCollector.getRawMessages() {
		rawMessage := new(protobuf.RawMessage)
		rawMessage.Payload = m.Payload
		rawMessage.MessageType = m.MessageType
		syncRawMessage.RawMessages = append(syncRawMessage.RawMessages, rawMessage)
	}

	accountService := s.backend.StatusNode().AccountService()
	var (
		subAccounts []*accounts.Account
		setting     settings.Settings
	)
	subAccounts, err = accountService.GetAccountsByKeyUID(keyUID)
	if err != nil {
		return nil, err
	}
	syncRawMessage.SubAccountsJsonBytes, err = json.Marshal(subAccounts)
	if err != nil {
		return nil, err
	}
	setting, err = accountService.GetSettings()
	if err != nil {
		return nil, err
	}
	syncRawMessage.SettingsJsonBytes, err = json.Marshal(setting)
	if err != nil {
		return nil, err
	}

	nodeConfig := s.backend.StatusNode().Config()
	dataDir := nodeConfig.DataDir
	disabledDataDir := nodeConfig.ShhextConfig.BackupDisabledDataDir
	defer func() {
		// restore data dir
		nodeConfig.DataDir = dataDir
		nodeConfig.ShhextConfig.BackupDisabledDataDir = disabledDataDir
	}()
	nodeConfig.DataDir = strings.Replace(dataDir, nodeConfig.RootDataDir, "", 1)
	nodeConfig.ShhextConfig.BackupDisabledDataDir = strings.Replace(disabledDataDir, nodeConfig.RootDataDir, "", 1)
	if syncRawMessage.NodeConfigJsonBytes, err = json.Marshal(nodeConfig); err != nil {
		return nil, err
	}
	return proto.Marshal(syncRawMessage)
}

func (s *SyncRawMessageHandler) HandleRawMessage(account *multiaccounts.Account, password, keystorePath string, payload []byte) error {
	rawMessages, subAccounts, setting, nodeConfig, err := s.unmarshalSyncRawMessage(payload)
	if err != nil {
		return err
	}

	newKeystoreDir := filepath.Join(keystorePath, account.KeyUID)
	accountManager := s.backend.AccountManager()
	err = accountManager.InitKeystore(newKeystoreDir)
	if err != nil {
		return err
	}

	nodeConfig.RootDataDir = filepath.Dir(keystorePath)
	nodeConfig.KeyStoreDir = filepath.Join(filepath.Base(keystorePath), account.KeyUID)
	installationID := uuid.New().String()
	nodeConfig.ShhextConfig.InstallationID = installationID
	setting.InstallationID = installationID

	logFile := "local_pair.log"
	nodeConfig.LogDir = nodeConfig.RootDataDir
	nodeConfig.LogFile = logFile
	nodeConfig.LogLevel = "DEBUG"
	nodeConfig.LogEnabled = true
	nodeConfig.LogMobileSystem = false
	nodeConfig.LogToStderr = true
	logSettings := logutils.LogSettings{
		Enabled: true,
		MobileSystem: false,
		Level:  "DEBUG",
		File:   filepath.Join(nodeConfig.LogDir, nodeConfig.LogFile),
		MaxSize: 0,
		MaxBackups: 0,
		CompressRotated: false,
	}
	if err = logutils.OverrideRootLogWithConfig(logSettings, false); err != nil {
		return err
	}
	logger := logutils.ZapLogger()
	logger.Info("HandleRawMessage, done OverrideRootLogWithConfig!")
	err = s.backend.StartNodeWithAccountAndInitialConfig(*account, password, *setting, nodeConfig, subAccounts)
	logger.Info("HandleRawMessage, StartNodeWithAccountAndInitialConfig!", zap.Error(err))
	if err != nil {
		return err
	}
	logger.Info("HandleRawMessage, done StartNodeWithAccountAndInitialConfig!")
	messenger := s.backend.Messenger()
	logger.Info("HandleRawMessage, backend.Messenger()", zap.Bool("messenger is nil", messenger == nil))
	if messenger == nil {
		return fmt.Errorf("messenger is nil when HandleRawMessage")
	}
	return messenger.HandleSyncRawMessages(rawMessages)
}

func (s *SyncRawMessageHandler) unmarshalSyncRawMessage(payload []byte) ([]*protobuf.RawMessage, []*accounts.Account, *settings.Settings, *params.NodeConfig, error) {
	var (
		syncRawMessage protobuf.SyncRawMessage
		subAccounts    []*accounts.Account
		setting        *settings.Settings
		nodeConfig     *params.NodeConfig
	)
	err := proto.Unmarshal(payload, &syncRawMessage)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = json.Unmarshal(syncRawMessage.SubAccountsJsonBytes, &subAccounts)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = json.Unmarshal(syncRawMessage.SettingsJsonBytes, &setting)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = json.Unmarshal(syncRawMessage.NodeConfigJsonBytes, &nodeConfig)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return syncRawMessage.RawMessages, subAccounts, setting, nodeConfig, nil
}
