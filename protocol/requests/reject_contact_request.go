package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrRejectContactRequestInvalidID = errors.New("reject-contact-request: invalid id")

type RejectContactRequest struct {
	ID types.HexBytes `json:"id"`
}

func (a *RejectContactRequest) Validate() error {
	if len(a.ID) == 0 {
		return ErrRejectContactRequestInvalidID
	}

	return nil
}
