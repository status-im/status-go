package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

var ErrUpdateCommunityPermissionInvalidPermissionID = errors.New("update-community-permission: invalid permission id")
var ErrUpdateCommunityPermissionInvalidCommunityID = errors.New("update-community-permission: invalid community id")

type UpdateCommunityPermission struct {
	CommunityID  types.HexBytes                            `json:"communityId"`
	PermissionID string                                    `json:"permissionId"`
	IsAllowedTo  protobuf.CommunityPermission_AllowedTypes `json:"isAllowedTo"`
	Hidden       bool                                      `json:"hidden"`
	HoldsTokens  bool                                      `json:"holdsTokens"`
	ChatIDs      []string                                  `json:"chatIds"`
}

func (j *UpdateCommunityPermission) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrUpdateCommunityPermissionInvalidCommunityID
	}

	if len(j.PermissionID) == 0 {
		return ErrUpdateCommunityPermissionInvalidPermissionID
	}

	return nil
}
