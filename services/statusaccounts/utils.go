package statusaccounts

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
)

var ErrInvalidKeystoreExtendedKey = errors.New("PrivateKey and ExtendedKey are different")

func ValidateKeystoreExtendedKey(key *keystore.Key) error {
	if key.ExtendedKey.IsZeroed() {
		return nil
	}

	if !bytes.Equal(crypto.FromECDSA(key.PrivateKey), crypto.FromECDSA(key.ExtendedKey.ToECDSA())) {
		return ErrInvalidKeystoreExtendedKey
	}

	return nil
}
