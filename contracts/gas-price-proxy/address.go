package gaspriceproxy

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	wallet_common "github.com/status-im/status-go/services/wallet/common"
)

var ErrorNotAvailableOnChainID = errors.New("not available for chainID")

// Addresses of the proxy contract `Proxy` on different chains.
var contractAddressByChainID = map[uint64]common.Address{
	wallet_common.OptimismMainnet: common.HexToAddress("0x420000000000000000000000000000000000000F"),
	wallet_common.OptimismSepolia: common.HexToAddress("0x420000000000000000000000000000000000000F"),
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return common.Address{}, ErrorNotAvailableOnChainID
	}
	return addr, nil
}
