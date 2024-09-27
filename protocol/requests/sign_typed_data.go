package requests

import (
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/services/typeddata"
)

// SignTypedData represents a request to sign typed data.
type SignTypedData struct {
	// TypedData is the typed data to sign.
	TypedData typeddata.TypedData `json:"typedData" validate:"required"`

	// Address is the address of the account to sign with.
	Address string `json:"address" validate:"required"`

	// Password is the password of the account to sign with.
	Password string `json:"password" validate:"required"`
}

// Validate checks the validity of the SignTypedData request.
func (r *SignTypedData) Validate() error {
	// Use the validator package to validate the struct fields
	if err := validator.New().Struct(r); err != nil {
		return err
	}

	// Additional validation logic from the old signTypedData function
	if err := r.TypedData.Validate(); err != nil {
		return err
	}

	return nil
}
