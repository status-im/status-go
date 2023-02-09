package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

var ErrCreateCommunityPermissionInvalidCommunityID = errors.New("create-community-permission: invalid community id")

type CreateCommunityPermission struct {
	CommunityID types.HexBytes                            `json:"communityId"`
	IsAllowedTo protobuf.CommunityPermission_AllowedTypes `json:"isAllowedTo"`
	Private     bool                                      `json:"private"`
	ChatIDs     []string                                  `json:"chatIds"`
}

func (j *CreateCommunityPermission) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrCreateCommunityCategoryInvalidCommunityID
	}

	return nil
}
