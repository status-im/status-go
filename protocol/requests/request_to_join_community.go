package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrRequestToJoinCommunityInvalidCommunityID = errors.New("request-to-join-community: invalid community id")

type RequestToJoinCommunity struct {
	CommunityID types.HexBytes `json:"communityId"`
	ENSName     string         `json:"ensName"`
	Password    string         `json:"password"`
}

func (j *RequestToJoinCommunity) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrRequestToJoinCommunityInvalidCommunityID
	}

	return nil
}
