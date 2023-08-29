package communitytokendeployer

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("deployer contract not available for chainID")

// TODO add addresses for other chains
var contractAddressByChainID = map[uint64]common.Address{
	420: common.HexToAddress("0xfFa8A255D905c909379859eA45B959D090DDC2d4"), // Optimism Goerli
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
