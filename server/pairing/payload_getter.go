package pairing

import (
	"encoding/json"
	"fmt"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/keystore"
	"github.com/status-im/status-go/multiaccounts"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"path/filepath"
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
	l := logger.Named("AccountPayloadGetter")
	l.Debug("fired", zap.Any("config", config))

	// A new SHARED AccountPayload
	p := new(AccountPayload)
	apg, err := NewAccountPayloadGetter(p, config)
	if err != nil {
		return nil, err
	}

	return &AccountPayloadMounter{
		logger:                   l,
		accountPayload:           p,
		encryptor:                pe,
		accountPayloadMarshaller: NewPairingPayloadMarshaller(p, l),
		payloadLoader:            apg,
	}, nil
}

// Mount loads and prepares the payload to be stored in the AccountPayloadGetter's state ready for later access
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

// AccountPayloadGetter is responsible for loading, parsing and validating AccountPayload data
type AccountPayloadGetter struct {
	*AccountPayload

	multiaccountsDB *multiaccounts.Database
	keystorePath    string
	keyUID          string
}

func NewAccountPayloadGetter(p *AccountPayload, config *SenderConfig) (*AccountPayloadGetter, error) {
	ppr := &AccountPayloadGetter{
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

func (apr *AccountPayloadGetter) LoadFromSource() error {
	err := apr.loadKeys(apr.keystorePath)
	if err != nil {
		return err
	}

	err = apr.validateKeys(apr.password)
	if err != nil {
		return err
	}

	apr.multiaccount, err = apr.multiaccountsDB.GetAccount(apr.keyUID)
	if err != nil {
		return err
	}

	return nil
}

func (apr *AccountPayloadGetter) loadKeys(keyStorePath string) error {
	apr.keys = make(map[string][]byte)

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

		apr.keys[fileInfo.Name()] = rawKeyFile

		return nil
	}

	err := filepath.Walk(keyStorePath, fileWalker)
	if err != nil {
		return fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	return nil
}

func (apr *AccountPayloadGetter) validateKeys(password string) error {
	for _, key := range apr.keys {
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

/*
func (apr *AccountPayloadRepository) StoreToSource() error {
	keyUID := apr.multiaccount.KeyUID
	if apr.loggedInKeyUID != "" && apr.loggedInKeyUID != keyUID {
		return ErrLoggedInKeyUIDConflict
	}
	if apr.loggedInKeyUID == keyUID {
		// skip storing keys if user is logged in with the same key
		return nil
	}

	err := apr.validateKeys(apr.password)
	if err != nil {
		return err
	}

	if err = apr.storeKeys(apr.keystorePath); err != nil && err != ErrKeyFileAlreadyExists {
		return err
	}

	// skip storing multiaccount if key already exists
	if err == ErrKeyFileAlreadyExists {
		apr.exist = true
		apr.multiaccount, err = apr.multiaccountsDB.GetAccount(keyUID)
		if err != nil {
			return err
		}
		return nil
	}
	return apr.storeMultiAccount()
}

func (apr *AccountPayloadRepository) storeKeys(keyStorePath string) error {
	if keyStorePath == "" {
		return fmt.Errorf("keyStorePath can not be empty")
	}

	_, lastDir := filepath.Split(keyStorePath)

	// If lastDir == "keystore" we presume we need to create the rest of the keystore path
	// else we presume the provided keystore is valid
	if lastDir == "keystore" {
		if apr.multiaccount == nil || apr.multiaccount.KeyUID == "" {
			return fmt.Errorf("no known Key UID")
		}
		keyStorePath = filepath.Join(keyStorePath, apr.multiaccount.KeyUID)
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

	for name, data := range apr.keys {
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

func (apr *AccountPayloadRepository) storeMultiAccount() error {
	apr.multiaccount.KDFIterations = apr.kdfIterations
	return apr.multiaccountsDB.SaveAccount(*apr.multiaccount)
}
*/
