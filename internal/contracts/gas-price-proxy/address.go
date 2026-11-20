package gaspriceproxy

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	wallet_common "github.com/status-im/status-go/pkg/services/wallet/common"
)

var ErrorNotAvailableOnChainID = errors.New("not available for chainID")

var (
	opStackOracle = common.HexToAddress("0x420000000000000000000000000000000000000F")
	scrollOracle  = common.HexToAddress("0x5300000000000000000000000000000000000002")
)

// L1 fee oracles (`getL1Fee(bytes)`) of the chains that charge an L1 data fee.
var contractAddressByChainID = map[uint64]common.Address{
	wallet_common.OptimismMainnet: opStackOracle,
	wallet_common.OptimismSepolia: opStackOracle,
	wallet_common.BaseMainnet:     opStackOracle,
	wallet_common.BaseSepolia:     opStackOracle,
	wallet_common.UnichainMainnet: opStackOracle,
	wallet_common.UnichainSepolia: opStackOracle,
	wallet_common.InkMainnet:      opStackOracle,
	wallet_common.InkSepolia:      opStackOracle,
	wallet_common.SoneiumMainnet:  opStackOracle,
	wallet_common.SoneiumMinato:   opStackOracle,
	wallet_common.KatanaMainnet:   opStackOracle,
	wallet_common.KatanaBokuto:    opStackOracle,
	wallet_common.ScrollMainnet:   scrollOracle,
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return common.Address{}, ErrorNotAvailableOnChainID
	}
	return addr, nil
}

// ChargesL1Fee reports whether transactions on the chain pay an L1 data fee on
// top of L2 gas.
func ChargesL1Fee(chainID uint64) bool {
	_, exists := contractAddressByChainID[chainID]
	return exists
}
