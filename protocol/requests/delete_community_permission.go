package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrDeleteCommunityPermissionInvalidCommunityID = errors.New("delete-community-permission: invalid community id")
var ErrDeleteCommunityPermissionInvalidPermissionyID = errors.New("delete-community-permission: invalid category id")

type DeleteCommunityPermission struct {
	CommunityID  types.HexBytes `json:"communityId"`
	PermissionID string         `json:"permissionId"`
}

func (j *DeleteCommunityPermission) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrDeleteCommunityPermissionInvalidCommunityID
	}

	if len(j.PermissionID) == 0 {
		return ErrDeleteCommunityPermissionInvalidPermissionyID

	}

	return nil
}
