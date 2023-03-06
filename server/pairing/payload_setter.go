package pairing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/keystore"
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
	StoreToSource() error
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
func NewAccountPayloadReceiver(encryptor PayloadEncryptor, config *ReceiverConfig, logger *zap.Logger) (*AccountPayloadReceiver, error) {
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
		encryptor:                &encryptor,
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

	return apr.accountStorer.StoreToSource()
}

func (apr *AccountPayloadReceiver) Received() []byte {
	if apr.encryptor.payload.locked {
		return nil
	}
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
	keyUID         string
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

func (aps *AccountPayloadStorer) StoreToSource() error {
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

	// If lastDir == "keystore" we presume we need to create the rest of the keystore path
	// else we presume the provided keystore is valid
	if lastDir == "keystore" {
		if aps.multiaccount == nil || aps.multiaccount.KeyUID == "" {
			return fmt.Errorf("no known Key UID")
		}
		keyStorePath = filepath.Join(keyStorePath, aps.multiaccount.KeyUID)
		_, err := os.Stat(keyStorePath)
		if os.IsNotExist(err) {
			err := os.MkdirAll(keyStorePath, 0777)
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
		accountKey := new(keystore.EncryptedKeyJSONV3)
		if err := json.Unmarshal(data, &accountKey); err != nil {
			return fmt.Errorf("failed to read key file: %s", err)
		}

		if len(accountKey.Address) != 40 {
			return fmt.Errorf("account key address has invalid length '%s'", accountKey.Address)
		}

		err := ioutil.WriteFile(filepath.Join(keyStorePath, name), data, 0600)
		if err != nil {
			return err
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

func NewRawMessagePayloadReceiver(logger *zap.Logger, accountPayload *AccountPayload, encryptor PayloadEncryptor, backend *api.GethStatusBackend, config *ReceiverConfig) *RawMessagePayloadReceiver {
	l := logger.Named("RawMessagePayloadManager")
	return &RawMessagePayloadReceiver{
		logger:    l,
		encryptor: &encryptor,
		storer:    NewRawMessageStorer(backend, accountPayload, config),
	}
}

func (r *RawMessagePayloadReceiver) Receive(data []byte) error {
	err := r.encryptor.decrypt(data)
	if err != nil {
		return err
	}
	r.storer.payload = r.Received()
	return r.storer.StoreToSource()
}

func (r *RawMessagePayloadReceiver) Received() []byte {
	if r.encryptor.payload.locked {
		return nil
	}
	return r.encryptor.getDecrypted()
}

func (r *RawMessagePayloadReceiver) LockPayload() {
	r.encryptor.lockPayload()
}

func (r *RawMessagePayloadReceiver) ResetPayload() {
	r.storer.payload = make([]byte, 0)
	r.encryptor.resetPayload()
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

func (r *RawMessageStorer) StoreToSource() error {
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

func NewInstallationPayloadReceiver(logger *zap.Logger, encryptor PayloadEncryptor, backend *api.GethStatusBackend, deviceType string) *InstallationPayloadReceiver {
	l := logger.Named("InstallationPayloadManager")
	return &InstallationPayloadReceiver{
		logger:    l,
		encryptor: &encryptor,
		storer:    NewInstallationPayloadStorer(backend, deviceType),
	}
}

func (i *InstallationPayloadReceiver) Receive(data []byte) error {
	err := i.encryptor.decrypt(data)
	if err != nil {
		return err
	}
	i.storer.payload = i.encryptor.getDecrypted()
	return i.storer.StoreToSource()
}

func (i *InstallationPayloadReceiver) Received() []byte {
	if i.encryptor.payload.locked {
		return nil
	}
	return i.encryptor.getDecrypted()
}

func (i *InstallationPayloadReceiver) LockPayload() {
	i.encryptor.lockPayload()
}

func (i *InstallationPayloadReceiver) ResetPayload() {
	i.storer.payload = make([]byte, 0)
	i.encryptor.resetPayload()
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

func (r *InstallationPayloadStorer) StoreToSource() error {
	messenger := r.backend.Messenger()
	if messenger == nil {
		return fmt.Errorf("messenger is nil when invoke InstallationPayloadRepository#StoreToSource")
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
