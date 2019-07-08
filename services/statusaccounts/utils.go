package statusaccounts

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
)

// ErrInvalidKeystoreExtendedKey is returned when the decrypted keystore file
// contains some old Status keys.
// The old version used to store the BIP44 account at index 0 as PrivateKey,
// and the BIP44 account at index 1 as ExtendedKey.
// The current version stores the same key as PrivateKey and ExtendedKey.
var ErrInvalidKeystoreExtendedKey = errors.New("PrivateKey and ExtendedKey are different")

// ValidateKeystoreExtendedKey validates the keystore keys, checking that
// ExtendedKey is the extended key of PrivateKey.
func ValidateKeystoreExtendedKey(key *keystore.Key) error {
	if key.ExtendedKey.IsZeroed() {
		return nil
	}

	if !bytes.Equal(crypto.FromECDSA(key.PrivateKey), crypto.FromECDSA(key.ExtendedKey.ToECDSA())) {
		return ErrInvalidKeystoreExtendedKey
	}

	return nil
}
