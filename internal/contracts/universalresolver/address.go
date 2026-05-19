package universalresolver

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// CanonicalAddress is the ENS DAO-owned upgradable proxy for the Universal
// Resolver. Per the ENS docs this same address is used on Mainnet and
// supported testnets. The proxy address is stable across implementation
// upgrades.
var CanonicalAddress = common.HexToAddress("0xeEeEEEeE14D718C2B47D9923Deab1335E144EeEe")

var errorNotAvailableOnChainID = errors.New("universal resolver address not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	1:        CanonicalAddress, // mainnet
	11155111: CanonicalAddress, // sepolia
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return common.Address{}, errorNotAvailableOnChainID
	}
	return addr, nil
}
