package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/keystore"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

// PayloadManager is the interface for PayloadManagers and wraps the basic functions for fulfilling payload management
type PayloadManager interface {
	// Mount Loads the payload into the PayloadManager's state
	Mount() error

	// Receive stores data from an inbound source into the PayloadManager's state
	Receive(data []byte) error

	// ToSend returns an outbound safe (encrypted) payload
	ToSend() []byte

	// Received returns a decrypted and parsed payload from an inbound source
	Received() []byte

	// ResetPayload resets all payloads the PayloadManager has in its state
	ResetPayload()

	// EncryptPlain encrypts the given plaintext using internal key(s)
	EncryptPlain(plaintext []byte) ([]byte, error)

	// LockPayload prevents future excess to outbound safe and received data
	LockPayload()
}

// PairingPayloadSourceConfig represents location and access data of the pairing payload
// ONLY available from the application client
type PairingPayloadSourceConfig struct {
	KeystorePath string `json:"keystorePath"`
	KeyUID       string `json:"keyUID"`
	Password     string `json:"password"`
}

// PairingPayloadManagerConfig represents the initialisation parameters required for a PairingPayloadManager
type PairingPayloadManagerConfig struct {
	DB *multiaccounts.Database
	PairingPayloadSourceConfig
}

// PairingPayloadManager is responsible for the whole lifecycle of a PairingPayload
type PairingPayloadManager struct {
	logger *zap.Logger
	pp     *PairingPayload
	*PayloadEncryptionManager
	ppm *PairingPayloadMarshaller
	ppr PayloadRepository
}

// NewPairingPayloadManager generates a new and initialised PairingPayloadManager
func NewPairingPayloadManager(aesKey []byte, config *PairingPayloadManagerConfig, logger *zap.Logger) (*PairingPayloadManager, error) {
	l := logger.Named("PairingPayloadManager")
	l.Debug("fired", zap.Binary("aesKey", aesKey), zap.Any("config", config))

	pem, err := NewPayloadEncryptionManager(aesKey, l)
	if err != nil {
		return nil, err
	}

	// A new SHARED PairingPayload
	p := new(PairingPayload)

	return &PairingPayloadManager{
		logger:                   l,
		pp:                       p,
		PayloadEncryptionManager: pem,
		ppm:                      NewPairingPayloadMarshaller(p, l),
		ppr:                      NewPairingPayloadRepository(p, config),
	}, nil
}

// Mount loads and prepares the payload to be stored in the PairingPayloadManager's state ready for later access
func (ppm *PairingPayloadManager) Mount() error {
	l := ppm.logger.Named("Mount()")
	l.Debug("fired")

	err := ppm.ppr.LoadFromSource()
	if err != nil {
		return err
	}
	l.Debug("after LoadFromSource")

	pb, err := ppm.ppm.MarshalToProtobuf()
	if err != nil {
		return err
	}
	l.Debug(
		"after MarshalToProtobuf",
		zap.Any("ppm.ppm.keys", ppm.ppm.keys),
		zap.Any("ppm.ppm.multiaccount", ppm.ppm.multiaccount),
		zap.String("ppm.ppm.password", ppm.ppm.password),
		zap.Binary("pb", pb),
	)

	return ppm.Encrypt(pb)
}

// Receive takes a []byte representing raw data, parses and stores the data
func (ppm *PairingPayloadManager) Receive(data []byte) error {
	l := ppm.logger.Named("Receive()")
	l.Debug("fired")

	err := ppm.Decrypt(data)
	if err != nil {
		return err
	}
	l.Debug("after Decrypt")

	err = ppm.ppm.UnmarshalProtobuf(ppm.Received())
	if err != nil {
		return err
	}
	l.Debug(
		"after UnmarshalProtobuf",
		zap.Any("ppm.ppm.keys", ppm.ppm.keys),
		zap.Any("ppm.ppm.multiaccount", ppm.ppm.multiaccount),
		zap.String("ppm.ppm.password", ppm.ppm.password),
		zap.Binary("ppm.Received()", ppm.Received()),
	)

	return ppm.ppr.StoreToSource()
}

// ResetPayload resets all payload state managed by the PairingPayloadManager
func (ppm *PairingPayloadManager) ResetPayload() {
	ppm.pp.ResetPayload()
	ppm.PayloadEncryptionManager.ResetPayload()
}

// EncryptionPayload represents the plain text and encrypted text of payload data
type EncryptionPayload struct {
	plain     []byte
	encrypted []byte
	locked    bool
}

