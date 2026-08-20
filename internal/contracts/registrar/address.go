package registrar

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

var errorNotAvailableOnChainID = errors.New("username registrar not available for chainID")

// contractAddressByChainID maps a chainID to the Status username registrar
// (the contract that owns `stateofus.eth` and handles registering/releasing
// `*.stateofus.eth` names). These are stable, hardcoded so we don't need to
// discover them dynamically through the ENS registry's `owner()` call.
var contractAddressByChainID = map[uint64]common.Address{
	walletCommon.EthereumMainnet: common.HexToAddress("0xDB5ac1a559b02E12F29fC0eC0e37Be8E046DEF49"),
	walletCommon.EthereumSepolia: common.HexToAddress("0x3174DceF2649Ef7D3585cFC52d45586cF0d0dC90"),
	// Anvil (local functional tests): the username registrar is deployed on a
	// fresh Anvil from account #0 by test/functional/deploy_contracts.sh, which
	// makes its address deterministic. Keep this in sync with the deployed
	// registrar if the deploy scripts or their contract dependencies change.
	walletCommon.AnvilMainnet: common.HexToAddress("0x9A676e781A523b5d0C0e43731313A708CB607508"),
}

// ContractAddress returns the Status username registrar address for a given chain ID.
func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
