package pairing

import (
	"context"
	"encoding/json"
	"fmt"
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

	//TODO root data dir should be passed from client, following is a temporary solution
	nodeConfig.RootDataDir = filepath.Dir(keystorePath)
	nodeConfig.KeyStoreDir = filepath.Join(filepath.Base(keystorePath), account.KeyUID)
	installationID := uuid.New().String()
	nodeConfig.ShhextConfig.InstallationID = installationID
	setting.InstallationID = installationID

	//TODO we need a better way(e.g. pass from client when doing local pair?) to handle this, following is a temporary solution
	nodeConfig.LogDir = nodeConfig.RootDataDir
	nodeConfig.LogFile = "geth.log"

	err = s.backend.StartNodeWithAccountAndInitialConfig(*account, password, *setting, nodeConfig, subAccounts)
	if err != nil {
		return err
	}

	messenger := s.backend.Messenger()
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
