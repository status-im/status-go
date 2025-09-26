package multistandardfetcher

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type AccountAddress = common.Address
type ContractAddress = common.Address
type HashableTokenID = [32]byte // *big.Int not comparable, use u256 fixed length array

type CollectibleID struct {
	ContractAddress ContractAddress
	TokenID         *big.Int
}

type HashableCollectibleID struct {
	ContractAddress ContractAddress
	TokenID         HashableTokenID
}

func (h HashableCollectibleID) ToCollectibleID() CollectibleID {
	return CollectibleID{
		ContractAddress: h.ContractAddress,
		TokenID:         new(big.Int).SetBytes(h.TokenID[:]),
	}
}

func (id CollectibleID) ToHashableCollectibleID() HashableCollectibleID {
	ret := HashableCollectibleID{
		ContractAddress: id.ContractAddress,
	}
	id.TokenID.FillBytes(ret.TokenID[:])
	return ret
}
