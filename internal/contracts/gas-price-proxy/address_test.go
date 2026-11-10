package gaspriceproxy_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	gaspriceproxy "github.com/status-im/status-go/internal/contracts/gas-price-proxy"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

var (
	opStackOracle = common.HexToAddress("0x420000000000000000000000000000000000000F")
	scrollOracle  = common.HexToAddress("0x5300000000000000000000000000000000000002")
)

func TestContractAddress_ChainsThatChargeAnL1Fee(t *testing.T) {
	cases := map[string]struct {
		chainID uint64
		oracle  common.Address
	}{
		"Optimism":         {walletCommon.OptimismMainnet, opStackOracle},
		"Optimism Sepolia": {walletCommon.OptimismSepolia, opStackOracle},
		"Base":             {walletCommon.BaseMainnet, opStackOracle},
		"Base Sepolia":     {walletCommon.BaseSepolia, opStackOracle},
		"Unichain":         {walletCommon.UnichainMainnet, opStackOracle},
		"Unichain Sepolia": {walletCommon.UnichainSepolia, opStackOracle},
		"Ink":              {walletCommon.InkMainnet, opStackOracle},
		"Ink Sepolia":      {walletCommon.InkSepolia, opStackOracle},
		"Soneium":          {walletCommon.SoneiumMainnet, opStackOracle},
		"Soneium Minato":   {walletCommon.SoneiumMinato, opStackOracle},
		"Katana":           {walletCommon.KatanaMainnet, opStackOracle},
		"Katana Bokuto":    {walletCommon.KatanaBokuto, opStackOracle},
		"Scroll":           {walletCommon.ScrollMainnet, scrollOracle},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			addr, err := gaspriceproxy.ContractAddress(tc.chainID)
			require.NoError(t, err)
			require.Equal(t, tc.oracle, addr)
			require.True(t, gaspriceproxy.ChargesL1Fee(tc.chainID))
		})
	}
}

// These chains fold the L1 cost into the L2 gas price or gas limit, so there is
// no separate fee to add.
func TestContractAddress_ChainsWithoutASeparateL1Fee(t *testing.T) {
	cases := map[string]uint64{
		"Ethereum":  walletCommon.EthereumMainnet,
		"Arbitrum":  walletCommon.ArbitrumMainnet,
		"BSC":       walletCommon.BSCMainnet,
		"Linea":     walletCommon.LineaMainnet,
		"ZKsync":    walletCommon.ZkSyncMainnet,
		"Abstract":  walletCommon.AbstractMainnet,
		"Robinhood": walletCommon.RobinhoodMainnet,
	}
	for name, chainID := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := gaspriceproxy.ContractAddress(chainID)
			require.ErrorIs(t, err, gaspriceproxy.ErrorNotAvailableOnChainID)
			require.False(t, gaspriceproxy.ChargesL1Fee(chainID))
		})
	}
}
