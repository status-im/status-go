package requests

import (
	"github.com/status-im/status-go/server/pairing"
	"gopkg.in/go-playground/validator.v9"
)

type InputConnectionStringForBootstrapping struct {
	ConnectionString     string                        `json:"connectionString" validate:"required"`
	ReceiverClientConfig *pairing.ReceiverClientConfig `json:"receiverClientConfig" validate:"required"`
}

func (r *InputConnectionStringForBootstrapping) Validate() error {
	return validator.New().Struct(r)
}
