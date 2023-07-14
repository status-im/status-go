package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrInvalidCommunityID = errors.New("invalid community id")
var ErrMissingPassword = errors.New("password is necessary when sending a list of addresses")
var ErrMissingSharedAddresses = errors.New("list of shared addresses is needed")
var ErrMissingAirdropAddress = errors.New("airdropAddress is needed")

type EditSharedAddresses struct {
	CommunityID       types.HexBytes `json:"communityId"`
	Password          string         `json:"password"`
	AddressesToReveal []string       `json:"addressesToReveal"`
	AirdropAddress    string         `json:"airdropAddress"`
}

func (j *EditSharedAddresses) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrInvalidCommunityID
	}
	if j.Password == "" {
		return ErrMissingPassword
	}
	if len(j.AddressesToReveal) == 0 {
		return ErrMissingSharedAddresses
	}
	if j.AirdropAddress == "" {
		return ErrMissingAirdropAddress
	}

	return nil
}
