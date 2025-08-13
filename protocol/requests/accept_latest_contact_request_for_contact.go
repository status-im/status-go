package requests

import (
	"errors"

	"github.com/status-im/status-go/crypto/types"
)

var ErrAcceptLatestContactRequestForContactInvalidID = errors.New("accept-latest-contact-request-for-contact: invalid id")

type AcceptLatestContactRequestForContact struct {
	ID types.HexBytes `json:"id"`
}

func (a *AcceptLatestContactRequestForContact) Validate() error {
	if len(a.ID) == 0 {
		return ErrAcceptLatestContactRequestForContactInvalidID
	}

	return nil
}
