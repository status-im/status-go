package geth

import (
	"crypto/ecdsa"
	"os"

	"github.com/status-im/extkeys"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/accounts-management/types"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

func (a *Adapter) find(address ethtypes.Address) (accounts.Account, error) {
	gethAccount, err := a.keystore.Find(accounts.Account{
		Address: common.Address(address),
	})
	if err != nil {
		return accounts.Account{}, err
	}
	return gethAccount, nil
}

func accountFrom(account accounts.Account) types.Account {
	return types.Account{
		Address: ethtypes.Address(account.Address),
		URL:     account.URL.String(),
	}
}

func readKeystoreFileAndDecryptedKey(path string, auth string) (*ethtypes.Key, error) {
	keyjson, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return DecryptKey(keyjson, auth)
}

func encryptKeyAndStoreToKeystoreFile(ethKey *ethtypes.Key, path string, scryptN int, scryptP int, passphrase string) error {
	key := &ethtypes.Key{
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
	passphrase string) (types.Account, error) {
	gethAccount, err := a.keystore.ImportECDSA(privateKey, passphrase)
	if err != nil {
		return types.Account{}, err
	}

	ethKey, err := readKeystoreFileAndDecryptedKey(gethAccount.URL.Path, passphrase)
	if err != nil {
		return types.Account{}, err
	}

	ethKey.ExtendedKey = extKey

	err = encryptKeyAndStoreToKeystoreFile(ethKey, gethAccount.URL.Path, scryptN, scryptP, passphrase)
	if err != nil {
		return types.Account{}, err
	}

	return accountFrom(gethAccount), nil
}
