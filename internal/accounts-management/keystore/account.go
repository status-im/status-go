package keystore

import (
	types2 "github.com/status-im/status-go/internal/crypto/types"
)

type KeystoreAccount struct {
	Address types2.Address `json:"address"` // Ethereum account address derived from the key
	URL     string         `json:"url"`     // Optional resource locator within a backend
}

// AddressToAccount parses a hex encoded string and returns it as an account
func AddressToAccount(address string) (KeystoreAccount, error) {
	if types2.IsHexAddress(address) {
		return KeystoreAccount{Address: types2.HexToAddress(address)}, nil
	}
	return KeystoreAccount{}, ErrInvalidAddress
}
