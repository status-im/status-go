package pairing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/keystore"
	"github.com/status-im/status-go/multiaccounts"
)

type PayloadMounter interface {
	PayloadLocker

	// Mount Loads the payload into the PayloadManager's state
	Mount() error

	// ToSend returns an outbound safe (encrypted) payload
	ToSend() []byte
}

type PayloadLoader interface {
	LoadFromSource() error
}

/*
|--------------------------------------------------------------------------
| AccountPayload
|--------------------------------------------------------------------------
|
| AccountPayloadMounter, AccountPayloadLoader and AccountPayloadMarshaller
|
*/

// AccountPayloadMounter is responsible for the whole lifecycle of an AccountPayload
type AccountPayloadMounter struct {
	logger                   *zap.Logger
	accountPayload           *AccountPayload
	encryptor                *PayloadEncryptor
	accountPayloadMarshaller *AccountPayloadMarshaller
	payloadLoader            PayloadLoader
}

// NewAccountPayloadMounter generates a new and initialised AccountPayloadMounter
func NewAccountPayloadMounter(pe *PayloadEncryptor, config *SenderConfig, logger *zap.Logger) (*AccountPayloadMounter, error) {
	l := logger.Named("AccountPayloadLoader")
	l.Debug("fired", zap.Any("config", config))

	// A new SHARED AccountPayload
	p := new(AccountPayload)
	apl, err := NewAccountPayloadLoader(p, config)
	if err != nil {
		return nil, err
	}

	return &AccountPayloadMounter{
		logger:                   l,
		accountPayload:           p,
		encryptor:                pe.Renew(),
		accountPayloadMarshaller: NewPairingPayloadMarshaller(p, l),
		payloadLoader:            apl,
	}, nil
}

// Mount loads and prepares the payload to be stored in the AccountPayloadLoader's state ready for later access
func (apm *AccountPayloadMounter) Mount() error {
	l := apm.logger.Named("Mount()")
	l.Debug("fired")

	err := apm.payloadLoader.LoadFromSource()
	if err != nil {
		return err
	}
	l.Debug("after LoadFromSource")

	pb, err := apm.accountPayloadMarshaller.MarshalToProtobuf()
	if err != nil {
		return err
	}
	l.Debug(
		"after MarshalToProtobuf",
		zap.Any("accountPayloadMarshaller.accountPayloadMarshaller.keys", apm.accountPayloadMarshaller.keys),
		zap.Any("accountPayloadMarshaller.accountPayloadMarshaller.multiaccount", apm.accountPayloadMarshaller.multiaccount),
		zap.String("accountPayloadMarshaller.accountPayloadMarshaller.password", apm.accountPayloadMarshaller.password),
		zap.Binary("pb", pb),
	)

	return apm.encryptor.encrypt(pb)
}

func (apm *AccountPayloadMounter) ToSend() []byte {
	return apm.encryptor.getEncrypted()
}

func (apm *AccountPayloadMounter) LockPayload() {
	apm.encryptor.lockPayload()
}

// AccountPayloadLoader is responsible for loading, parsing and validating AccountPayload data
type AccountPayloadLoader struct {
	*AccountPayload

	multiaccountsDB *multiaccounts.Database
	keystorePath    string
	keyUID          string
}

func NewAccountPayloadLoader(p *AccountPayload, config *SenderConfig) (*AccountPayloadLoader, error) {
	ppr := &AccountPayloadLoader{
		AccountPayload: p,
	}

	if config == nil {
		return ppr, nil
	}

	ppr.multiaccountsDB = config.DB
	ppr.keyUID = config.KeyUID
	ppr.password = config.Password
	ppr.keystorePath = config.KeystorePath
	return ppr, nil
}

func (apl *AccountPayloadLoader) LoadFromSource() error {
	err := apl.loadKeys(apl.keystorePath)
	if err != nil {
		return err
	}

	err = validateKeys(apl.keys, apl.password)
	if err != nil {
		return err
	}

	apl.multiaccount, err = apl.multiaccountsDB.GetAccount(apl.keyUID)
	if err != nil {
		return err
	}

	return nil
}

func (apl *AccountPayloadLoader) loadKeys(keyStorePath string) error {
	apl.keys = make(map[string][]byte)

	fileWalker := func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fileInfo.IsDir() || filepath.Dir(path) != keyStorePath {
			return nil
		}

		rawKeyFile, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("invalid account key file: %v", err)
		}

		accountKey := new(keystore.EncryptedKeyJSONV3)
		if err := json.Unmarshal(rawKeyFile, &accountKey); err != nil {
			return fmt.Errorf("failed to read key file: %s", err)
		}

		if len(accountKey.Address) != 40 {
			return fmt.Errorf("account key address has invalid length '%s'", accountKey.Address)
		}

		apl.keys[fileInfo.Name()] = rawKeyFile

		return nil
	}

	err := filepath.Walk(keyStorePath, fileWalker)
	if err != nil {
		return fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	return nil
}

