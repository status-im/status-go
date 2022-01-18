package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrShareMessageInvalidID = errors.New("share-image-message: invalid id")
var ErrShareMessageEmptyUsers = errors.New("share-image-message: empty users")

type ShareImageMessage struct {
	MessageID string           `json:"id"`
	Users     []types.HexBytes `json:"users"`
}

func (j *ShareImageMessage) Validate() error {
	if len(j.MessageID) == 0 {
		return ErrShareMessageInvalidID
	}

	if len(j.Users) == 0 {
		return ErrShareMessageEmptyUsers
	}

	return nil
}
