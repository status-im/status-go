package statusaccounts

import (
	"errors"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/keystore"
)

var ErrInvalidKeystoreExtendedKey = errors.New("PrivateKey and ExtendedKey are different")

func ValidateKeystoreExtendedKey(key *keystore.Key) error {
	if key.ExtendedKey.IsZeroed() {
		return nil
	}

	if !reflect.DeepEqual(key.PrivateKey, key.ExtendedKey.ToECDSA()) {
		return ErrInvalidKeystoreExtendedKey
	}

	return nil
}
