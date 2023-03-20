package ethscan

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	1:      common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), // mainnet
	5:      common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), // goerli
	10:     common.HexToAddress("0x9e5076df494fc949abc4461f4e57592b81517d81"), // optimism
	420:    common.HexToAddress("0xf532c75239fa61b66d31e73f44300c46da41aadd"), // goerli optimism
	42161:  common.HexToAddress("0xbb85398092b83a016935a17fc857507b7851a071"), // arbitrum
	421613: common.HexToAddress("0xec21ebe1918e8975fc0cd0c7747d318c00c0acd5"), // goerli arbitrum
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
