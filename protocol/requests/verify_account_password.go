package requests

import (
	"gopkg.in/go-playground/validator.v9"
)

// VerifyAccountPassword represents a request to verify an account password.
type VerifyAccountPassword struct {
	// KeyStoreDir is the directory where the keystore files are located.
	KeyStoreDir string `json:"keyStoreDir" validate:"required"`

	// Address is the Ethereum address for the account.
	Address string `json:"address" validate:"required"`

	// Password is the password to verify against the keystore.
	Password string `json:"password" validate:"required"`
}

// Validate checks the validity of the VerifyAccountPassword request.
func (v *VerifyAccountPassword) Validate() error {
	return validator.New().Struct(v)
}
