package requests

import (
	"errors"
)

// VerifyDatabasePassword represents a request to verify the database password.
type VerifyDatabasePassword struct {
	// KeyUID identifies the specific key in the database.
	KeyUID string `json:"keyUID"`

	// Password is the password to verify against the database entry.
	Password string `json:"password"`
}

// Validate checks the validity of the VerifyDatabasePasswordV2 request.
var (
	ErrVerifyDatabasePasswordEmptyKeyUID   = errors.New("verify-database-password: KeyUID cannot be empty")
	ErrVerifyDatabasePasswordEmptyPassword = errors.New("verify-database-password: Password cannot be empty")
)

func (v *VerifyDatabasePassword) Validate() error {
	if v.KeyUID == "" {
		return ErrVerifyDatabasePasswordEmptyKeyUID
	}
	if v.Password == "" {
		return ErrVerifyDatabasePasswordEmptyPassword
	}
	return nil
}
