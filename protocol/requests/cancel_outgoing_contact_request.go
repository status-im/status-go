package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrCancelOutgoingContactRequestInvalidID = errors.New("cancel-outgoing-contact-request: invalid id")

type CancelOutgoingContactRequest struct {
	ID types.HexBytes `json:"id"`
}

func (a *CancelOutgoingContactRequest) Validate() error {
	if len(a.ID) == 0 {
		return ErrCancelOutgoingContactRequestInvalidID
	}

	return nil
}
