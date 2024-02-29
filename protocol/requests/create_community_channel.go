package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrCreateCommunityChannelInvalidCommunityID = errors.New("create-community-channel: invalid community id")
var ErrCreateCommunityChannelInvalidName = errors.New("create-community-channel: invalid channel name")

type CreateCommunityChannel struct {
	CommunityID types.HexBytes `json:"communityId"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Color       string         `json:"color"`
	CategoryID  string         `json:"categoryId"`
	Position    int32          `json:"position"`
}

func (j *CreateCommunityChannel) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrCreateCommunityChannelInvalidCommunityID
	}

	if len(j.Name) == 0 {
		return ErrCreateCommunityChannelInvalidName
	}

	return nil
}
