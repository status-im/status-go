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

/*
|--------------------------------------------------------------------------
| AccountPayload
|--------------------------------------------------------------------------
|
| AccountPayloadReceiver, AccountPayloadStorer and AccountPayloadMarshaller
|
*/

// AccountPayloadReceiver is responsible for the whole lifecycle of a AccountPayload
type AccountPayloadReceiver struct {
	logger                   *zap.Logger
	accountPayload           *AccountPayload
	encryptor                *PayloadEncryptor
	accountPayloadMarshaller *AccountPayloadMarshaller
	accountStorer            *AccountPayloadStorer
}

// NewAccountPayloadReceiver generates a new and initialised AccountPayloadManager
func NewAccountPayloadReceiver(encryptor *PayloadEncryptor, config *ReceiverConfig, logger *zap.Logger) (*AccountPayloadReceiver, error) {
	l := logger.Named("AccountPayloadManager")
	l.Debug("fired", zap.Any("config", config))

	// A new SHARED AccountPayload
	p := new(AccountPayload)
	accountPayloadRepository, err := NewAccountPayloadStorer(p, config)
	if err != nil {
		return nil, err
	}

	return &AccountPayloadReceiver{
		logger:                   l,
		accountPayload:           p,
		encryptor:                encryptor.Renew(),
		accountPayloadMarshaller: NewPairingPayloadMarshaller(p, l),
		accountStorer:            accountPayloadRepository,
	}, nil
}

// Receive takes a []byte representing raw data, parses and stores the data
func (apr *AccountPayloadReceiver) Receive(data []byte) error {
	l := apr.logger.Named("Receive()")
	l.Debug("fired")

	err := apr.encryptor.decrypt(data)
	if err != nil {
		return err
	}
	l.Debug("after Decrypt")

	err = apr.accountPayloadMarshaller.UnmarshalProtobuf(apr.Received())
	if err != nil {
		return err
	}
	l.Debug(
		"after UnmarshalProtobuf",
		zap.Any("accountPayloadMarshaller.accountPayloadMarshaller.keys", apr.accountPayloadMarshaller.keys),
		zap.Any("accountPayloadMarshaller.accountPayloadMarshaller.multiaccount", apr.accountPayloadMarshaller.multiaccount),
		zap.String("accountPayloadMarshaller.accountPayloadMarshaller.password", apr.accountPayloadMarshaller.password),
		zap.Binary("accountPayloadMarshaller.Received()", apr.Received()),
	)

	signal.SendLocalPairingEvent(Event{Type: EventReceivedAccount, Action: ActionPairingAccount, Data: apr.accountPayload.multiaccount})

	return apr.accountStorer.Store()
}

func (apr *AccountPayloadReceiver) Received() []byte {
	return apr.encryptor.getDecrypted()
}

