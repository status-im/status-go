// +build nimbus

package nimbusbridge

// https://golang.org/cmd/cgo/

/*
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
#include <libnimbus.h>
*/
import "C"
import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"unsafe"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pborman/uuid"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/extkeys"
)

var (
	ErrInvalidSeed   = errors.New("seed is invalid")
	ErrInvalidKeyLen = errors.New("Nimbus serialized extended key length is invalid")
)

type nimbusKeyStoreAdapter struct {
}

// WrapKeyStore creates a types.KeyStore wrapper over the singleton Nimbus node
func WrapKeyStore() types.KeyStore {
	return &nimbusKeyStoreAdapter{}
}

func (k *nimbusKeyStoreAdapter) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (types.Account, error) {
	fmt.Println("ImportECDSA")
	panic("ImportECDSA")

	var privateKeyC unsafe.Pointer
	if priv != nil {
		privateKeyC = C.CBytes(crypto.FromECDSA(priv))
		defer C.free(privateKeyC)
	}
	passphraseC := C.CString(passphrase)
	defer C.free(unsafe.Pointer(passphraseC))

	var nimbusAccount C.account
	if !C.nimbus_keystore_import_ecdsa((*C.uchar)(privateKeyC), passphraseC, &nimbusAccount) {
		return types.Account{}, errors.New("failed to import ECDSA private key")
	}
	return accountFrom(&nimbusAccount), nil
}

func (k *nimbusKeyStoreAdapter) ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, passphrase string) (types.Account, error) {
	fmt.Println("ImportSingleExtendedKey")
	panic("ImportSingleExtendedKey")

	extKeyJSONC := C.CString(extKey.String())
	defer C.free(unsafe.Pointer(extKeyJSONC))
	passphraseC := C.CString(passphrase)
	defer C.free(unsafe.Pointer(passphraseC))

	var nimbusAccount C.account
	if !C.nimbus_keystore_import_single_extendedkey(extKeyJSONC, passphraseC, &nimbusAccount) {
		return types.Account{}, errors.New("failed to import extended key")
	}
	return accountFrom(&nimbusAccount), nil
}

func (k *nimbusKeyStoreAdapter) ImportExtendedKeyForPurpose(keyPurpose extkeys.KeyPurpose, extKey *extkeys.ExtendedKey, passphrase string) (types.Account, error) {
	fmt.Println("ImportExtendedKeyForPurpose")

	passphraseC := C.CString(passphrase)
	defer C.free(unsafe.Pointer(passphraseC))
	extKeyJSONC := C.CString(extKey.String())
	defer C.free(unsafe.Pointer(extKeyJSONC))

	var nimbusAccount C.account
	if !C.nimbus_keystore_import_extendedkeyforpurpose(C.int(keyPurpose), extKeyJSONC, passphraseC, &nimbusAccount) {
		return types.Account{}, errors.New("failed to import extended key")
	}
	return accountFrom(&nimbusAccount), nil
}

func (k *nimbusKeyStoreAdapter) AccountDecryptedKey(a types.Account, auth string) (types.Account, *types.Key, error) {
	fmt.Println("AccountDecryptedKey")
	panic("AccountDecryptedKey")

	authC := C.CString(auth)
	defer C.free(unsafe.Pointer(authC))

	var nimbusAccount C.account
	err := nimbusAccountFrom(a, &nimbusAccount)
	if err != nil {
		return types.Account{}, nil, err
	}

	var nimbusKey C.key
	if !C.nimbus_keystore_account_decrypted_key(authC, &nimbusAccount, &nimbusKey) {
		return types.Account{}, nil, errors.New("failed to decrypt account key")
	}
	key, err := keyFrom(&nimbusKey)
	if err != nil {
		return types.Account{}, nil, err
	}
	return accountFrom(&nimbusAccount), key, nil
}

func (k *nimbusKeyStoreAdapter) Delete(a types.Account, auth string) error {
	fmt.Println("Delete")

	var nimbusAccount C.account
	err := nimbusAccountFrom(a, &nimbusAccount)
	if err != nil {
		return err
	}

	authC := C.CString(auth)
	defer C.free(unsafe.Pointer(authC))

	if !C.nimbus_keystore_delete(&nimbusAccount, authC) {
		return errors.New("failed to delete account")
	}

	return nil
}

