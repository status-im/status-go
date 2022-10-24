package ethscan

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), // mainnet
	5: common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), // goerli
	420: common.HexToAddress("0xf532c75239fa61b66d31e73f44300c46da41aadd"), // goerli optimism
	421613: common.HexToAddress("0xec21ebe1918e8975fc0cd0c7747d318c00c0acd5"), // goerli arbitrum
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
