package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrGetCommunityAccessRolesWithBalancesMissingID = errors.New("CommunityAccessRolesWithBalances: missing community ID")

type GetCommunityAccessRolesWithBalances struct {
	CommunityID types.HexBytes `json:"communityId"`
}

func (r *GetCommunityAccessRolesWithBalances) Validate() error {
	if len(r.CommunityID) == 0 {
		return ErrGetCommunityAccessRolesWithBalancesMissingID
	}

	return nil
}
