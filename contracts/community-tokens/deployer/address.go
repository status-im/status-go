package communitytokendeployer

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

var errorNotAvailableOnChainID = errors.New("deployer contract not available for chainID")

// addresses can be found on https://github.com/status-im/communities-contracts#deployments
var contractAddressByChainID = map[uint64]common.Address{
	walletCommon.EthereumMainnet:      common.HexToAddress("0xB3Ef5B0825D5f665bE14394eea41E684CE96A4c5"),
	walletCommon.OptimismMainnet:      common.HexToAddress("0x31463D22750324C8721FF7751584EF62F2ff93b3"),
	walletCommon.ArbitrumMainnet:      common.HexToAddress("0x744Fd6e98dad09Fb8CCF530B5aBd32B56D64943b"),
	walletCommon.BaseMainnet:          common.HexToAddress("0x898331B756EE1f29302DeF227a4471e960c50612"),
	walletCommon.EthereumSepolia:      common.HexToAddress("0xCDE984e57cdb88c70b53437cc694345B646371f9"),
	walletCommon.ArbitrumSepolia:      common.HexToAddress("0x7Ff554af5b6624db2135E4364F416d1D397f43e6"),
	walletCommon.OptimismSepolia:      common.HexToAddress("0xcE2A896eEA2F585BC0C3753DC8116BbE2AbaE541"),
	walletCommon.BaseSepolia:          common.HexToAddress("0x7Ff554af5b6624db2135E4364F416d1D397f43e6"),
	walletCommon.StatusNetworkSepolia: common.HexToAddress("0x06716eCfA9B5Ae0210F4c3279cA304A7F563c59e"),
	walletCommon.BSCMainnet:           common.HexToAddress("0x76d0e484e7c3398922636960ab33bde6e9936d81"),
	walletCommon.BSCTestnet:           common.HexToAddress("0x7Ff554af5b6624db2135E4364F416d1D397f43e6"),
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
