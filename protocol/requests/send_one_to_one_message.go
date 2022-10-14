package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrSendOneToOneMessageInvalidID = errors.New("send-one-to-one-message: invalid id")
var ErrSendOneToOneMessageInvalidMessage = errors.New("send-one-to-one-message: invalid message")

type SendOneToOneMessage struct {
	ID      types.HexBytes `json:"id"`
	Message string         `json:"message"`
}

func (a *SendOneToOneMessage) Validate() error {
	if len(a.ID) == 0 {
		return ErrSendOneToOneMessageInvalidID
	}

	if len(a.Message) == 0 {
		return ErrSendOneToOneMessageInvalidMessage
	}

	return nil
}
