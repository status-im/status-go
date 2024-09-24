package requests

import (
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/multiaccounts"
)

// MigrateKeystoreDir represents a request to migrate keystore directory.
type MigrateKeystoreDir struct {
	// Account is the account associated with the keystore.
	Account multiaccounts.Account `json:"account"`

	// Password is the password for the keystore.
	Password string `json:"password" validate:"required"`

	// OldDir is the old keystore directory.
	OldDir string `json:"oldDir" validate:"required"`

	// NewDir is the new keystore directory.
	NewDir string `json:"newDir" validate:"required"`
}

// Validate checks the validity of the MigrateKeystoreDir request.
func (r *MigrateKeystoreDir) Validate() error {
	return validator.New().Struct(r)
}
