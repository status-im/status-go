package requests

import (
	"errors"
)

// DeleteImportedKey represents a request to delete an imported key.
type DeleteImportedKey struct {
	// Address is the address of the imported key to delete.
	Address string `json:"address"`

	// Password is the password used to decrypt the key.
	Password string `json:"password"`

	// KeyStoreDir is the directory where the key is stored.
	KeyStoreDir string `json:"keyStoreDir"`
}

// Validate checks the validity of the DeleteImportedKey request.
var (
	ErrDeleteImportedKeyEmptyAddress     = errors.New("delete-imported-key: Address cannot be empty")
	ErrDeleteImportedKeyEmptyPassword    = errors.New("delete-imported-key: Password cannot be empty")
	ErrDeleteImportedKeyEmptyKeyStoreDir = errors.New("delete-imported-key: KeyStoreDir cannot be empty")
)

func (r *DeleteImportedKey) Validate() error {
	if r.Address == "" {
		return ErrDeleteImportedKeyEmptyAddress
	}
	if r.Password == "" {
		return ErrDeleteImportedKeyEmptyPassword
	}
	if r.KeyStoreDir == "" {
		return ErrDeleteImportedKeyEmptyKeyStoreDir
	}
	return nil
}
