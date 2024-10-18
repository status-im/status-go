package requests

import (
	"gopkg.in/go-playground/validator.v9"
)

// VerifyDatabasePassword represents a request to verify the database password.
type VerifyDatabasePassword struct {
	// KeyUID identifies the specific key in the database.
	KeyUID string `json:"keyUID" validate:"required"`

	// Password is the password to verify against the database entry.
	Password string `json:"password" validate:"required"`
}

// Validate checks the validity of the VerifyDatabasePassword request.
func (v *VerifyDatabasePassword) Validate() error {
	return validator.New().Struct(v)
}
