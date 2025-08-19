package requests

import (
	"gopkg.in/go-playground/validator.v9"
)

// DeleteMultiaccount represents a request to delete a multiaccount.
type DeleteMultiaccount struct {
	// KeyUID is the unique identifier for the key.
	KeyUID string `json:"keyUID" validate:"required"`
}

// Validate checks the validity of the DeleteMultiaccount request.
func (v *DeleteMultiaccount) Validate() error {
	return validator.New().Struct(v)
}
