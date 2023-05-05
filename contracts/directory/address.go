package directory

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	420: common.HexToAddress("0x22EE86A14b49890957fE71990073C6C68E05F9C5"), // optimism goerli testnet
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