func (ep *EncryptionPayload) lock() {
	ep.locked = true
}

// PayloadEncryptionManager is responsible for encrypting and decrypting payload data
type PayloadEncryptionManager struct {
	logger   *zap.Logger
	aesKey   []byte
	toSend   *EncryptionPayload
	received *EncryptionPayload
}

func NewPayloadEncryptionManager(aesKey []byte, logger *zap.Logger) (*PayloadEncryptionManager, error) {
	return &PayloadEncryptionManager{logger.Named("PayloadEncryptionManager"), aesKey, new(EncryptionPayload), new(EncryptionPayload)}, nil
}

// EncryptPlain encrypts any given plain text using the internal AES key and returns the encrypted value
// This function is different to Encrypt as the internal EncryptionPayload.encrypted value is not set
func (pem *PayloadEncryptionManager) EncryptPlain(plaintext []byte) ([]byte, error) {
	l := pem.logger.Named("EncryptPlain()")
	l.Debug("fired")

	return common.Encrypt(plaintext, pem.aesKey, rand.Reader)
}

func (pem *PayloadEncryptionManager) Encrypt(data []byte) error {
	l := pem.logger.Named("Encrypt()")
	l.Debug("fired")

	ep, err := common.Encrypt(data, pem.aesKey, rand.Reader)
	if err != nil {
		return err
	}

	pem.toSend.plain = data
	pem.toSend.encrypted = ep

	l.Debug(
		"after common.Encrypt",
		zap.Binary("data", data),
		zap.Binary("pem.aesKey", pem.aesKey),
		zap.Binary("ep", ep),
	)

	return nil
}

func (pem *PayloadEncryptionManager) Decrypt(data []byte) error {
	l := pem.logger.Named("Decrypt()")
	l.Debug("fired")

	pd, err := common.Decrypt(data, pem.aesKey)
	l.Debug(
		"after common.Decrypt(data, pem.aesKey)",
		zap.Binary("data", data),
		zap.Binary("pem.aesKey", pem.aesKey),
		zap.Binary("pd", pd),
		zap.Error(err),
	)
	if err != nil {
		return err
	}

	pem.received.encrypted = data
	pem.received.plain = pd
	return nil
}

func (pem *PayloadEncryptionManager) ToSend() []byte {
	if pem.toSend.locked {
		return nil
	}
	return pem.toSend.encrypted
}

func (pem *PayloadEncryptionManager) Received() []byte {
	if pem.toSend.locked {
		return nil
	}
	return pem.received.plain
}

func (pem *PayloadEncryptionManager) ResetPayload() {
	pem.toSend = new(EncryptionPayload)
	pem.received = new(EncryptionPayload)
}

func (pem *PayloadEncryptionManager) LockPayload() {
	l := pem.logger.Named("LockPayload")
	l.Debug("fired")

	pem.toSend.lock()
	pem.received.lock()
}

// PairingPayload represents the payload structure a PairingServer handles
type PairingPayload struct {
	keys         map[string][]byte
	multiaccount *multiaccounts.Account
	password     string
}

func (pp *PairingPayload) ResetPayload() {
	*pp = PairingPayload{}
}

// PairingPayloadMarshaller is responsible for marshalling and unmarshalling PairingServer payload data
type PairingPayloadMarshaller struct {
	logger *zap.Logger
	*PairingPayload
}

func NewPairingPayloadMarshaller(p *PairingPayload, logger *zap.Logger) *PairingPayloadMarshaller {
	return &PairingPayloadMarshaller{logger: logger, PairingPayload: p}
}

func (ppm *PairingPayloadMarshaller) MarshalToProtobuf() ([]byte, error) {
	return proto.Marshal(&protobuf.LocalPairingPayload{
		Keys:         ppm.accountKeysToProtobuf(),
		Multiaccount: ppm.multiaccount.ToProtobuf(),
		Password:     ppm.password,
	})
}

func (ppm *PairingPayloadMarshaller) accountKeysToProtobuf() []*protobuf.LocalPairingPayload_Key {
	var keys []*protobuf.LocalPairingPayload_Key
	for name, data := range ppm.keys {
		keys = append(keys, &protobuf.LocalPairingPayload_Key{Name: name, Data: data})
	}
	return keys
}

