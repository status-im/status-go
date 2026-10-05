package onramp_test

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/pkg/services/wallet/onramp"

	"github.com/stretchr/testify/require"
)

func TestMercuryoProvider_GetURL_validation(t *testing.T) {
	provider := onramp.NewMercuryoProvider(nil, onramp.MercuryoParams{})
	dest := common.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := uint64(1)
	unsupportedChainID := uint64(999999)
	symbol := "ETH"

	tests := []struct {
		name        string
		params      onramp.Parameters
		errContains string
	}{
		{
			name:        "missing destination address",
			params:      onramp.Parameters{ChainID: &chainID, Symbol: &symbol},
			errContains: "destination address is required",
		},
		{
			name:        "missing chain ID",
			params:      onramp.Parameters{DestAddress: &dest, Symbol: &symbol},
			errContains: "chainID is required",
		},
		{
			name:        "missing symbol",
			params:      onramp.Parameters{DestAddress: &dest, ChainID: &chainID},
			errContains: "symbol is required",
		},
		{
			name:        "unsupported chain ID",
			params:      onramp.Parameters{DestAddress: &dest, ChainID: &unsupportedChainID, Symbol: &symbol},
			errContains: "unsupported chainID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.GetURL(context.Background(), tt.params)
			require.ErrorContains(t, err, tt.errContains)
		})
	}
}
