package pairing

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	keystorepkg "github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/pkg/backend"
	"github.com/status-im/status-go/protocol/requests"
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
			data := AccountData{Account: p.multiaccount, Password: p.password, ChatKey: p.chatKey}
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

	if config.CreateAccount != nil {
		ppr.kdfIterations = config.CreateAccount.KdfIterations
		ppr.keystorePath = config.AbsoluteKeystorePath()
	}

	ppr.multiaccountsDB = config.DB
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

	_, profileDirStatErr := os.Stat(aps.profileKeystorePathFor(keyUID))
	profileDirExistedBefore := profileDirStatErr == nil

	if err = aps.storeKeys(aps.keystorePath); err != nil && err != ErrKeyFileAlreadyExists {
		if !profileDirExistedBefore {
			aps.cleanupProfileState(keyUID)
		}
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

	// A brand-new profile on this device adopts the DEK encryption scheme right away, so the first login creates the databases
	// under a device-local DEK. The transferred keystore files (password-encrypted wire format) are re-encrypted to the DEK.
	if err := aps.adoptDEKScheme(keyUID); err != nil {
		aps.cleanupProfileState(keyUID)
		return err
	}

	if err := aps.storeMultiAccount(); err != nil {
		aps.cleanupProfileState(keyUID)
		return err
	}
	return nil
}

// profileKeystorePathFor returns the profile keystore directory for the given keyUID.
func (aps *AccountPayloadStorer) profileKeystorePathFor(keyUID string) string {
	path := aps.keystorePath
	if filepath.Base(strings.TrimRight(path, "/\\")) == backend.DefaultKeystoreRelativePath {
		path = filepath.Join(path, keyUID)
	}
	return path
}

// cleanupProfileState removes the state created for a profile with passed keyUID
func (aps *AccountPayloadStorer) cleanupProfileState(keyUID string) {
	profileKeystorePath := aps.profileKeystorePathFor(keyUID)
	rootDataDir := rootDataDirFromKeystorePath(profileKeystorePath, keyUID)
	_ = envelope.Remove(rootDataDir, keyUID)
	_ = os.RemoveAll(profileKeystorePath)
	_ = os.RemoveAll(profileKeystorePath + "-backup")
	_ = os.RemoveAll(profileKeystorePath + "-re-encrypted")
}

func (aps *AccountPayloadStorer) adoptDEKScheme(keyUID string) error {
	profileKeystorePath := aps.profileKeystorePathFor(keyUID)
	rootDataDir := rootDataDirFromKeystorePath(profileKeystorePath, keyUID)

	dek, err := envelope.Generate()
	if err != nil {
		return err
	}
	if err := envelope.Write(rootDataDir, keyUID, dek, aps.password, sqlite.ReducedKDFIterationsNumber); err != nil {
		return err
	}

	if err := keystorepkg.ReEncryptKeyStoreDirAtPath(profileKeystorePath, aps.password, dek); err != nil {
		if verifyErr := keystorepkg.VerifyKeyStoreDirAtPath(profileKeystorePath, dek); verifyErr == nil {
			// fully on the DEK - the adoption effectively succeeded
			aps.kdfIterations = sqlite.ReducedKDFIterationsNumber
			return nil
		}
		if verifyErr := keystorepkg.VerifyKeyStoreDirAtPath(profileKeystorePath, aps.password); verifyErr == nil {
			// untouched - drop the envelope and leave the profile legacy
			_ = envelope.Remove(rootDataDir, keyUID)
			return nil
		}
		return err
	}

	aps.kdfIterations = sqlite.ReducedKDFIterationsNumber
	return nil
}

