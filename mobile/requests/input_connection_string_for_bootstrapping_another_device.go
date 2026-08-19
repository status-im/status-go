package requests

import (
	"github.com/status-im/status-go/services/pairing"
)

type InputConnectionStringForBootstrappingAnotherDevice struct {
	ConnectionString   string                      `json:"connectionString" validate:"required"`
	SenderClientConfig *pairing.SenderClientConfig `json:"senderClientConfig" validate:"required"`
}

func (r *InputConnectionStringForBootstrappingAnotherDevice) Validate() error {
	return pairing.ValidateStruct(r)
}