/*
|--------------------------------------------------------------------------
| RawMessagePayload
|--------------------------------------------------------------------------
|
| RawMessagePayloadMounter and RawMessageLoader
|
*/

type RawMessagePayloadMounter struct {
	logger *zap.Logger

	encryptor *PayloadEncryptor
	loader    *RawMessageLoader
}

func NewRawMessagePayloadMounter(logger *zap.Logger, pe *PayloadEncryptor, backend *api.GethStatusBackend, config *SenderConfig) *RawMessagePayloadMounter {
	l := logger.Named("RawMessagePayloadManager")
	return &RawMessagePayloadMounter{
		logger:    l,
		encryptor: pe.Renew(),
		loader:    NewRawMessageLoader(backend, config),
	}
}

func (r *RawMessagePayloadMounter) Mount() error {
	err := r.loader.LoadFromSource()
	if err != nil {
		return err
	}
	return r.encryptor.encrypt(r.loader.payload)
}

func (r *RawMessagePayloadMounter) ToSend() []byte {
	return r.encryptor.getEncrypted()
}

func (r *RawMessagePayloadMounter) LockPayload() {
	r.encryptor.lockPayload()
}

func (r *RawMessagePayloadMounter) ResetPayload() {
	r.loader.payload = make([]byte, 0)
	r.encryptor.resetPayload()
}

type RawMessageLoader struct {
	payload               []byte
	syncRawMessageHandler *SyncRawMessageHandler
	keyUID                string
	deviceType            string
}

func NewRawMessageLoader(backend *api.GethStatusBackend, config *SenderConfig) *RawMessageLoader {
	return &RawMessageLoader{
		syncRawMessageHandler: NewSyncRawMessageHandler(backend),
		payload:               make([]byte, 0),
		keyUID:                config.KeyUID,
		deviceType:            config.DeviceType,
	}
}

func (r *RawMessageLoader) LoadFromSource() (err error) {
	r.payload, err = r.syncRawMessageHandler.PrepareRawMessage(r.keyUID, r.deviceType)
	return err
}

/*
|--------------------------------------------------------------------------
| InstallationPayload
|--------------------------------------------------------------------------
|
| InstallationPayloadMounter and InstallationPayloadLoader
|
*/

type InstallationPayloadMounter struct {
	logger    *zap.Logger
	encryptor *PayloadEncryptor
	loader    *InstallationPayloadLoader
}

func NewInstallationPayloadMounter(logger *zap.Logger, pe *PayloadEncryptor, backend *api.GethStatusBackend, deviceType string) *InstallationPayloadMounter {
	return &InstallationPayloadMounter{
		logger:    logger.Named("InstallationPayloadManager"),
		encryptor: pe.Renew(),
		loader:    NewInstallationPayloadLoader(backend, deviceType),
	}
}

func (i *InstallationPayloadMounter) Mount() error {
	err := i.loader.LoadFromSource()
	if err != nil {
		return err
	}
	return i.encryptor.encrypt(i.loader.payload)
}

func (i *InstallationPayloadMounter) ToSend() []byte {
	return i.encryptor.getEncrypted()
}

func (i *InstallationPayloadMounter) LockPayload() {
	i.encryptor.lockPayload()
}

func (i *InstallationPayloadMounter) ResetPayload() {
	i.loader.payload = make([]byte, 0)
	i.encryptor.resetPayload()
}

type InstallationPayloadLoader struct {
	payload               []byte
	syncRawMessageHandler *SyncRawMessageHandler
	deviceType            string
}

func NewInstallationPayloadLoader(backend *api.GethStatusBackend, deviceType string) *InstallationPayloadLoader {
	return &InstallationPayloadLoader{
		syncRawMessageHandler: NewSyncRawMessageHandler(backend),
		deviceType:            deviceType,
	}
}

func (r *InstallationPayloadLoader) LoadFromSource() error {
	rawMessageCollector := new(RawMessageCollector)
	err := r.syncRawMessageHandler.CollectInstallationData(rawMessageCollector, r.deviceType)
	if err != nil {
		return err
	}
	r.payload, err = proto.Marshal(rawMessageCollector.convertToSyncRawMessage())
	return err
}