func (ppm *PairingPayloadMarshaller) UnmarshalProtobuf(data []byte) error {
	l := ppm.logger.Named("UnmarshalProtobuf()")
	l.Debug("fired")

	pb := new(protobuf.LocalPairingPayload)
	err := proto.Unmarshal(data, pb)
	l.Debug(
		"after protobuf.LocalPairingPayload",
		zap.Any("pb", pb),
		zap.Any("pb.Multiaccount", pb.Multiaccount),
		zap.Any("pb.Keys", pb.Keys),
	)
	if err != nil {
		return err
	}

	ppm.accountKeysFromProtobuf(pb.Keys)
	ppm.multiaccountFromProtobuf(pb.Multiaccount)
	ppm.password = pb.Password
	return nil
}

func (ppm *PairingPayloadMarshaller) accountKeysFromProtobuf(pbKeys []*protobuf.LocalPairingPayload_Key) {
	l := ppm.logger.Named("accountKeysFromProtobuf()")
	l.Debug("fired")

	if ppm.keys == nil {
		ppm.keys = make(map[string][]byte)
	}

	for _, key := range pbKeys {
		ppm.keys[key.Name] = key.Data
	}
	l.Debug(
		"after for _, key := range pbKeys",
		zap.Any("pbKeys", pbKeys),
		zap.Any("ppm.keys", ppm.keys),
	)
}

func (ppm *PairingPayloadMarshaller) multiaccountFromProtobuf(pbMultiAccount *protobuf.MultiAccount) {
	ppm.multiaccount = new(multiaccounts.Account)
	ppm.multiaccount.FromProtobuf(pbMultiAccount)
}

type PayloadRepository interface {
	LoadFromSource() error
	StoreToSource() error
}

// PairingPayloadRepository is responsible for loading, parsing, validating and storing PairingServer payload data
type PairingPayloadRepository struct {
	*PairingPayload

	multiaccountsDB *multiaccounts.Database

	keystorePath, keyUID string
}

func NewPairingPayloadRepository(p *PairingPayload, config *PairingPayloadManagerConfig) *PairingPayloadRepository {
	ppr := &PairingPayloadRepository{
		PairingPayload: p,
	}

	if config == nil {
		return ppr
	}

	ppr.multiaccountsDB = config.DB
	ppr.keystorePath = config.KeystorePath
	ppr.keyUID = config.KeyUID
	ppr.password = config.Password
	return ppr
}

func (ppr *PairingPayloadRepository) LoadFromSource() error {
	err := ppr.loadKeys(ppr.keystorePath)
	if err != nil {
		return err
	}

	err = ppr.validateKeys(ppr.password)
	if err != nil {
		return err
	}

	ppr.multiaccount, err = ppr.multiaccountsDB.GetAccount(ppr.keyUID)
	if err != nil {
		return err
	}

	return nil
}

func (ppr *PairingPayloadRepository) loadKeys(keyStorePath string) error {
	ppr.keys = make(map[string][]byte)

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

		ppr.keys[fileInfo.Name()] = rawKeyFile

		return nil
	}

	err := filepath.Walk(keyStorePath, fileWalker)
	if err != nil {
		return fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	return nil
}

func (ppr *PairingPayloadRepository) StoreToSource() error {
	err := ppr.validateKeys(ppr.password)
	if err != nil {
		return err
	}

	err = ppr.storeKeys(ppr.keystorePath)
	if err != nil {
		return err
	}

	err = ppr.storeMultiAccount()
	if err != nil {
		return err
	}

	// TODO install PublicKey into settings, probably do this outside of StoreToSource
	return nil
}

func (ppr *PairingPayloadRepository) validateKeys(password string) error {
	for _, key := range ppr.keys {
		k, err := keystore.DecryptKey(key, password)
		if err != nil {
			return err
		}

		err = generator.ValidateKeystoreExtendedKey(k)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ppr *PairingPayloadRepository) storeKeys(keyStorePath string) error {
	if keyStorePath == "" {
		return fmt.Errorf("keyStorePath can not be empty")
	}

	_, lastDir := filepath.Split(keyStorePath)

	// If lastDir == "keystore" we presume we need to create the rest of the keystore path
	// else we presume the provided keystore is valid
	if lastDir == "keystore" {
		if ppr.multiaccount == nil || ppr.multiaccount.KeyUID == "" {
			return fmt.Errorf("no known Key UID")
		}
		keyStorePath = filepath.Join(keyStorePath, ppr.multiaccount.KeyUID)

		err := os.MkdirAll(keyStorePath, 0777)
		if err != nil {
			return err
		}
	}

	for name, data := range ppr.keys {
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

func (ppr *PairingPayloadRepository) storeMultiAccount() error {
	return ppr.multiaccountsDB.SaveAccount(*ppr.multiaccount)
}
