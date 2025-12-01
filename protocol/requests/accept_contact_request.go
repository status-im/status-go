package requests

import (
	"errors"

	"github.com/status-im/status-go/crypto/types"
)

var ErrAcceptContactRequestInvalidID = errors.New("accept-contact-request: invalid id")
var ErrAcceptContactRequestInvalidContactID = errors.New("accept-contact-request: invalid contact id")

type AcceptContactRequest struct {
	ID        types.HexBytes `json:"id"`
	ContactID string         `json:"contactId"`
}

func (a *AcceptContactRequest) Validate() error {
	if len(a.ID) == 0 {
		return ErrAcceptContactRequestInvalidID
	}
	if len(a.ContactID) == 0 {
		return ErrAcceptContactRequestInvalidContactID
	}

	return nil
}
