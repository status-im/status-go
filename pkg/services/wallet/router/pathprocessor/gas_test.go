package pathprocessor

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
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