func (apr *AccountPayloadReceiver) LockPayload() {
	apr.encryptor.lockPayload()
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

type RawMessagePayloadReceiver struct {
	logger    *zap.Logger
	encryptor *PayloadEncryptor
	storer    *RawMessageStorer
}

func NewRawMessagePayloadReceiver(logger *zap.Logger, accountPayload *AccountPayload, encryptor *PayloadEncryptor, backend *api.GethStatusBackend, config *ReceiverConfig) *RawMessagePayloadReceiver {
	l := logger.Named("RawMessagePayloadManager")
	return &RawMessagePayloadReceiver{
		logger:    l,
		encryptor: encryptor.Renew(),
		storer:    NewRawMessageStorer(backend, accountPayload, config),
	}
}

func (r *RawMessagePayloadReceiver) Receive(data []byte) error {
	err := r.encryptor.decrypt(data)
	if err != nil {
		return err
	}
	r.storer.payload = r.Received()
	return r.storer.Store()
}

func (r *RawMessagePayloadReceiver) Received() []byte {
	return r.encryptor.getDecrypted()
}

func (r *RawMessagePayloadReceiver) LockPayload() {
	r.encryptor.lockPayload()
}

type RawMessageStorer struct {
	payload               []byte
	syncRawMessageHandler *SyncRawMessageHandler
	accountPayload        *AccountPayload
	nodeConfig            *params.NodeConfig
	settingCurrentNetwork string
	deviceType            string
}

func NewRawMessageStorer(backend *api.GethStatusBackend, accountPayload *AccountPayload, config *ReceiverConfig) *RawMessageStorer {
	return &RawMessageStorer{
		syncRawMessageHandler: NewSyncRawMessageHandler(backend),
		payload:               make([]byte, 0),
		accountPayload:        accountPayload,
		nodeConfig:            config.NodeConfig,
		settingCurrentNetwork: config.SettingCurrentNetwork,
		deviceType:            config.DeviceType,
	}
}

func (r *RawMessageStorer) Store() error {
	accountPayload := r.accountPayload
	if accountPayload == nil || accountPayload.multiaccount == nil {
		return fmt.Errorf("no known multiaccount when storing raw messages")
	}
	return r.syncRawMessageHandler.HandleRawMessage(accountPayload, r.nodeConfig, r.settingCurrentNetwork, r.deviceType, r.payload)
}

/*
|--------------------------------------------------------------------------
| InstallationPayload
|--------------------------------------------------------------------------
|
| InstallationPayloadReceiver and InstallationPayloadStorer
|
*/

type InstallationPayloadReceiver struct {
	logger    *zap.Logger
	encryptor *PayloadEncryptor
	storer    *InstallationPayloadStorer
}

func NewInstallationPayloadReceiver(logger *zap.Logger, encryptor *PayloadEncryptor, backend *api.GethStatusBackend, deviceType string) *InstallationPayloadReceiver {
	l := logger.Named("InstallationPayloadManager")
	return &InstallationPayloadReceiver{
		logger:    l,
		encryptor: encryptor.Renew(),
		storer:    NewInstallationPayloadStorer(backend, deviceType),
	}
}

func (i *InstallationPayloadReceiver) Receive(data []byte) error {
	err := i.encryptor.decrypt(data)
	if err != nil {
		return err
	}
	i.storer.payload = i.encryptor.getDecrypted()
	return i.storer.Store()
}

func (i *InstallationPayloadReceiver) Received() []byte {
	return i.encryptor.getDecrypted()
}

func (i *InstallationPayloadReceiver) LockPayload() {
	i.encryptor.lockPayload()
}

type InstallationPayloadStorer struct {
	payload               []byte
	syncRawMessageHandler *SyncRawMessageHandler
	deviceType            string
	backend               *api.GethStatusBackend
}

func NewInstallationPayloadStorer(backend *api.GethStatusBackend, deviceType string) *InstallationPayloadStorer {
	return &InstallationPayloadStorer{
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
	rawMessages, _, _, err := r.syncRawMessageHandler.unmarshalSyncRawMessage(r.payload)
	if err != nil {
		return err
	}
	err = messenger.SetInstallationDeviceType(r.deviceType)
	if err != nil {
		return err
	}
	return messenger.HandleSyncRawMessages(rawMessages)
}

/*
|--------------------------------------------------------------------------
| PayloadReceivers
|--------------------------------------------------------------------------
|
| Funcs for all PayloadReceivers AccountPayloadReceiver, RawMessagePayloadReceiver and InstallationPayloadMounter
|
*/

func NewPayloadReceivers(logger *zap.Logger, pe *PayloadEncryptor, backend *api.GethStatusBackend, config *ReceiverConfig) (*AccountPayloadReceiver, *RawMessagePayloadReceiver, *InstallationPayloadMounterReceiver, error) {
	ar, err := NewAccountPayloadReceiver(pe, config, logger)
	if err != nil {
		return nil, nil, nil, err
	}
	rmr := NewRawMessagePayloadReceiver(logger, ar.accountPayload, pe, backend, config)
	imr := NewInstallationPayloadMounterReceiver(logger, pe, backend, config.DeviceType)
	return ar, rmr, imr, nil
}
