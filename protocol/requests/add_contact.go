package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrAddContactInvalidID = errors.New("add-contact: invalid id")

type AddContact struct {
	ID       types.HexBytes `json:"id"`
	Nickname string         `json:"nickname"`
}

func (a *AddContact) Validate() error {
	if len(a.ID) == 0 {
		return ErrAddContactInvalidID
	}

	return nil
}
