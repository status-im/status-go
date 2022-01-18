package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrDismissContactRequestInvalidID = errors.New("dismiss-contact-request: invalid id")

type DismissContactRequest struct {
	ID types.HexBytes `json:"id"`
}

func (a *DismissContactRequest) Validate() error {
	if len(a.ID) == 0 {
		return ErrDismissContactRequestInvalidID
	}

	return nil
}
