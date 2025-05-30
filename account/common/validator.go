package common

import (
	"bytes"
	"errors"

	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

var (
	// ErrInvalidKeystoreExtendedKey is returned when the decrypted keystore file
	// contains some old Status keys.
	// The old version used to store the BIP44 account at index 0 as PrivateKey,
	// and the BIP44 account at index 1 as ExtendedKey.
	// The current version stores the same key as PrivateKey and ExtendedKey.
	ErrInvalidKeystoreExtendedKey = errors.New("PrivateKey and ExtendedKey are different")
)

// ValidateExtendedKey validates the keystore keys, checking that
// ExtendedKey is the extended key of PrivateKey
func ValidateExtendedKey(key *ethtypes.Key) error {
	if key == nil || key.ExtendedKey == nil || key.ExtendedKey.IsZeroed() {
		return nil
	}

	privKey := crypto.FromECDSA(key.PrivateKey)
	extKey := crypto.FromECDSA(key.ExtendedKey.ToECDSA())
	if !bytes.Equal(privKey, extKey) {
		return ErrInvalidKeystoreExtendedKey
	}

	return nil
}
