package requests

import (
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/services/wallet/wallettypes"
)

// SendTransaction represents a request to send a transaction.
type SendTransaction struct {
	TxArgs   wallettypes.SendTxArgs `json:"txArgs"`
	Password string                 `json:"password" validate:"required"`
}

// Validate checks the fields of SendTransaction to ensure they meet the requirements.
func (st *SendTransaction) Validate() error {
	validate := validator.New()
	return validate.Struct(st)
}
