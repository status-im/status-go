package pairing

import (
	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/protobuf"
)

type PayloadMounter interface {
	PayloadLocker

	// Mount Loads the payload into the PayloadManager's state
	Mount() error

	// ToSend returns an outbound safe (encrypted) payload
	ToSend() []byte
}

type PayloadLoader interface {
	Load() error
}

type BasePayloadMounter struct {
	*PayloadLockPayload
	*PayloadToSend

	payloadLoader     PayloadLoader
	payloadMarshaller ProtobufMarshaller
	encryptor         *PayloadEncryptor
}

func NewBasePayloadMounter(loader PayloadLoader, marshaller ProtobufMarshaller, e *PayloadEncryptor) *BasePayloadMounter {
	return &BasePayloadMounter{
		PayloadLockPayload: &PayloadLockPayload{e},
		PayloadToSend:      &PayloadToSend{e},
		payloadLoader:      loader,
		payloadMarshaller:  marshaller,
		encryptor:          e,
	}
}

// Mount loads and prepares the payload to be stored in the PayloadLoader's state ready for later access
func (bpm *BasePayloadMounter) Mount() error {
	err := bpm.payloadLoader.Load()
	if err != nil {
		return err
	}

	p, err := bpm.payloadMarshaller.MarshalProtobuf()
	if err != nil {
		return err
	}

	return bpm.encryptor.encrypt(p)
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
	*BasePayloadMounter
}

// NewAccountPayloadMounter generates a new and initialised AccountPayloadMounter
func NewAccountPayloadMounter(pe *PayloadEncryptor, config *SenderConfig, logger *zap.Logger) (*AccountPayloadMounter, error) {
	l := logger.Named("AccountPayloadLoader")
	l.Debug("fired", zap.Any("config", config))

	pe = pe.Renew()

	// A new SHARED AccountPayload
	p := new(AccountPayload)
	apl, err := NewAccountPayloadLoader(p, config)
	if err != nil {
		return nil, err
	}

	return &AccountPayloadMounter{
		BasePayloadMounter: NewBasePayloadMounter(
			apl,
			NewPairingPayloadMarshaller(p, l),
			pe,
		),
	}, nil
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

func (apl *AccountPayloadLoader) Load() error {
	apl.keys = make(map[string][]byte)
	err := loadKeys(apl.keys, apl.keystorePath)
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

/*
|--------------------------------------------------------------------------
| RawMessagePayload
|--------------------------------------------------------------------------
|
| RawMessagePayloadMounter and RawMessageLoader
|
*/

type RawMessagePayloadMounter struct {
	*BasePayloadMounter
}

func NewRawMessagePayloadMounter(logger *zap.Logger, pe *PayloadEncryptor, backend *api.GethStatusBackend, config *SenderConfig) *RawMessagePayloadMounter {
	pe = pe.Renew()
	payload := new(protobuf.SyncRawMessage)

	return &RawMessagePayloadMounter{
		BasePayloadMounter: NewBasePayloadMounter(
			NewRawMessageLoader(backend, payload, config),
			NewRawMessagePayloadMarshaller(payload),
			pe,
		),
	}
}

type RawMessageLoader struct {
	payload               *protobuf.SyncRawMessage
	syncRawMessageHandler *SyncRawMessageHandler
	keyUID                string
	deviceType            string
}

func NewRawMessageLoader(backend *api.GethStatusBackend, payload *protobuf.SyncRawMessage, config *SenderConfig) *RawMessageLoader {
	return &RawMessageLoader{
		syncRawMessageHandler: NewSyncRawMessageHandler(backend),
		payload:               payload,
		keyUID:                config.KeyUID,
		deviceType:            config.DeviceType,
	}
}

func (r *RawMessageLoader) Load() (err error) {
	*r.payload, err = r.syncRawMessageHandler.PrepareRawMessage(r.keyUID, r.deviceType)
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
	*BasePayloadMounter
}

func NewInstallationPayloadMounter(logger *zap.Logger, pe *PayloadEncryptor, backend *api.GethStatusBackend, deviceType string) *InstallationPayloadMounter {
	pe = pe.Renew()
	payload := new(protobuf.SyncRawMessage)

	return &InstallationPayloadMounter{
		BasePayloadMounter: NewBasePayloadMounter(
			NewInstallationPayloadLoader(backend, payload, deviceType),
			NewRawMessagePayloadMarshaller(payload),
			pe,
		),
	}
}

type InstallationPayloadLoader struct {
	payload               *protobuf.SyncRawMessage
	syncRawMessageHandler *SyncRawMessageHandler
	deviceType            string
}

func NewInstallationPayloadLoader(backend *api.GethStatusBackend, payload *protobuf.SyncRawMessage, deviceType string) *InstallationPayloadLoader {
	return &InstallationPayloadLoader{
		payload:               payload,
		syncRawMessageHandler: NewSyncRawMessageHandler(backend),
		deviceType:            deviceType,
	}
}

func (r *InstallationPayloadLoader) Load() error {
	rawMessageCollector := new(RawMessageCollector)
	err := r.syncRawMessageHandler.CollectInstallationData(rawMessageCollector, r.deviceType)
	if err != nil {
		return err
	}
	*r.payload = rawMessageCollector.convertToSyncRawMessage()
	return nil
}

/*
|--------------------------------------------------------------------------
| PayloadMounters
|--------------------------------------------------------------------------
|
| Funcs for all PayloadMounters AccountPayloadMounter, RawMessagePayloadMounter and InstallationPayloadMounter
|
*/

func NewPayloadMounters(logger *zap.Logger, pe *PayloadEncryptor, backend *api.GethStatusBackend, config *SenderConfig) (*AccountPayloadMounter, *RawMessagePayloadMounter, *InstallationPayloadMounterReceiver, error) {
	am, err := NewAccountPayloadMounter(pe, config, logger)
	if err != nil {
		return nil, nil, nil, err
	}
	rmm := NewRawMessagePayloadMounter(logger, pe, backend, config)
	imr := NewInstallationPayloadMounterReceiver(logger, pe, backend, config.DeviceType)
	return am, rmm, imr, nil
}
