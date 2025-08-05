package geth

import (
	"crypto/ecdsa"
	"errors"
	"os"

	"github.com/status-im/extkeys"

	"github.com/ethereum/go-ethereum/accounts"
	gethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/accounts-management/keystore"
	"github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type Adapter struct {
	keystoreDir string
	scryptN     int
	scryptP     int
	keystore    *gethkeystore.KeyStore
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

	ks := gethkeystore.NewKeyStore(keydir, scryptN, scryptP)

	return &Adapter{
		keystoreDir: keydir,
		scryptN:     scryptN,
		scryptP:     scryptP,
		keystore:    ks,
	}, nil
}

func mapToKeystoreError(err error) error {
	if errors.Is(err, gethkeystore.ErrNoMatch) {
		return keystore.ErrNoMatch
	}
	if errors.Is(err, gethkeystore.ErrDecrypt) {
		return keystore.ErrDecrypt
	}
	return err
}

// ImportECDSA imports an ECDSA private key
func (a *Adapter) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (account types.KeystoreAccount, err error) {
	defer func() {
		err = mapToKeystoreError(err)
	}()

	var gethAccount accounts.Account
	gethAccount, err = a.keystore.ImportECDSA(priv, passphrase)
	if err != nil {
		return
	}

	account = keystoreAccountFrom(gethAccount)
	return
}

// ImportSingleExtendedKey imports an extended key by converting it to ECDSA private key
func (a *Adapter) ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, passphrase string) (account types.KeystoreAccount, err error) {
	defer func() {
		err = mapToKeystoreError(err)
	}()

	privateKey := extKey.ToECDSA()
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	account, err = a.Find(address)
	if err == nil {
		return
	}

	account, err = a.updateKeystoreFile(privateKey, extKey, a.scryptN, a.scryptP, passphrase)
	return
}

// AccountDecryptedKey gets the decrypted key for an account using standard go-ethereum functions
func (a *Adapter) AccountDecryptedKey(address ethtypes.Address, passphrase string) (account types.KeystoreAccount, privateKey *ecdsa.PrivateKey, extendedKey *extkeys.ExtendedKey, err error) {
	defer func() {
		err = mapToKeystoreError(err)
	}()

	var gethAccount accounts.Account
	gethAccount, err = a.find(address)
	if err != nil {
		return
	}

	ethKey, err := readKeystoreFileAndDecryptedKey(gethAccount.URL.Path, passphrase)
	if err != nil {
		return
	}

	account = keystoreAccountFrom(gethAccount)
	privateKey = ethKey.PrivateKey
	extendedKey = ethKey.ExtendedKey
	return
}

func (a *Adapter) Delete(address ethtypes.Address) (err error) {
	defer func() {
		err = mapToKeystoreError(err)
	}()

	var gethAccount accounts.Account
	gethAccount, err = a.find(address)
	if err != nil {
		return
	}

	// TODO: think about how to use `Delete` method from `keystore` package, not from our fork
	// this is the only that depends on our fork for the account management part of the app.
	err = a.keystore.Delete(gethAccount)
	return
}

func (a *Adapter) Accounts() []types.KeystoreAccount {
	gethAccounts := a.keystore.Accounts()
	accounts := make([]types.KeystoreAccount, len(gethAccounts))
	for i, acc := range gethAccounts {
		accounts[i] = keystoreAccountFrom(acc)
	}
	return accounts
}

func (a *Adapter) Find(address ethtypes.Address) (account types.KeystoreAccount, err error) {
	defer func() {
		err = mapToKeystoreError(err)
	}()

	var gethAccount accounts.Account
	gethAccount, err = a.find(address)
	if err != nil {
		return
	}

	account = keystoreAccountFrom(gethAccount)
	return
}

func (a *Adapter) ReEncryptKeyStoreDir(oldPass, newPass string) (err error) {
	defer func() {
		err = mapToKeystoreError(err)
	}()

	err = reEncryptKeyStoreDir(a.keystoreDir, oldPass, newPass)
	return
}

func (a *Adapter) MigrateKeyStoreDir(newDir string) (err error) {
	defer func() {
		err = mapToKeystoreError(err)
	}()

	addresses := a.Accounts()
	addressesStr := make([]string, len(addresses))
	for i, address := range addresses {
		addressesStr[i] = address.Address.Hex()
	}

	err = migrateKeyStoreDir(a.keystoreDir, newDir, addressesStr)
	if err != nil {
		return err
	}
	a.keystoreDir = newDir
	return nil
}

func (a *Adapter) KeystorePath() string {
	return a.keystoreDir
}
