package pairing

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
)

type PayloadReceiver interface {
	PayloadLocker

	// Receive accepts data from an inbound source into the PayloadReceiver's state
	Receive(data []byte) error

	// Received returns a decrypted and parsed payload from an inbound source
	Received() []byte
}

type PayloadStorer interface {
	Store() error
}

type BasePayloadReceiver struct {
	*PayloadLockPayload
	*PayloadReceived

	encryptor    *PayloadEncryptor
	unmarshaller ProtobufUnmarshaller
	storer       PayloadStorer

	receiveCallback func()
}

func NewBasePayloadReceiver(e *PayloadEncryptor, um ProtobufUnmarshaller, s PayloadStorer, callback func()) *BasePayloadReceiver {
	return &BasePayloadReceiver{
		PayloadLockPayload: &PayloadLockPayload{e},
		PayloadReceived:    &PayloadReceived{e},
		encryptor:          e,
		unmarshaller:       um,
		storer:             s,
		receiveCallback:    callback,
	}
}

// Receive takes a []byte representing raw data, parses and stores the data
func (bpr *BasePayloadReceiver) Receive(data []byte) error {
	err := bpr.encryptor.decrypt(data)
	if err != nil {
		return err
	}

	err = bpr.unmarshaller.UnmarshalProtobuf(bpr.Received())
	if err != nil {
		return err
	}

	err = bpr.storer.Store()
	if err != nil {
		return err
	}

	if bpr.receiveCallback != nil {
		bpr.receiveCallback()
	}

	return nil
}

/*
|--------------------------------------------------------------------------
| AccountPayload
|--------------------------------------------------------------------------
|
| AccountPayloadReceiver, AccountPayloadStorer and AccountPayloadMarshaller
|
*/

// NewAccountPayloadReceiver generates a new and initialised AccountPayload flavoured BasePayloadReceiver
// AccountPayloadReceiver is responsible for the whole receive and store cycle of an AccountPayload
func NewAccountPayloadReceiver(e *PayloadEncryptor, p *AccountPayload, config *ReceiverConfig, logger *zap.Logger) (*BasePayloadReceiver, error) {
	l := logger.Named("AccountPayloadManager")
	l.Debug("fired", zap.Any("config", config))

	e = e.Renew()

	aps, err := NewAccountPayloadStorer(p, config)
	if err != nil {
		return nil, err
	}

	return NewBasePayloadReceiver(e, NewPairingPayloadMarshaller(p, l), aps,
		func() {
			data := AccountData{Account: p.multiaccount, Password: p.password}
			signal.SendLocalPairingEvent(Event{Type: EventReceivedAccount, Action: ActionPairingAccount, Data: data})
		},
	), nil
}

// AccountPayloadStorer is responsible for parsing, validating and storing AccountPayload data
type AccountPayloadStorer struct {
	*AccountPayload
	multiaccountsDB *multiaccounts.Database

	keystorePath   string
	kdfIterations  int
	loggedInKeyUID string
}

func NewAccountPayloadStorer(p *AccountPayload, config *ReceiverConfig) (*AccountPayloadStorer, error) {
	ppr := &AccountPayloadStorer{
		AccountPayload: p,
	}

	if config == nil {
		return ppr, nil
	}

	ppr.multiaccountsDB = config.DB
	ppr.kdfIterations = config.KDFIterations
	ppr.keystorePath = config.KeystorePath
	ppr.loggedInKeyUID = config.LoggedInKeyUID
	return ppr, nil
}

func (aps *AccountPayloadStorer) Store() error {
	keyUID := aps.multiaccount.KeyUID
	if aps.loggedInKeyUID != "" && aps.loggedInKeyUID != keyUID {
		return ErrLoggedInKeyUIDConflict
	}
	if aps.loggedInKeyUID == keyUID {
		// skip storing keys if user is logged in with the same key
		return nil
	}

	err := validateKeys(aps.keys, aps.password)
	if err != nil {
		return err
	}

	if err = aps.storeKeys(aps.keystorePath); err != nil && err != ErrKeyFileAlreadyExists {
		return err
	}

	// skip storing multiaccount if key already exists
	if err == ErrKeyFileAlreadyExists {
		aps.exist = true
		aps.multiaccount, err = aps.multiaccountsDB.GetAccount(keyUID)
		if err != nil {
			return err
		}
		return nil
	}
	return aps.storeMultiAccount()
}

func (aps *AccountPayloadStorer) storeKeys(keyStorePath string) error {
	if keyStorePath == "" {
		return fmt.Errorf("keyStorePath can not be empty")
	}

	_, lastDir := filepath.Split(keyStorePath)

	// If lastDir == keystoreDir we presume we need to create the rest of the keystore path
	// else we presume the provided keystore is valid
	if lastDir == keystoreDir {
		if aps.multiaccount == nil || aps.multiaccount.KeyUID == "" {
			return fmt.Errorf("no known Key UID")
		}
		keyStorePath = filepath.Join(keyStorePath, aps.multiaccount.KeyUID)
		_, err := os.Stat(keyStorePath)
		if os.IsNotExist(err) {
			err := os.MkdirAll(keyStorePath, 0700)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			return ErrKeyFileAlreadyExists
		}
	}

	for name, data := range aps.keys {
		err := ioutil.WriteFile(filepath.Join(keyStorePath, name), data, 0600)
		if err != nil {
			writeErr := fmt.Errorf("failed to write key to path '%s' : %w", filepath.Join(keyStorePath, name), err)
			// If we get an error on any of the key files attempt to revert
			err := emptyDir(keyStorePath)
			if err != nil {
				// If we get an error when trying to empty the dir combine the write error and empty error
				emptyDirErr := fmt.Errorf("failed to revert and cleanup storeKeys : %w", err)
				return multierr.Combine(writeErr, emptyDirErr)
			}
			return writeErr
		}
	}
	return nil
}

