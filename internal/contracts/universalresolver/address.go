package universalresolver

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

var errorNotAvailableOnChainID = errors.New("universal resolver not available for chainID")

// canonicalAddress is the ENS Universal Resolver deployed at the same address on
// Ethereum mainnet and its testnets (a proxy owned by the ENS DAO).
// source: https://docs.ens.domains/resolvers/universal/
var canonicalAddress = common.HexToAddress("0xeEeEEEeE14D718C2B47D9923Deab1335E144EeEe")

// contractAddressByChainID maps a chainID to its ENS Universal Resolver.
var contractAddressByChainID = map[uint64]common.Address{
	walletCommon.EthereumMainnet: canonicalAddress,
	walletCommon.EthereumSepolia: canonicalAddress,
	// Anvil (local functional tests): the canonical Universal Resolver has no
	// Anvil deployment, so test/functional/deploy_contracts.sh deploys a minimal
	// stand-in at a deterministic address. Keep this in sync with that deploy.
	walletCommon.AnvilMainnet: common.HexToAddress("0x68B1D87F95878fE05B998F19b66F4baba5De1aed"),
}

// ContractAddress returns the ENS Universal Resolver address for a given chain ID.
func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
