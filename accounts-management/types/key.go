package types

import (
	"crypto/ecdsa"

	"github.com/google/uuid"

	"github.com/status-im/extkeys"

	cryptotypes "github.com/status-im/status-go/crypto/types"
)

// Key represents a cryptographic key with both private key and extended key
type Key struct {
	ID uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address cryptotypes.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
	// ExtendedKey is the extended key of the PrivateKey itself, and it's used
	// to derive child keys.
	ExtendedKey *extkeys.ExtendedKey
	// Deprecated: SubAccountIndex
	// It was use in Status to keep track of the number of sub-account created
	// before having multi-account support.
	SubAccountIndex uint32
}

// NewKey creates a new Key instance
func NewKey(privateKey *ecdsa.PrivateKey, extendedKey *extkeys.ExtendedKey) *Key {
	return &Key{
		PrivateKey:  privateKey,
		ExtendedKey: extendedKey,
	}
}
