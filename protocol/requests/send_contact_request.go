package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrSendContactRequestInvalidID = errors.New("send-contact-request: invalid id")
var ErrSendContactRequestInvalidMessage = errors.New("send-contact-request: invalid message")

type SendContactRequest struct {
	ID      types.HexBytes `json:"id"`
	Message string         `json:"message"`
}

func (a *SendContactRequest) Validate() error {
	if len(a.ID) == 0 {
		return ErrSendContactRequestInvalidID
	}

	if len(a.Message) == 0 {
		return ErrSendContactRequestInvalidMessage
	}

	return nil
}
