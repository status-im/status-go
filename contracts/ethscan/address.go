package ethscan

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), // mainnet
	3: common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), // ropsten
	4: common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), // rinkeby
	5: common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), // goerli
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
