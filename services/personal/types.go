package personal

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

// SignParams required to sign messages
type SignParams struct {
	Data     interface{} `json:"data"`
	Address  string      `json:"account"`
	Password string      `json:"password"`
}

func (sp *SignParams) Validate(checkPassword bool) error {
	if len(sp.Address) != 2*types.AddressLength+2 {
		return errors.New("address has to be provided")
	}

	if sp.Data == "" {
		return errors.New("data has to be provided")
	}

	if checkPassword && sp.Password == "" {
		return errors.New("password has to be provided")
	}

	return nil
}

// RecoverParams are for calling `personal_ecRecover`
type RecoverParams struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
}