func (aps *AccountPayloadStorer) storeMultiAccount() error {
	aps.multiaccount.KDFIterations = aps.kdfIterations
	return aps.multiaccountsDB.SaveAccount(*aps.multiaccount)
}

/*
|--------------------------------------------------------------------------
| RawMessagePayload
|--------------------------------------------------------------------------
|
| RawMessagePayloadReceiver and RawMessageStorer
|
*/

// NewRawMessagePayloadReceiver generates a new and initialised RawMessagesPayload flavoured BasePayloadReceiver
// RawMessagePayloadReceiver is responsible for the whole receive and store cycle of a RawMessagesPayload
func NewRawMessagePayloadReceiver(accountPayload *AccountPayload, e *PayloadEncryptor, backend *api.GethStatusBackend, config *ReceiverConfig) *BasePayloadReceiver {
	e = e.Renew()
	payload := NewRawMessagesPayload()

	return NewBasePayloadReceiver(e,
		NewRawMessagePayloadMarshaller(payload),
		NewRawMessageStorer(backend, payload, accountPayload, config), nil)
}

type RawMessageStorer struct {
	payload               *RawMessagesPayload
	syncRawMessageHandler *SyncRawMessageHandler
	accountPayload        *AccountPayload
	nodeConfig            *params.NodeConfig
	settingCurrentNetwork string
	deviceType            string
}

func NewRawMessageStorer(backend *api.GethStatusBackend, payload *RawMessagesPayload, accountPayload *AccountPayload, config *ReceiverConfig) *RawMessageStorer {
	return &RawMessageStorer{
		syncRawMessageHandler: NewSyncRawMessageHandler(backend),
		payload:               payload,
		accountPayload:        accountPayload,
		nodeConfig:            config.NodeConfig,
		settingCurrentNetwork: config.SettingCurrentNetwork,
		deviceType:            config.DeviceType,
	}
}

func (r *RawMessageStorer) Store() error {
	if r.accountPayload == nil || r.accountPayload.multiaccount == nil {
		return fmt.Errorf("no known multiaccount when storing raw messages")
	}
	return r.syncRawMessageHandler.HandleRawMessage(r.accountPayload, r.nodeConfig, r.settingCurrentNetwork, r.deviceType, r.payload)
}

/*
|--------------------------------------------------------------------------
| InstallationPayload
|--------------------------------------------------------------------------
|
| InstallationPayloadReceiver and InstallationPayloadStorer
|
*/

// NewInstallationPayloadReceiver generates a new and initialised InstallationPayload flavoured BasePayloadReceiver
// InstallationPayloadReceiver is responsible for the whole receive and store cycle of a RawMessagesPayload specifically
// for sending / requesting installation data from the Receiver device.
func NewInstallationPayloadReceiver(e *PayloadEncryptor, backend *api.GethStatusBackend, deviceType string) *BasePayloadReceiver {
	e = e.Renew()
	payload := NewRawMessagesPayload()

	return NewBasePayloadReceiver(e,
		NewRawMessagePayloadMarshaller(payload),
		NewInstallationPayloadStorer(backend, payload, deviceType), nil)
}

type InstallationPayloadStorer struct {
	payload               *RawMessagesPayload
	syncRawMessageHandler *SyncRawMessageHandler
	deviceType            string
	backend               *api.GethStatusBackend
}

func NewInstallationPayloadStorer(backend *api.GethStatusBackend, payload *RawMessagesPayload, deviceType string) *InstallationPayloadStorer {
	return &InstallationPayloadStorer{
		payload:               payload,
		syncRawMessageHandler: NewSyncRawMessageHandler(backend),
		deviceType:            deviceType,
		backend:               backend,
	}
}

func (r *InstallationPayloadStorer) Store() error {
	messenger := r.backend.Messenger()
	if messenger == nil {
		return fmt.Errorf("messenger is nil when invoke InstallationPayloadRepository#Store()")
	}
	err := messenger.SetInstallationDeviceType(r.deviceType)
	if err != nil {
		return err
	}

	installations := GetMessengerInstallationsMap(messenger)

	err = messenger.HandleSyncRawMessages(r.payload.rawMessages)

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

/*
|--------------------------------------------------------------------------
| PayloadReceivers
|--------------------------------------------------------------------------
|
| Funcs for all PayloadReceivers AccountPayloadReceiver, RawMessagePayloadReceiver and InstallationPayloadMounter
|
*/

func NewPayloadReceivers(logger *zap.Logger, pe *PayloadEncryptor, backend *api.GethStatusBackend, config *ReceiverConfig) (PayloadReceiver, PayloadReceiver, PayloadMounterReceiver, error) {
	// A new SHARED AccountPayload
	p := new(AccountPayload)

	ar, err := NewAccountPayloadReceiver(pe, p, config, logger)
	if err != nil {
		return nil, nil, nil, err
	}
	rmr := NewRawMessagePayloadReceiver(p, pe, backend, config)
	imr := NewInstallationPayloadMounterReceiver(pe, backend, config.DeviceType)
	return ar, rmr, imr, nil
}