func nimbusAccountFrom(account types.Account, nimbusAccount *C.account) error {
	fmt.Println("nimbusAccountFrom")
	err := copyAddressToCBuffer(&nimbusAccount.address[0], account.Address.Bytes())
	if err != nil {
		return err
	}
	if account.URL == "" {
		nimbusAccount.url[0] = C.char(0)
	} else if len(account.URL) >= C.URL_LEN {
		return errors.New("URL is too long to fit in Nimbus struct")
	} else {
		copyURLToCBuffer(&nimbusAccount.url[0], account.URL)
	}
	return err
}

func accountFrom(nimbusAccount *C.account) types.Account {
	return types.Account{
		Address: types.BytesToAddress(C.GoBytes(unsafe.Pointer(&nimbusAccount.address[0]), C.ADDRESS_LEN)),
		URL:     C.GoString(&nimbusAccount.url[0]),
	}
}

// copyAddressToCBuffer copies a Go buffer to a C buffer without allocating new memory
func copyAddressToCBuffer(dst *C.uchar, src []byte) error {
	if len(src) != C.ADDRESS_LEN {
		return errors.New("invalid buffer size")
	}

	p := (*[C.ADDRESS_LEN]C.uchar)(unsafe.Pointer(dst))
	for index, b := range src {
		p[index] = C.uchar(b)
	}

	return nil
}

// copyURLToCBuffer copies a Go buffer to a C buffer without allocating new memory
func copyURLToCBuffer(dst *C.char, src string) error {
	if len(src)+1 > C.URL_LEN {
		return errors.New("URL is too long to fit in Nimbus struct")
	}

	p := (*[C.URL_LEN]C.uchar)(unsafe.Pointer(dst))
	for index := 0; index <= len(src); index++ {
		p[index] = C.uchar(src[index])
	}

	return nil
}

func keyFrom(k *C.key) (*types.Key, error) {
	fmt.Println("keyFrom")
	if k == nil {
		return nil, nil
	}

	var err error
	key := types.Key{
		ID: uuid.Parse(C.GoString(&k.id[0])),
	}
	key.Address = types.BytesToAddress(C.GoBytes(unsafe.Pointer(&k.address[0]), C.ADDRESS_LEN))
	key.PrivateKey, err = crypto.ToECDSA(C.GoBytes(unsafe.Pointer(&k.privateKeyID[0]), C.PRIVKEY_LEN))
	if err != nil {
		return nil, err
	}
	key.ExtendedKey, err = newExtKeyFromBuffer(C.GoBytes(unsafe.Pointer(&k.extKey[0]), C.EXTKEY_LEN))
	if err != nil {
		return nil, err
	}
	return &key, err
}

// newExtKeyFromBuffer returns a new extended key instance from a serialized
// extended key.
func newExtKeyFromBuffer(key []byte) (*extkeys.ExtendedKey, error) {
	if len(key) == 0 {
		return &extkeys.ExtendedKey{}, nil
	}

	if len(key) != C.EXTKEY_LEN {
		return nil, ErrInvalidKeyLen
	}

	// The serialized format is:
	//   version (4) || depth (1) || parent fingerprint (4)) ||
	//   child num (4) || chain code (32) || key data (33)

	payload := key

	// Deserialize each of the payload fields.
	version := payload[:4]
	depth := payload[4:5][0]
	fingerPrint := payload[5:9]
	childNumber := binary.BigEndian.Uint32(payload[9:13])
	chainCode := payload[13:45]
	keyData := payload[45:78]

	// The key data is a private key if it starts with 0x00.  Serialized
	// compressed pubkeys either start with 0x02 or 0x03.
	isPrivate := keyData[0] == 0x00
	if isPrivate {
		// Ensure the private key is valid.  It must be within the range
		// of the order of the secp256k1 curve and not be 0.
		keyData = keyData[1:]
		keyNum := new(big.Int).SetBytes(keyData)
		if keyNum.Cmp(btcec.S256().N) >= 0 || keyNum.Sign() == 0 {
			return nil, ErrInvalidSeed
		}
	} else {
		// Ensure the public key parses correctly and is actually on the
		// secp256k1 curve.
		_, err := btcec.ParsePubKey(keyData, btcec.S256())
		if err != nil {
			return nil, err
		}
	}

	return &extkeys.ExtendedKey{
		Version:     version,
		KeyData:     keyData,
		ChainCode:   chainCode,
		FingerPrint: fingerPrint,
		Depth:       depth,
		ChildNumber: childNumber,
		IsPrivate:   isPrivate,
	}, nil
}
