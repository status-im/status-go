package requests

import (
	"errors"

	"github.com/status-im/status-go/services/typeddata"
)

// SignTypedData represents a request to sign typed data.
type SignTypedData struct {
	// TypedData is the typed data to sign.
	TypedData typeddata.TypedData `json:"typedData"`

	// Address is the address of the account to sign with.
	Address string `json:"address"`

	// Password is the password of the account to sign with.
	Password string `json:"password"`
}

// Validate checks the validity of the SignTypedData request.
var (
	ErrSignTypedDataEmptyTypedData = errors.New("sign-typed-data: TypedData cannot be empty")
	ErrSignTypedDataEmptyAddress   = errors.New("sign-typed-data: Address cannot be empty")
	ErrSignTypedDataEmptyPassword  = errors.New("sign-typed-data: Password cannot be empty")
)

func (r *SignTypedData) Validate() error {
	if err := r.TypedData.Validate(); err != nil {
		return ErrSignTypedDataEmptyTypedData
	}
	if r.Address == "" {
		return ErrSignTypedDataEmptyAddress
	}
	if r.Password == "" {
		return ErrSignTypedDataEmptyPassword
	}
	return nil
}
