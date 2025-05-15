package requests

import (
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/v10/server/pairing"
)

type InputConnectionStringForImportingKeypairsKeystores struct {
	ConnectionString                  string                                     `json:"connectionString" validate:"required"`
	KeystoreFilesReceiverClientConfig *pairing.KeystoreFilesReceiverClientConfig `json:"keystoreFilesReceiverClientConfig" validate:"required"`
}

func (r *InputConnectionStringForImportingKeypairsKeystores) Validate() error {
	return validator.New().Struct(r)
}
