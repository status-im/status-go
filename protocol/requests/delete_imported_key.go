package requests

import (
	"gopkg.in/go-playground/validator.v9"
)

// DeleteImportedKey represents a request to delete an imported key.
type DeleteImportedKey struct {
	// Address is the address of the imported key to delete.
	Address string `json:"address" validate:"required"`

	// Password is the password used to decrypt the key.
	Password string `json:"password" validate:"required"`

	// KeyStoreDir is the directory where the key is stored.
	KeyStoreDir string `json:"keyStoreDir" validate:"required"`
}

// Validate checks the validity of the DeleteImportedKey request.
func (r *DeleteImportedKey) Validate() error {
	return validator.New().Struct(r)
}
