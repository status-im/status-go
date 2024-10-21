package requests

import (
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/server/pairing"
)

type InputConnectionStringForBootstrappingAnotherDevice struct {
	ConnectionString   string                      `json:"connectionString" validate:"required"`
	SenderClientConfig *pairing.SenderClientConfig `json:"senderClientConfig" validate:"required"`
}

func (r *InputConnectionStringForBootstrappingAnotherDevice) Validate() error {
	return validator.New().Struct(r)
}
