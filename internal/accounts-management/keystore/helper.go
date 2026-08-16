package keystore

import (
	"crypto/ecdsa"
	"os"

	"github.com/status-im/extkeys"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"

	geth "github.com/status-im/status-go/internal/accounts-management/keystore/internal/geth"
	"github.com/status-im/status-go/internal/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// This function is exposed from here just to be used for validating transferred keystore files while local pairing.
// The rest of the code should use the keystore package, via KeyStore interface.
func DecryptKey(keyjson []byte, auth string) (*types.Key, error) {
	return geth.DecryptKey(keyjson, auth)
}

func (a *Adapter) find(address cryptotypes.Address) (accounts.Account, error) {
	gethAccount, err := a.keystore.Find(accounts.Account{
		Address: common.Address(address),
	})
	if err != nil {
		return accounts.Account{}, err
	}
	return gethAccount, nil
}

func readKeystoreFileAndDecryptedKey(path string, auth string) (*types.Key, error) {
	keyjson, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return geth.DecryptKey(keyjson, auth)
}

func readKeystoreFileAndDecryptedPrivateKey(path string, auth string) (*types.Key, error) {
	keyjson, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return geth.DecryptPrivateKey(keyjson, auth)
}

func encryptKeyAndStoreToKeystoreFile(ethKey *types.Key, path string, scryptN int, scryptP int, passphrase string) error {
	key := &types.Key{
		ID:              ethKey.ID,
		Address:         ethKey.Address,
		PrivateKey:      ethKey.PrivateKey,
		ExtendedKey:     ethKey.ExtendedKey,
		SubAccountIndex: ethKey.SubAccountIndex,
	}

	keyjson, err := geth.EncryptKey(key, passphrase, scryptN, scryptP)
	if err != nil {
		return err
	}

	return os.WriteFile(path, keyjson, 0600)
}

func (a *Adapter) updateKeystoreFile(privateKey *ecdsa.PrivateKey, extKey *extkeys.ExtendedKey, scryptN int, scryptP int,
	passphrase string) (accounts.Account, error) {
	gethAccount, err := a.keystore.ImportECDSA(privateKey, passphrase)
	if err != nil {
		return accounts.Account{}, err
	}

	ethKey, err := readKeystoreFileAndDecryptedKey(gethAccount.URL.Path, passphrase)
	if err != nil {
		return accounts.Account{}, err
	}

	ethKey.ExtendedKey = extKey

	err = encryptKeyAndStoreToKeystoreFile(ethKey, gethAccount.URL.Path, scryptN, scryptP, passphrase)
	if err != nil {
		return accounts.Account{}, err
	}

	return gethAccount, nil
}
