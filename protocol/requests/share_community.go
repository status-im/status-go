package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrShareCommunityInvalidID = errors.New("share-community: invalid id")
var ErrShareCommunityEmptyUsers = errors.New("share-community: empty users")

type ShareCommunity struct {
	CommunityID types.HexBytes
	Users       []types.HexBytes
}

func (j *ShareCommunity) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrShareCommunityInvalidID
	}

	if len(j.Users) == 0 {
		return ErrShareCommunityEmptyUsers
	}

	return nil
}
