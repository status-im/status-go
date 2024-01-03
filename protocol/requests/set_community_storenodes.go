package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/mailservers"
)

var (
	ErrSetCommunityStorenodesEmpty   = errors.New("set-community-storenodes: empty payload")
	ErrSetCommunityStorenodesTooMany = errors.New("set-community-storenodes: too many")
)

type SetCommunityStorenodes struct {
	CommunityID types.HexBytes           `json:"communityId"`
	Storenodes  []mailservers.Mailserver `json:"storenodes"`
}

func (s *SetCommunityStorenodes) Validate() error {
	if s == nil || len(s.Storenodes) == 0 {
		return ErrSetCommunityStorenodesEmpty
	}
	if len(s.Storenodes) > 1 {
		// TODO for now only allow one
		return ErrSetCommunityStorenodesTooMany
	}
	return nil
}
