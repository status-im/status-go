package requests

import (
	"errors"

	"github.com/status-im/status-go/internal/crypto/types"
)

type ReevaluateCommunityMembersPermissions struct {
	CommunityID types.HexBytes `json:"communityId"`
	// Force runs immediate reevaluation on the control node. Requires forcing to be enabled on the node.
	Force bool `json:"force"`
}

func (r *ReevaluateCommunityMembersPermissions) Validate() error {
	if r.CommunityID == nil || len(r.CommunityID) == 0 {
		return errors.New("reevaluate community members permissions does not contain communityID")
	}

	return nil
}
