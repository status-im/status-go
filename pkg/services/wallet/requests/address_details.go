package requests

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var ErrAddresInvalid = errors.New("address-details: invalid address")

type AddressDetails struct {
	Address               string   `json:"address"`
	ChainIDs              []uint64 `json:"chainIds"`
	TimeoutInMilliseconds int64    `json:"timeoutInMilliseconds"`
}

func (a *AddressDetails) Validate() error {
	if !common.IsHexAddress(a.Address) {
		return ErrAddresInvalid
	}

	return nil
}
