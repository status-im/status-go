package ensresolver

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

var errorNotAvailableOnChainID = errors.New("nameWrapper contract address not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	walletCommon.EthereumMainnet: common.HexToAddress("0xD4416b13d2b3a9aBae7AcD5D6C2BbDBE25686401"), // mainnet
	walletCommon.EthereumSepolia: common.HexToAddress("0x0635513f179D50A207757E05759CbD106d7dFcE8"), // sepolia testnet
}

// GetNameWrapperAddress returns the address of the ENS Name Wrapper contract for a given chain ID.
// source: https://docs.ens.domains/wrapper/contracts/
func NameWrapperContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
