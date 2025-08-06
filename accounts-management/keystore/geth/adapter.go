package geth

import (
	"crypto/ecdsa"
	"errors"
	"os"

	"github.com/status-im/extkeys"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type Adapter struct {
	keystoreDir string
	scryptN     int
	scryptP     int
	keystore    *keystore.KeyStore
}

func NewGethKeystoreAdapter(keystoreDir string, scryptN int, scryptP int) (*Adapter, error) {
	var (
		keydir = keystoreDir
		err    error
	)
	if keydir == "" {
		keydir, err = os.MkdirTemp("", "go-ethereum-keystore")
		if err != nil {
			return nil, err
		}
	}

	if err = os.MkdirAll(keydir, 0700); err != nil {
		return nil, err
	}

	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	return &Adapter{
		keystoreDir: keydir,
		scryptN:     scryptN,
		scryptP:     scryptP,
		keystore:    ks,
	}, nil
}

// ImportECDSA imports an ECDSA private key
func (a *Adapter) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (types.Account, error) {
	gethAccount, err := a.keystore.ImportECDSA(priv, passphrase)
	return accountFrom(gethAccount), err
}

// ImportSingleExtendedKey imports an extended key by converting it to ECDSA private key
func (a *Adapter) ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, passphrase string) (types.Account, error) {
	privateKey := extKey.ToECDSA()
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	account, err := a.Find(address)
	if err == nil {
		return account, nil
	}

	return a.updateKeystoreFile(privateKey, extKey, a.scryptN, a.scryptP, passphrase)
}

// AccountDecryptedKey gets the decrypted key for an account using standard go-ethereum functions
func (a *Adapter) AccountDecryptedKey(address ethtypes.Address, passphrase string) (types.Account, *ecdsa.PrivateKey, *extkeys.ExtendedKey, error) {
	gethAccount, err := a.find(address)
	if err != nil {
		return types.Account{}, nil, nil, err
	}

	ethKey, err := readKeystoreFileAndDecryptedKey(gethAccount.URL.Path, passphrase)
	if err != nil {
		return types.Account{}, nil, nil, err
	}

	return accountFrom(gethAccount), ethKey.PrivateKey, ethKey.ExtendedKey, nil
}

func (a *Adapter) Delete(address ethtypes.Address) error {
	gethAccount, err := a.find(address)
	if err != nil {
		return err
	}

	// TODO: think about how to use `Delete` method from `keystore` package, not from our fork
	// this is the only that depends on our fork for the account management part of the app.
	return a.keystore.Delete(gethAccount)
}

func (a *Adapter) Accounts() []types.Account {
	gethAccounts := a.keystore.Accounts()
	accounts := make([]types.Account, len(gethAccounts))
	for i, acc := range gethAccounts {
		accounts[i] = accountFrom(acc)
	}
	return accounts
}

func (a *Adapter) Find(address ethtypes.Address) (types.Account, error) {
	gethAccount, err := a.find(address)
	if err != nil {
		if errors.Is(err, keystore.ErrNoMatch) {
			return types.Account{}, ErrNoMatch
		}
		return types.Account{}, err
	}
	return accountFrom(gethAccount), nil
}

func (a *Adapter) ReEncryptKeyStoreDir(oldPass, newPass string) error {
	return reEncryptKeyStoreDir(a.keystoreDir, oldPass, newPass)
}

func (a *Adapter) MigrateKeyStoreDir(newDir string) error {
	addresses := a.Accounts()
	addressesStr := make([]string, len(addresses))
	for i, address := range addresses {
		addressesStr[i] = address.Address.Hex()
	}

	err := migrateKeyStoreDir(a.keystoreDir, newDir, addressesStr)
	if err != nil {
		return err
	}
	a.keystoreDir = newDir
	return nil
}

func (a *Adapter) KeystorePath() string {
	return a.keystoreDir
}
