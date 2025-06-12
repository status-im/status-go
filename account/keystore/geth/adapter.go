package geth

import (
	"crypto/ecdsa"
	"errors"
	"os"

	"github.com/status-im/extkeys"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/account/types"
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

// ImportExtendedKeyForWallet imports an extended key by converting it to ECDSA private key and then deriving
// the appropriate child key (CKD#1) default wallet account for a wallet purpose
func (a *Adapter) ImportExtendedKeyForWallet(extKey *extkeys.ExtendedKey, passphrase string) (types.Account, error) {
	_, err := a.ImportSingleExtendedKey(extKey, passphrase) // create master account if it doesn't exist
	if err != nil {
		return types.Account{}, err
	}

	const keyPurpose = extkeys.KeyPurposeWallet
	key, err := newKeyForPurposeFromExtendedKey(keyPurpose, extKey)
	if err != nil {
		zeroKey(key.PrivateKey)
		return types.Account{}, err
	}

	account, err := a.Find(key.Address)
	if err == nil {
		return account, nil
	}

	return a.updateKeystoreFile(key.PrivateKey, key.ExtendedKey, a.scryptN, a.scryptP, passphrase)
}

// AccountDecryptedKey gets the decrypted key for an account using standard go-ethereum functions
func (a *Adapter) AccountDecryptedKey(address ethtypes.Address, passphrase string) (types.Account, *ethtypes.Key, error) {
	gethAccount, err := a.find(address)
	if err != nil {
		return types.Account{}, nil, err
	}

	ethKey, err := readKeystoreFileAndDecryptedKey(gethAccount.URL.Path, passphrase)
	if err != nil {
		return types.Account{}, nil, err
	}

	return accountFrom(gethAccount), ethKey, nil
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

func (a *Adapter) MigrateKeyStoreDir(newDir string, addresses []string) error {
	return migrateKeyStoreDir(a.keystoreDir, newDir, addresses)
}
