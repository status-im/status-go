package pairing

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
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
	_, err = messenger.SendPairInstallation(context.TODO(), "", rawMessageCollector.dispatchMessage)
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

func (s *SyncRawMessageHandler) HandleRawMessage(
	accountPayload *AccountPayload,
	createAccountRequest *requests.CreateAccount,
	deviceType string,
	rmp *RawMessagesPayload) (err error) {

	activeAccount, _ := s.backend.GetActiveAccount()
	if activeAccount == nil { // not login yet
		err = s.login(accountPayload, createAccountRequest, rmp)
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

func (s *SyncRawMessageHandler) login(accountPayload *AccountPayload, createAccountRequest *requests.CreateAccount, rmp *RawMessagesPayload) error {
	account := accountPayload.multiaccount
	installationID := multidevice.GenerateInstallationID()
	nodeConfig, err := api.DefaultNodeConfig(installationID, createAccountRequest)
	if err != nil {
		return err
	}

	keystoreRelativePath, err := s.backend.InitKeyStoreDirWithAccount(nodeConfig.RootDataDir, account.KeyUID)
	nodeConfig.KeyStoreDir = keystoreRelativePath

	if err != nil {
		return err
	}

	var chatKey *ecdsa.PrivateKey
	if accountPayload.chatKey != "" {
		chatKeyHex := strings.Trim(accountPayload.chatKey, "0x")
		chatKey, err = ethcrypto.HexToECDSA(chatKeyHex)
		if err != nil {
			return err
		}
	}

	if accountPayload.exist {
		return s.backend.StartNodeWithAccount(*account, accountPayload.password, nodeConfig, chatKey)
	}

	// Override some of received settings
	rmp.setting.DeviceName = createAccountRequest.DeviceName
	rmp.setting.InstallationID = installationID
	rmp.setting.CurrentNetwork = api.DefaultCurrentNetwork

	return s.backend.StartNodeWithAccountAndInitialConfig(
		*account,
		accountPayload.password,
		*rmp.setting,
		nodeConfig,
		rmp.profileKeypair.Accounts,
		chatKey,
	)
}
