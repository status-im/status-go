package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrEditMessageInvalidID = errors.New("edit-message: invalid id")
var ErrEditMessageInvalidText = errors.New("edit-message: invalid text")

type EditMessage struct {
	ID   types.HexBytes `json:"id"`
	Text string         `json:"text"`
}

func (e *EditMessage) Validate() error {
	if len(e.ID) == 0 {
		return ErrEditMessageInvalidID
	}

	if len(e.Text) == 0 {
		return ErrEditMessageInvalidText
	}

	return nil
}
