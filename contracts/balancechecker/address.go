package balancechecker

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("BalanceChecker not available for chainID")

var contractDataByChainID = map[uint64]common.Address{
	1:        common.HexToAddress("0x040EA8bFE441597849A9456182fa46D38B75BC05"), // mainnet
	5:        common.HexToAddress("0xA5522A3194B78Dd231b64d0ccd6deA6156DCa7C8"), // goerli
	10:       common.HexToAddress("0x55bD303eA3D50FC982A8a5b43972d7f38D129bbF"), // optimism
	42161:    common.HexToAddress("0x55bD303eA3D50FC982A8a5b43972d7f38D129bbF"), // arbitrum
	777333:   common.HexToAddress("0x0000000000000000000000000000000010777333"), // unit tests
}

func ContractAddress(chainID uint64) (common.Address, error) {
	contract, exists := contractDataByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return contract, nil
}
