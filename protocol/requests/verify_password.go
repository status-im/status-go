package requests

import "gopkg.in/go-playground/validator.v9"

type VerifyPassword struct {
	// Password is the password to verify against the keystore.
	Password string `json:"password" validate:"required"`
}

func (v *VerifyPassword) Validate() error {
	return validator.New().Struct(v)
}
