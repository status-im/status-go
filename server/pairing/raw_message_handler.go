package pairing

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
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

	return proto.Marshal(syncRawMessage)
}

func (s *SyncRawMessageHandler) HandleRawMessage(account *multiaccounts.Account, password string, nodeConfig *params.NodeConfig, settingCurrentNetwork string, payload []byte) error {
	rawMessages, subAccounts, setting, err := s.unmarshalSyncRawMessage(payload)
	if err != nil {
		return err
	}

	s.backend.UpdateRootDataDir(nodeConfig.RootDataDir)
	// because client don't know keyUID before received data, we need help client to update keystore dir
	newKeystoreDir := filepath.Join(nodeConfig.KeyStoreDir, account.KeyUID)
	nodeConfig.KeyStoreDir = newKeystoreDir
	accountManager := s.backend.AccountManager()
	err = accountManager.InitKeystore(filepath.Join(nodeConfig.RootDataDir, newKeystoreDir))
	if err != nil {
		return err
	}
	setting.InstallationID = nodeConfig.ShhextConfig.InstallationID
	setting.CurrentNetwork = settingCurrentNetwork

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

func (s *SyncRawMessageHandler) unmarshalSyncRawMessage(payload []byte) ([]*protobuf.RawMessage, []*accounts.Account, *settings.Settings, error) {
	var (
		syncRawMessage protobuf.SyncRawMessage
		subAccounts    []*accounts.Account
		setting        *settings.Settings
	)
	err := proto.Unmarshal(payload, &syncRawMessage)
	if err != nil {
		return nil, nil, nil, err
	}
	err = json.Unmarshal(syncRawMessage.SubAccountsJsonBytes, &subAccounts)
	if err != nil {
		return nil, nil, nil, err
	}
	err = json.Unmarshal(syncRawMessage.SettingsJsonBytes, &setting)
	if err != nil {
		return nil, nil, nil, err
	}
	return syncRawMessage.RawMessages, subAccounts, setting, nil
}
