package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrGetCommunityPermissionsInvalidCommunityID = errors.New("get-community-permissions: invalid community id")

type GetCommunityPermissions struct {
	CommunityID types.HexBytes `json:"communityId"`
}

func (j *GetCommunityPermissions) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrGetCommunityPermissionsInvalidCommunityID
	}

	return nil
}
