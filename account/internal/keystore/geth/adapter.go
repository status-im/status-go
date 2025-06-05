package geth

import (
	"crypto/ecdsa"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/extkeys"

	"github.com/status-im/status-go/account/types"
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

func (a *Adapter) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (types.Account, error) {
	gethAccount, err := a.keystore.ImportECDSA(priv, passphrase)
	return accountFrom(gethAccount), err
}

func (a *Adapter) ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, passphrase string) (types.Account, error) {
	gethAccount, err := a.keystore.ImportSingleExtendedKey(extKey, passphrase)
	return accountFrom(gethAccount), err
}

func (a *Adapter) ImportExtendedKeyForPurpose(keyPurpose extkeys.KeyPurpose, extKey *extkeys.ExtendedKey, passphrase string) (types.Account, error) {
	gethAccount, err := a.keystore.ImportExtendedKeyForPurpose(keyPurpose, extKey, passphrase)
	return accountFrom(gethAccount), err
}

func (a *Adapter) AccountDecryptedKey(account types.Account, auth string) (types.Account, *ethtypes.Key, error) {
	gethAccount, err := gethAccountFrom(account)
	if err != nil {
		return types.Account{}, nil, err
	}

	var gethKey *keystore.Key
	gethAccount, gethKey, err = a.keystore.AccountDecryptedKey(gethAccount, auth)
	return accountFrom(gethAccount), keyFrom(gethKey), err
}

func (a *Adapter) Delete(account types.Account) error {
	gethAccount, err := gethAccountFrom(account)
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

// parseGethURL converts a user supplied URL into the accounts specific structure.
func parseGethURL(url string) (accounts.URL, error) {
	parts := strings.Split(url, "://")
	if len(parts) != 2 || parts[0] == "" {
		return accounts.URL{}, errors.New("protocol scheme missing")
	}
	return accounts.URL{
		Scheme: parts[0],
		Path:   parts[1],
	}, nil
}

func gethAccountFrom(account types.Account) (accounts.Account, error) {
	var (
		gethAccount accounts.Account
		err         error
	)
	gethAccount.Address = gethcommon.Address(account.Address)
	if account.URL != "" {
		gethAccount.URL, err = parseGethURL(account.URL)
	}
	return gethAccount, err
}

func accountFrom(gethAccount accounts.Account) types.Account {
	return types.Account{
		Address: ethtypes.Address(gethAccount.Address),
		URL:     gethAccount.URL.String(),
	}
}

func keyFrom(k *keystore.Key) *ethtypes.Key {
	if k == nil {
		return nil
	}

	return &ethtypes.Key{
		ID:              k.Id,
		Address:         ethtypes.Address(k.Address),
		PrivateKey:      k.PrivateKey,
		ExtendedKey:     k.ExtendedKey,
		SubAccountIndex: k.SubAccountIndex,
	}
}
