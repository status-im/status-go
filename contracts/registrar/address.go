package registrar

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0xDB5ac1a559b02E12F29fC0eC0e37Be8E046DEF49"), // mainnet
	3: common.HexToAddress("0xdaae165beb8c06e0b7613168138ebba774aff071"), // ropsten
	5: common.HexToAddress("0xD1f7416F91E7Eb93dD96A61F12FC092aD6B67B11"), //goerli
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