func (aps *AccountPayloadStorer) storeKeys(keyStorePath string) error {
	if keyStorePath == "" {
		return fmt.Errorf("keyStorePath can not be empty")
	}

	_, lastDir := filepath.Split(keyStorePath)

	// If lastDir == keystoreDir we presume we need to create the rest of the keystore path
	// else we presume the provided keystore is valid
	if lastDir == backend.DefaultKeystoreRelativePath {
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
func NewRawMessagePayloadReceiver(accountPayload *AccountPayload, e *PayloadEncryptor, backend *backend.StatusBackend, config *ReceiverConfig) *BasePayloadReceiver {
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
	createAccount         *requests.CreateAccount
	deviceType            string
}

func NewRawMessageStorer(backend *backend.StatusBackend, payload *RawMessagesPayload, accountPayload *AccountPayload, config *ReceiverConfig) *RawMessageStorer {
	return &RawMessageStorer{
		syncRawMessageHandler: NewSyncRawMessageHandler(backend),
		payload:               payload,
		accountPayload:        accountPayload,
		deviceType:            config.DeviceType,
		createAccount:         config.CreateAccount,
	}
}

func (r *RawMessageStorer) Store() error {
	if r.accountPayload == nil || r.accountPayload.multiaccount == nil {
		return fmt.Errorf("no known multiaccount when storing raw messages")
	}
	return r.syncRawMessageHandler.HandleRawMessage(r.accountPayload, r.createAccount, r.deviceType, r.payload)
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
func NewInstallationPayloadReceiver(e *PayloadEncryptor, backend *backend.StatusBackend, deviceType string) *BasePayloadReceiver {
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
	backend               *backend.StatusBackend
}

func NewInstallationPayloadStorer(backend *backend.StatusBackend, payload *RawMessagesPayload, deviceType string) *InstallationPayloadStorer {
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

func NewPayloadReceivers(logger *zap.Logger, pe *PayloadEncryptor, backend *backend.StatusBackend, config *ReceiverConfig) (PayloadReceiver, PayloadReceiver, PayloadMounterReceiver, error) {
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

/*
|--------------------------------------------------------------------------
| KeystoreFilesPayload
|--------------------------------------------------------------------------
*/

func NewKeystoreFilesPayloadReceiver(backend *backend.StatusBackend, e *PayloadEncryptor, config *KeystoreFilesReceiverConfig, logger *zap.Logger) (*BasePayloadReceiver, error) {
	l := logger.Named("KeystoreFilesPayloadManager")
	l.Debug("fired", zap.Any("config", config))

	e = e.Renew()

	// A new SHARED AccountPayload
	p := new(AccountPayload)

	kfps, err := NewKeystoreFilesPayloadStorer(backend, p, config)
	if err != nil {
		return nil, err
	}

	return NewBasePayloadReceiver(e, NewPairingPayloadMarshaller(p, l), kfps,
		func() {
			data := config.KeypairsToImport
			signal.SendLocalPairingEvent(Event{Type: EventReceivedKeystoreFiles, Action: ActionKeystoreFilesTransfer, Data: data})
		},
	), nil
}

type KeystoreFilesPayloadStorer struct {
	*AccountPayload

	keystorePath                   string
	loggedInKeyUID                 string
	expectedKeypairsToImport       []string
	expectedKeystoreFilesToReceive []string
	backend                        *backend.StatusBackend
}

func NewKeystoreFilesPayloadStorer(backend *backend.StatusBackend, p *AccountPayload, config *KeystoreFilesReceiverConfig) (*KeystoreFilesPayloadStorer, error) {
	if config == nil {
		return nil, fmt.Errorf("empty keystore files receiver config")
	}

	kfps := &KeystoreFilesPayloadStorer{
		AccountPayload:           p,
		keystorePath:             config.KeystorePath,
		loggedInKeyUID:           config.LoggedInKeyUID,
		expectedKeypairsToImport: config.KeypairsToImport,
		backend:                  backend,
	}

	accountService := backend.StatusNode().AccountService()

	for _, keyUID := range kfps.expectedKeypairsToImport {
		kp, err := accountService.GetKeypairByKeyUID(keyUID)
		if err != nil {
			return nil, err
		}

		if kp.Type == accsmanagementtypes.KeypairTypeSeed {
			kfps.expectedKeystoreFilesToReceive = append(kfps.expectedKeystoreFilesToReceive, kp.DerivedFrom[2:])
		}

		for _, acc := range kp.Accounts {
			kfps.expectedKeystoreFilesToReceive = append(kfps.expectedKeystoreFilesToReceive, acc.Address.Hex()[2:])
		}
	}

	return kfps, nil
}

func (kfps *KeystoreFilesPayloadStorer) Store() error {
	err := validateReceivedKeystoreFiles(kfps.expectedKeystoreFilesToReceive, kfps.keys, kfps.password)
	if err != nil {
		return err
	}

	return kfps.storeKeys(kfps.keystorePath)
}

func (kfps *KeystoreFilesPayloadStorer) storeKeys(keyStorePath string) error {
	messenger := kfps.backend.Messenger()
	if messenger == nil {
		return fmt.Errorf("messenger is nil")
	}

	if keyStorePath == "" {
		return fmt.Errorf("keyStorePath can not be empty")
	}

	_, lastDir := filepath.Split(keyStorePath)

	// If lastDir == keystoreDir we presume we need to create the rest of the keystore path
	// else we presume the provided keystore is valid
	if lastDir == backend.DefaultKeystoreRelativePath {
		keyStorePath = filepath.Join(keyStorePath, kfps.loggedInKeyUID)
		_, err := os.Stat(keyStorePath)
		if os.IsNotExist(err) {
			err := os.MkdirAll(keyStorePath, 0700)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	// When the logged-in profile uses the DEK encryption scheme, the received files (password-encrypted wire format)
	// must be stored under the profile's keystore secret
	storeSecret, err := kfps.backend.ResolveKeystoreSecret(kfps.loggedInKeyUID, kfps.password)
	if err != nil {
		return err
	}

	for name, data := range kfps.keys {
		address := ""
		for _, key := range kfps.expectedKeystoreFilesToReceive {
			if strings.Contains(name, strings.ToLower(key)) {
				address = strings.ToLower(key)
				break
			}
		}
		if address == "" {
			continue
		}

		// skip storing keystore file if it already exists
		exists, err := keystoreFileForAddressExists(keyStorePath, address)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		if storeSecret != kfps.password {
			data, err = keystorepkg.ReEncryptRawKey(data, kfps.password, storeSecret)
			if err != nil {
				return fmt.Errorf("failed to re-encrypt received keystore file %s: %w", name, err)
			}
		}

		err = ioutil.WriteFile(filepath.Join(keyStorePath, name), data, 0600)
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

	for _, keyUID := range kfps.expectedKeypairsToImport {
		err := messenger.MarkKeypairFullyOperable(keyUID)
		if err != nil {
			return err
		}
	}

	return nil
}

func keystoreFileForAddressExists(keyStorePath, address string) (bool, error) {
	entries, err := os.ReadDir(keyStorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.Contains(strings.ToLower(entry.Name()), address) {
			return true, nil
		}
	}
	return false, nil
}
