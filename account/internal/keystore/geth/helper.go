package geth

import (
	"crypto/ecdsa"
	"os"

	"github.com/google/uuid"
	"github.com/status-im/extkeys"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account/types"
	"github.com/status-im/status-go/eth-node/crypto"
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

func newKeyForPurposeFromExtendedKey(keyPurpose extkeys.KeyPurpose, extKey *extkeys.ExtendedKey) (*ethtypes.Key, error) {
	var (
		extChild1 *extkeys.ExtendedKey // CKD#1 - main account (extChild2 removed in comparison to original code we have in go-ethereum fork)
		err       error
		id        uuid.UUID
	)

	if extKey.Depth == 0 { // we are dealing with master key
		// CKD#1 - main account
		extChild1, err = extKey.ChildForPurpose(keyPurpose, 0)
		if err != nil {
			return &ethtypes.Key{}, err
		}
	} else { // we are dealing with non-master key, so it is safe to persist and extend from it
		extChild1 = extKey
	}

	privateKeyECDSA := extChild1.ToECDSA()
	id, err = uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	key := &ethtypes.Key{
		ID:          id,
		Address:     crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey:  privateKeyECDSA,
		ExtendedKey: extChild1,
	}
	return key, nil
}

func zeroKey(k *ecdsa.PrivateKey) {
	if k == nil {
		return
	}

	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

func readKeystoreFileAndDecryptedKey(path string, auth string) (*ethtypes.Key, error) {
	keyjson, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return DecryptKey(keyjson, auth)
}

func encryptKeyAndStoreToKeystoreFile(ethKey *ethtypes.Key, path string, passphrase string) error {
	key := &keystore.Key{
		Id:              ethKey.ID,
		Address:         common.Address(ethKey.Address),
		PrivateKey:      ethKey.PrivateKey,
		ExtendedKey:     ethKey.ExtendedKey,
		SubAccountIndex: ethKey.SubAccountIndex,
	}

	keyjson, err := keystore.EncryptKey(key, passphrase, keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		return err
	}

	return os.WriteFile(path, keyjson, 0600)
}

func (a *Adapter) updateKeystoreFile(privateKey *ecdsa.PrivateKey, extKey *extkeys.ExtendedKey, passphrase string) (types.Account, error) {
	gethAccount, err := a.keystore.ImportECDSA(privateKey, passphrase)
	if err != nil {
		return types.Account{}, err
	}

	ethKey, err := readKeystoreFileAndDecryptedKey(gethAccount.URL.Path, passphrase)
	if err != nil {
		return types.Account{}, err
	}

	ethKey.ExtendedKey = extKey

	err = encryptKeyAndStoreToKeystoreFile(ethKey, gethAccount.URL.Path, passphrase)
	if err != nil {
		return types.Account{}, err
	}

	return accountFrom(gethAccount), nil
}
