package types

import (
	"errors"

	"github.com/status-im/status-go/crypto/types"
)

var (
	ErrInvalidAddress = errors.New("cannot parse address to valid account address")
)

type KeystoreAccount struct {
	Address types.Address `json:"address"` // Ethereum account address derived from the key
	URL     string        `json:"url"`     // Optional resource locator within a backend
}

// AddressToAccount parses a hex encoded string and returns it as an account
func AddressToAccount(address string) (KeystoreAccount, error) {
	if types.IsHexAddress(address) {
		return KeystoreAccount{Address: types.HexToAddress(address)}, nil
	}
	return KeystoreAccount{}, ErrInvalidAddress
}
