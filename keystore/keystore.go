package keystore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	"github.com/status-im/status-go/extkeys"
	"github.com/status-im/status-go/keychain"
)

func NewKeystore(directory string, keychain keychain.Keychain) *Keystore {
	return &Keystore{
		directory: directory,
		keychain:  keychain,
	}
}

type Keystore struct {
	directory string
	keychain  keychain.Keychain
}

func (k *Keystore) ImportKey(ekey *extkeys.ExtendedKey, auth string) (common.Address, error) {
	// TODO zero ecdsa key and marshalled bytes
	key, err := newKeyForPurposeFromExtendedKey(extkeys.KeyPurposeWallet, ekey)
	if err != nil {
		return common.Address{}, err
	}
	path := filepath.Join(k.directory, key.Address.String())
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return common.Address{}, fmt.Errorf("account with address %x already exists", key.Address)
	}
	bytes, err := json.Marshal(key)
	if err != nil {
		return common.Address{}, err
	}
	err = k.keychain.CreateKey(auth)
	if err != nil {
		return common.Address{}, err
	}
	encrypted, err := k.keychain.Encrypt(auth, bytes)
	if err != nil {
		return common.Address{}, err
	}
	return key.Address, ioutil.WriteFile(path, encrypted, 0666)
}

func (k *Keystore) GetDecryptedKey(address common.Address, auth string) (*keystore.Key, error) {
	// TODO zero decrypted bytes
	path := filepath.Join(k.directory, address.String())
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	keydata, err := k.keychain.Decrypt(auth, bytes)
	if err != nil {
		return nil, err
	}
	var key *keystore.Key
	err = json.Unmarshal(keydata, key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (k *Keystore) Accounts() (rst []common.Address, err error) {
	files, err := ioutil.ReadDir(k.directory)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		parts := strings.Split(f.Name(), "/")
		if len(parts) < 2 {
			continue
		}
		rst = append(rst, common.HexToAddress(parts[len(parts[-1])]))
	}
	return rst, nil
}

func newKeyForPurposeFromExtendedKey(keyPurpose extkeys.KeyPurpose, extKey *extkeys.ExtendedKey) (*keystore.Key, error) {
	var (
		extChild1, extChild2 *extkeys.ExtendedKey
		err                  error
	)

	if extKey.Depth == 0 { // we are dealing with master key
		// CKD#1 - main account
		extChild1, err = extKey.ChildForPurpose(keyPurpose, 0)
		if err != nil {
			return &keystore.Key{}, err
		}

		// CKD#2 - sub-accounts root
		extChild2, err = extKey.ChildForPurpose(keyPurpose, 1)
		if err != nil {
			return &keystore.Key{}, err
		}
	} else { // we are dealing with non-master key, so it is safe to persist and extend from it
		extChild1 = extKey
		extChild2 = extKey
	}

	privateKeyECDSA := extChild1.ToECDSA()
	id := uuid.NewRandom()
	key := &keystore.Key{
		Id:          id,
		Address:     crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey:  privateKeyECDSA,
		ExtendedKey: extChild2,
	}
	return key, nil
}
