package geth

import (
	"crypto/ecdsa"
	"os"

	"github.com/status-im/extkeys"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	keystoretypes "github.com/status-im/status-go/accounts-management/keystore/types"
	"github.com/status-im/status-go/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/crypto/types"
)

func (a *Adapter) find(address cryptotypes.Address) (accounts.Account, error) {
	gethAccount, err := a.keystore.Find(accounts.Account{
		Address: common.Address(address),
	})
	if err != nil {
		return accounts.Account{}, err
	}
	return gethAccount, nil
}

func keystoreAccountFrom(account accounts.Account) keystoretypes.KeystoreAccount {
	return keystoretypes.KeystoreAccount{
		Address: cryptotypes.Address(account.Address),
		URL:     account.URL.String(),
	}
}

func readKeystoreFileAndDecryptedKey(path string, auth string) (*types.Key, error) {
	keyjson, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return DecryptKey(keyjson, auth)
}

func encryptKeyAndStoreToKeystoreFile(ethKey *types.Key, path string, scryptN int, scryptP int, passphrase string) error {
	key := &types.Key{
		ID:              ethKey.ID,
		Address:         ethKey.Address,
		PrivateKey:      ethKey.PrivateKey,
		ExtendedKey:     ethKey.ExtendedKey,
		SubAccountIndex: ethKey.SubAccountIndex,
	}

	keyjson, err := EncryptKey(key, passphrase, scryptN, scryptP)
	if err != nil {
		return err
	}

	return os.WriteFile(path, keyjson, 0600)
}

func (a *Adapter) updateKeystoreFile(privateKey *ecdsa.PrivateKey, extKey *extkeys.ExtendedKey, scryptN int, scryptP int,
	passphrase string) (keystoretypes.KeystoreAccount, error) {
	gethAccount, err := a.keystore.ImportECDSA(privateKey, passphrase)
	if err != nil {
		return keystoretypes.KeystoreAccount{}, err
	}

	ethKey, err := readKeystoreFileAndDecryptedKey(gethAccount.URL.Path, passphrase)
	if err != nil {
		return keystoretypes.KeystoreAccount{}, err
	}

	ethKey.ExtendedKey = extKey

	err = encryptKeyAndStoreToKeystoreFile(ethKey, gethAccount.URL.Path, scryptN, scryptP, passphrase)
	if err != nil {
		return keystoretypes.KeystoreAccount{}, err
	}

	return keystoreAccountFrom(gethAccount), nil
}
