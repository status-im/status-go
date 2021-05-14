package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrInviteUsersToCommunityInvalidID = errors.New("invite-users-to-community: invalid id")
var ErrInviteUsersToCommunityEmptyUsers = errors.New("invite-users-to-community: empty users")

type InviteUsersToCommunity struct {
	CommunityID types.HexBytes   `json:"communityId"`
	Users       []types.HexBytes `json:"users"`
}

func (j *InviteUsersToCommunity) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrInviteUsersToCommunityInvalidID
	}

	if len(j.Users) == 0 {
		return ErrInviteUsersToCommunityEmptyUsers
	}

	return nil
}
