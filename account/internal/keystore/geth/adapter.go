package geth

import (
	"crypto/ecdsa"

	"github.com/status-im/extkeys"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/account/types"
	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type Adapter struct {
	keystore *keystore.KeyStore
}

func NewAdapter(ks *keystore.KeyStore) *Adapter {
	return &Adapter{
		keystore: ks,
	}
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

	return a.updateKeystoreFile(privateKey, extKey, passphrase)
}

// ImportExtendedKeyForWallet imports an extended key for a wallet purpose by deriving the appropriate child key
func (a *Adapter) ImportExtendedKeyForWallet(extKey *extkeys.ExtendedKey, passphrase string) (types.Account, error) {
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
	return a.updateKeystoreFile(key.PrivateKey, key.ExtendedKey, passphrase)
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
		return types.Account{}, err
	}
	return accountFrom(gethAccount), nil
}

func (a *Adapter) VerifyPassword(address ethtypes.Address, passphrase string) (*ethtypes.Key, error) {
	gethAccount, err := a.find(address)
	if err != nil {
		return nil, err
	}

	return readKeystoreFileAndDecryptedKey(gethAccount.URL.Path, passphrase)
}

func (a *Adapter) ReEncryptKeyStoreDir(keyDirPath, oldPass, newPass string) error {
	return ReEncryptKeyStoreDir(keyDirPath, oldPass, newPass)
}
