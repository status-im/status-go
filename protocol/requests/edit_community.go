package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"

	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	ErrEditCommunityInvalidID          = errors.New("edit-community: invalid id")
	ErrEditCommunityInvalidName        = errors.New("edit-community: invalid name")
	ErrEditCommunityInvalidColor       = errors.New("edit-community: invalid color")
	ErrEditCommunityInvalidDescription = errors.New("edit-community: invalid description")
	ErrEditCommunityInvalidMembership  = errors.New("edit-community: invalid membership")
)

type EditCommunity struct {
	CommunityID types.HexBytes
	CreateCommunity
}

func (u *EditCommunity) Validate() error {

	if len(u.CommunityID) == 0 {
		return ErrEditCommunityInvalidID
	}

	if u.Name == "" {
		return ErrEditCommunityInvalidName
	}

	if u.Description == "" {
		return ErrEditCommunityInvalidDescription
	}

	if u.Membership == protobuf.CommunityPermissions_UNKNOWN_ACCESS {
		return ErrEditCommunityInvalidMembership
	}

	if u.Color == "" {
		return ErrEditCommunityInvalidColor
	}

	return nil
}
