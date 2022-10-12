package directory

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	69:  common.HexToAddress("0x4BbCCa869E9931280Cb46AE0DfF18881Be581a4d"), // optimism kovan testnet
	420: common.HexToAddress("0xecc8e76abc781c411f7bf79f3b1254b66afe3b75"), // optimism goerli testnet
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
