package pairing

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/api"
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

func (s *SyncRawMessageHandler) CollectInstallationData(rawMessageCollector *RawMessageCollector, deviceType string) error {
	// TODO Could this function be part of the installation data exchange flow?
	messenger := s.backend.Messenger()
	if messenger == nil {
		return fmt.Errorf("messenger is nil when CollectInstallationData")
	}
	err := messenger.SetInstallationDeviceType(deviceType)
	if err != nil {
		return err
	}
	_, err = messenger.SendPairInstallation(context.TODO(), rawMessageCollector.dispatchMessage)
	return err
}

func (s *SyncRawMessageHandler) PrepareRawMessage(keyUID, deviceType string) ([]byte, error) {
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

	err = s.CollectInstallationData(rawMessageCollector, deviceType)
	if err != nil {
		return nil, err
	}

	syncRawMessage := rawMessageCollector.convertToSyncRawMessage()

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

func (s *SyncRawMessageHandler) HandleRawMessage(accountPayload *AccountPayload, nodeConfig *params.NodeConfig, settingCurrentNetwork, deviceType string, rawMessagePayload []byte) error {
	account := accountPayload.multiaccount
	rawMessages, subAccounts, setting, err := s.unmarshalSyncRawMessage(rawMessagePayload)
	if err != nil {
		return err
	}

	activeAccount, _ := s.backend.GetActiveAccount()
	if activeAccount == nil { // not login yet
		s.backend.UpdateRootDataDir(nodeConfig.RootDataDir)
		// because client don't know keyUID before received data, we need help client to update keystore dir
		keystoreDir := filepath.Join(nodeConfig.KeyStoreDir, account.KeyUID)
		nodeConfig.KeyStoreDir = keystoreDir
		if accountPayload.exist {
			err = s.backend.StartNodeWithAccount(*account, accountPayload.password, nodeConfig)
		} else {
			accountManager := s.backend.AccountManager()
			err = accountManager.InitKeystore(filepath.Join(nodeConfig.RootDataDir, keystoreDir))
			if err != nil {
				return err
			}
			setting.InstallationID = nodeConfig.ShhextConfig.InstallationID
			setting.CurrentNetwork = settingCurrentNetwork

			err = s.backend.StartNodeWithAccountAndInitialConfig(*account, accountPayload.password, *setting, nodeConfig, subAccounts)
		}
		if err != nil {
			return err
		}
	}

	messenger := s.backend.Messenger()
	if messenger == nil {
		return fmt.Errorf("messenger is nil when HandleRawMessage")
	}
	err = messenger.SetInstallationDeviceType(deviceType)
	if err != nil {
		return err
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
	if syncRawMessage.SubAccountsJsonBytes != nil {
		err = json.Unmarshal(syncRawMessage.SubAccountsJsonBytes, &subAccounts)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if syncRawMessage.SettingsJsonBytes != nil {
		err = json.Unmarshal(syncRawMessage.SettingsJsonBytes, &setting)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return syncRawMessage.RawMessages, subAccounts, setting, nil
}
