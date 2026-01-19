package requests

import (
	"errors"

	"github.com/status-im/status-go/internal/crypto/types"
)

var ErrDeclineContactRequestInvalidID = errors.New("decline-contact-request: invalid id")
var ErrDeclineContactRequestInvalidContactID = errors.New("decline-contact-request: invalid contact id")

type DeclineContactRequest struct {
	ID        types.HexBytes `json:"id"`
	ContactID string         `json:"contactId"`
}

func (a *DeclineContactRequest) Validate() error {
	if len(a.ID) == 0 {
		return ErrDeclineContactRequestInvalidID
	}
	if len(a.ContactID) == 0 {
		return ErrDeclineContactRequestInvalidContactID
	}

	return nil
}
