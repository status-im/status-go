package permit2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

// attackerAddress stands in for any address a compromised or spoofed API response could
// put in the permit2 / permit2Proxy fields.
var attackerAddress = common.HexToAddress("0xdEAD000000000000000000000000000000000bad")

// The pinned addresses spelled out, so a typo in deployments.go fails here rather than
// silently disabling the permit path (or, worse, pointing it somewhere else).
func TestDeploymentForChain(t *testing.T) {
	for _, chainID := range []uint64{
		walletCommon.EthereumMainnet,
		walletCommon.OptimismMainnet,
		walletCommon.BaseMainnet,
		walletCommon.ArbitrumMainnet,
	} {
		deployment, ok := DeploymentForChain(chainID)
		require.True(t, ok)
		require.Equal(t, common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3"), deployment.Permit2,
			"every enabled chain uses the canonical Permit2 singleton")
		require.Equal(t, common.HexToAddress("0x89c6340B1a1f4b25D36cd8B063D49045caF3f818"), deployment.Proxy)
		require.Equal(t, common.HexToAddress("0x1231DEB6f5749EF6cE6943a275A1D3E7486F4EaE"), deployment.Diamond)
	}

	// Chains whose gas accounting or Permit2 deployment doesn't fit the path.
	for _, chainID := range []uint64{
		walletCommon.AbstractMainnet,
		walletCommon.ZkSyncMainnet,
		walletCommon.EthereumSepolia,
		walletCommon.UnknownChainID,
	} {
		_, ok := DeploymentForChain(chainID)
		require.False(t, ok)
	}
}

func TestTrustedAddresses(t *testing.T) {
	deployment, ok := DeploymentForChain(walletCommon.EthereumMainnet)
	require.True(t, ok)

	tests := []struct {
		name    string
		chainID uint64
		permit2 common.Address
		proxy   common.Address
		want    bool
	}{
		{"pinned pair", walletCommon.EthereumMainnet, deployment.Permit2, deployment.Proxy, true},
		{"attacker permit2", walletCommon.EthereumMainnet, attackerAddress, deployment.Proxy, false},
		{"attacker proxy", walletCommon.EthereumMainnet, deployment.Permit2, attackerAddress, false},
		{"zero addresses", walletCommon.EthereumMainnet, common.Address{}, common.Address{}, false},
		{"chain not enabled", walletCommon.ZkSyncMainnet, deployment.Permit2, deployment.Proxy, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, TrustedAddresses(tt.chainID, tt.permit2, tt.proxy))
		})
	}
}

func TestValidateSwapTarget(t *testing.T) {
	deployment, ok := DeploymentForChain(walletCommon.EthereumMainnet)
	require.True(t, ok)

	// permit2Details() already carries the pinned pair as its Permit2/Spender.
	valid := func() *Details {
		details := permit2Details()
		details.ChainID = walletCommon.EthereumMainnet
		return details
	}

	t.Run("pinned contracts pass", func(t *testing.T) {
		require.NoError(t, ValidateSwapTarget(walletCommon.EthereumMainnet, valid(), deployment.Diamond))
	})

	t.Run("EIP-2612 details carry no Permit2", func(t *testing.T) {
		details := valid()
		details.Type = TypeEIP2612
		details.Permit2 = common.Address{}
		require.NoError(t, ValidateSwapTarget(walletCommon.EthereumMainnet, details, deployment.Diamond))
	})

	t.Run("nil details", func(t *testing.T) {
		require.ErrorIs(t, ValidateSwapTarget(walletCommon.EthereumMainnet, nil, deployment.Diamond),
			ErrMissingPermitDetails)
	})

	t.Run("chain not enabled", func(t *testing.T) {
		require.ErrorIs(t, ValidateSwapTarget(walletCommon.ZkSyncMainnet, valid(), deployment.Diamond),
			ErrChainNotEnabled)
	})

	t.Run("permit signed for another chain", func(t *testing.T) {
		details := valid()
		details.ChainID = walletCommon.OptimismMainnet
		require.ErrorIs(t, ValidateSwapTarget(walletCommon.EthereumMainnet, details, deployment.Diamond),
			ErrUntrustedPermitTarget)
	})

	t.Run("spender redirected", func(t *testing.T) {
		details := valid()
		details.Spender = attackerAddress
		require.ErrorIs(t, ValidateSwapTarget(walletCommon.EthereumMainnet, details, deployment.Diamond),
			ErrUntrustedPermitTarget)
	})

	t.Run("permit2 redirected", func(t *testing.T) {
		details := valid()
		details.Permit2 = attackerAddress
		require.ErrorIs(t, ValidateSwapTarget(walletCommon.EthereumMainnet, details, deployment.Diamond),
			ErrUntrustedPermitTarget)
	})

	t.Run("calldata target redirected", func(t *testing.T) {
		require.ErrorIs(t, ValidateSwapTarget(walletCommon.EthereumMainnet, valid(), attackerAddress),
			ErrUntrustedPermitTarget)
	})
}
