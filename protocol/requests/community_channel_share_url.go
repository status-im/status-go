package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var (
	ErrCommunityChannelShareURLCommunityInvalidID = errors.New("check-permission-to-join-community: invalid id")
)

type CommunityChannelShareURL struct {
	CommunityID types.HexBytes
	ChannelID   string
}

func (r *CommunityChannelShareURL) Validate() error {
	if len(r.CommunityID) == 0 {
		return ErrCheckPermissionToJoinCommunityInvalidID
	}

	return nil
}
