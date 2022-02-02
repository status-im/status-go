package registrar

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0xDB5ac1a559b02E12F29fC0eC0e37Be8E046DEF49"), // mainnet
	3: common.HexToAddress("0xdaae165beb8c06e0b7613168138ebba774aff071"), // ropsten
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
