package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrRetractContactRequestInvalidContactID = errors.New("retract-contact-request: invalid id")

type RetractContactRequest struct {
	ContactID types.HexBytes `json:"contactId"`
}

func (a *RetractContactRequest) Validate() error {
	if len(a.ContactID) == 0 {
		return ErrRetractContactRequestInvalidContactID
	}

	return nil
}
