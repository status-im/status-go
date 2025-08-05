package ensresolver

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"

	"github.com/stretchr/testify/require"
)

func TestNameWrapperContractAddress(t *testing.T) {
	tests := []struct {
		name        string
		chainID     uint64
		expected    string
		expectError bool
	}{
		{
			name:        "Ethereum Mainnet",
			chainID:     walletCommon.EthereumMainnet,
			expected:    "0xD4416b13d2b3a9aBae7AcD5D6C2BbDBE25686401",
			expectError: false,
		},
		{
			name:        "Ethereum Sepolia",
			chainID:     walletCommon.EthereumSepolia,
			expected:    "0x0635513f179D50A207757E05759CbD106d7dFcE8",
			expectError: false,
		},
		{
			name:        "Unsupported Chain",
			chainID:     12345,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := NameWrapperContractAddress(tt.chainID)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, common.HexToAddress(tt.expected), addr)
			}
		})
	}
}
