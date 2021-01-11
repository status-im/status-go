package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrCreateOneToOneChatInvalidID = errors.New("create-one-to-one-chat: invalid id")

type CreateOneToOneChat struct {
	ID types.HexBytes
}

func (j *CreateOneToOneChat) Validate() error {
	if len(j.ID) == 0 {
		return ErrCreateOneToOneChatInvalidID
	}

	return nil
}
