package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrRequestToJoinCommunityInvalidCommunityID = errors.New("request-to-join-community: invalid community id")
var ErrRequestToJoinCommunityMissingPassword = errors.New("request-to-join-community: password is necessary when sending a list of addresses")

type RequestToJoinCommunity struct {
	CommunityID       types.HexBytes `json:"communityId"`
	ENSName           string         `json:"ensName"`
	Password          string         `json:"password"`
	AddressesToReveal []string       `json:"addressesToReveal"`
}

func (j *RequestToJoinCommunity) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrRequestToJoinCommunityInvalidCommunityID
	}
	if len(j.AddressesToReveal) > 0 && j.Password == "" {
		return ErrRequestToJoinCommunityMissingPassword
	}

	return nil
}
