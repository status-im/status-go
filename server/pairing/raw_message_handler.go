package pairing

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/protobuf"

	"github.com/status-im/status-go/signal"
)

type SyncRawMessageHandler struct {
	backend *api.GethStatusBackend
}

func NewSyncRawMessageHandler(backend *api.GethStatusBackend) *SyncRawMessageHandler {
	return &SyncRawMessageHandler{backend: backend}
}

func (s *SyncRawMessageHandler) CollectInstallationData(rawMessageCollector *RawMessageCollector, deviceType string) error {
	// TODO Could this function be part of the installation data exchange flow?
	//  https://github.com/status-im/status-go/issues/3304
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

func (s *SyncRawMessageHandler) PrepareRawMessage(keyUID, deviceType string) (rm []*protobuf.RawMessage, kp *accounts.Keypair, syncSettings *settings.Settings, err error) {
	syncSettings = new(settings.Settings)
	messenger := s.backend.Messenger()
	if messenger == nil {
		return nil, nil, nil, fmt.Errorf("messenger is nil when PrepareRawMessage")
	}

	currentAccount, err := s.backend.GetActiveAccount()
	if err != nil {
		return
	}
	if keyUID != currentAccount.KeyUID {
		return nil, nil, nil, fmt.Errorf("keyUID not equal")
	}

	messenger.SetLocalPairing(true)
	defer func() {
		messenger.SetLocalPairing(false)
	}()
	rawMessageCollector := new(RawMessageCollector)
	err = messenger.SyncDevices(context.TODO(), currentAccount.Name, currentAccount.Identicon, rawMessageCollector.dispatchMessage)
	if err != nil {
		return
	}

	err = s.CollectInstallationData(rawMessageCollector, deviceType)
	if err != nil {
		return
	}

	rsm := rawMessageCollector.convertToSyncRawMessage()
	rm = rsm.RawMessages

	accountService := s.backend.StatusNode().AccountService()

	kp, err = accountService.GetKeypairByKeyUID(keyUID)
	if err != nil {
		return
	}
	*syncSettings, err = accountService.GetSettings()
	if err != nil {
		return
	}

	return
}

func (s *SyncRawMessageHandler) HandleRawMessage(accountPayload *AccountPayload, nodeConfig *params.NodeConfig, settingCurrentNetwork, deviceType string, rmp *RawMessagesPayload) (err error) {
	account := accountPayload.multiaccount

	activeAccount, _ := s.backend.GetActiveAccount()
	if activeAccount == nil { // not login yet
		s.backend.UpdateRootDataDir(nodeConfig.RootDataDir)
		// because client don't know keyUID before received data, we need help client to update keystore dir
		keystoreDir := filepath.Join(nodeConfig.KeyStoreDir, account.KeyUID)
		nodeConfig.KeyStoreDir = keystoreDir
		if accountPayload.exist {
			if len(accountPayload.chatKey) == 0 {
				err = s.backend.StartNodeWithAccount(*account, accountPayload.password, nodeConfig)
			} else {
				err = s.backend.StartNodeWithKey(*account, accountPayload.password, accountPayload.chatKey)
			}
		} else {
			accountManager := s.backend.AccountManager()
			err = accountManager.InitKeystore(filepath.Join(nodeConfig.RootDataDir, keystoreDir))
			if err != nil {
				return err
			}
			rmp.setting.InstallationID = nodeConfig.ShhextConfig.InstallationID
			rmp.setting.CurrentNetwork = settingCurrentNetwork

			if len(accountPayload.chatKey) == 0 {
				err = s.backend.StartNodeWithAccountAndInitialConfig(*account, accountPayload.password, *rmp.setting, nodeConfig, rmp.profileKeypair.Accounts)
			} else {
				err = s.backend.SaveAccountAndStartNodeWithKey(*account, accountPayload.password, *rmp.setting, nodeConfig, rmp.profileKeypair.Accounts, accountPayload.chatKey)
			}
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

	installations := GetMessengerInstallationsMap(messenger)

	err = messenger.HandleSyncRawMessages(rmp.rawMessages)

	if err != nil {
		return err
	}

	if newInstallation := FindNewInstallations(messenger, installations); newInstallation != nil {
		signal.SendLocalPairingEvent(Event{
			Type:   EventReceivedInstallation,
			Action: ActionPairingInstallation,
			Data:   newInstallation})
	}

	return nil
}
