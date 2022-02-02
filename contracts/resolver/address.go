package resolver

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"), // mainnet
	3: common.HexToAddress("0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"), // ropsten
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
