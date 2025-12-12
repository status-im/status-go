package directory

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	wallet_common.OptimismMainnet: common.HexToAddress("0xA8d270048a086F5807A8dc0a9ae0e96280C41e3A"),
	wallet_common.OptimismSepolia: common.HexToAddress("0x6B94e21FAB8Af38E8d89dd4A0480C04e9a5c53Ab"),
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
