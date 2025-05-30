package types

import "errors"

// Account represents an Ethereum account located at a specific location defined
// by the optional URL field.
var (
	ErrInvalidAddress = errors.New("cannot parse address to valid account address")
)

type Account struct {
	Address Address `json:"address"` // Ethereum account address derived from the key
	URL     string  `json:"url"`     // Optional resource locator within a backend
}

// AddressToAccount parses a hex encoded string and returns it as an account
func AddressToAccount(address string) (Account, error) {
	if IsHexAddress(address) {
		return Account{Address: HexToAddress(address)}, nil
	}
	return Account{}, ErrInvalidAddress
}

// FromStringToAddress converts an account address from string to types.Address
// Useful for formatting the "From" field of send transaction struct
func FromStringToAddress(address string) Address {
	account, err := AddressToAccount(address)
	if err != nil {
		return Address{}
	}
	return account.Address
}
