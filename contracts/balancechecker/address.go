package balancechecker

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	wallet_common "github.com/status-im/status-go/services/wallet/common"
)

var errorNotAvailableOnChainID = errors.New("BalanceChecker not available for chainID")

var contractDataByChainID = map[uint64]common.Address{
	wallet_common.EthereumMainnet:      common.HexToAddress("0x040EA8bFE441597849A9456182fa46D38B75BC05"),
	wallet_common.OptimismMainnet:      common.HexToAddress("0x55bD303eA3D50FC982A8a5b43972d7f38D129bbF"),
	wallet_common.ArbitrumMainnet:      common.HexToAddress("0x54764eF12d29b249fDC7FC3caDc039955A396A8e"),
	wallet_common.BaseMainnet:          common.HexToAddress("0x84A1C94fcc5EcFA292771f6aE7Fbf24ec062D34e"),
	wallet_common.BSCMainnet:           common.HexToAddress("0xaf9ac152537801c562c0ea14ad186f4f4946b53d"),
	wallet_common.EthereumSepolia:      common.HexToAddress("0x55bD303eA3D50FC982A8a5b43972d7f38D129bbF"),
	wallet_common.ArbitrumSepolia:      common.HexToAddress("0x54764eF12d29b249fDC7FC3caDc039955A396A8e"),
	wallet_common.OptimismSepolia:      common.HexToAddress("0x55bD303eA3D50FC982A8a5b43972d7f38D129bbF"),
	wallet_common.BaseSepolia:          common.HexToAddress("0x84A1C94fcc5EcFA292771f6aE7Fbf24ec062D34e"),
	wallet_common.StatusNetworkSepolia: common.HexToAddress("0x84A1C94fcc5EcFA292771f6aE7Fbf24ec062D34e"),
	wallet_common.BSCTestnet:           common.HexToAddress("0xaf9ac152537801c562c0ea14ad186f4f4946b53d"),
	wallet_common.TestnetChainID:       common.HexToAddress("0x0000000000000000000000000000000010777333"),
}

func ContractAddress(chainID uint64) (common.Address, error) {
	contract, exists := contractDataByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return contract, nil
}
