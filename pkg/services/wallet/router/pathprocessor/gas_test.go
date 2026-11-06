package pathprocessor

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	pathprocessorcommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

func TestGasForQuotedTx(t *testing.T) {
	gas := func(v uint64) *hexutil.Uint64 { g := hexutil.Uint64(v); return &g }

	testCases := []struct {
		name      string
		routerGas *hexutil.Uint64
		quotedGas uint64
		expected  uint64
	}{
		{"quote below router estimate", gas(161845), 60000, 161845},
		{"quote above router estimate", gas(150000), 300000, 300000},
		{"equal", gas(200000), 200000, 200000},
		{"no router estimate", nil, 250000, 250000},
		{"zero router estimate", gas(0), 250000, 250000},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, gasForQuotedTx(tc.routerGas, tc.quotedGas))
		})
	}
}

func TestBufferGasEstimate(t *testing.T) {
	require.Equal(t, uint64(22050), BufferGasEstimate(21000, pathprocessorcommon.IncreaseEstimatedGasFactor))
	// truncated, so the raw estimate is ceil(buffered / factor)
	require.Equal(t, uint64(217760), BufferGasEstimate(207391, pathprocessorcommon.IncreaseEstimatedGasFactor))
	require.Equal(t, uint64(240001), BufferGasEstimate(200001, pathprocessorcommon.IncreaseEstimatedGasFactorForBridge))
}

func TestGasBufferFactorDefaultsToIncreaseEstimatedGasFactor(t *testing.T) {
	require.Equal(t, pathprocessorcommon.IncreaseEstimatedGasFactor, GasBufferFactor(&TransferProcessor{}, ProcessorInputParams{}))
	require.Equal(t, pathprocessorcommon.IncreaseEstimatedGasFactorForBridge, GasBufferFactor(&HopBridgeProcessor{}, ProcessorInputParams{}))
}
