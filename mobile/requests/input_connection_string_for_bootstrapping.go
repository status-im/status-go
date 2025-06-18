package requests

import (
	"github.com/status-im/status-go/server/pairing"
)

type InputConnectionStringForBootstrapping struct {
	ConnectionString     string                        `json:"connectionString" validate:"required"`
	ReceiverClientConfig *pairing.ReceiverClientConfig `json:"receiverClientConfig" validate:"required"`
}

func (r *InputConnectionStringForBootstrapping) Validate() error {
	return pairing.ValidateStruct(r)
}
