package routes

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/require"
)

func TestGasEstimateFreshness(t *testing.T) {
	now := time.Unix(1_000_000, 0)

	testCases := []struct {
		name          string
		estimatedAgo  time.Duration // 0 = never estimated
		expectedDue   bool
		expectedFresh bool
	}{
		{"never estimated", 0, true, false},
		{"just estimated", time.Second, false, true},
		{"right before the re-estimate interval", GasReestimateInterval - time.Second, false, true},
		{"at the re-estimate interval", GasReestimateInterval, true, true},
		{"at the max age", GasEstimateMaxAge, true, true},
		{"past the max age", GasEstimateMaxAge + time.Second, true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var estimatedAt time.Time
			if tc.estimatedAgo > 0 {
				estimatedAt = now.Add(-tc.estimatedAgo)
			}

			// the main and the approval tx are tracked independently
			mainPath := &Path{TxGasEstimatedAt: estimatedAt}
			require.Equal(t, tc.expectedDue, mainPath.GasEstimateDue(false, now))
			require.Equal(t, tc.expectedFresh, mainPath.GasEstimateFresh(false, now))
			require.True(t, mainPath.GasEstimateDue(true, now))
			require.False(t, mainPath.GasEstimateFresh(true, now))

			approvalPath := &Path{ApprovalGasEstimatedAt: estimatedAt}
			require.Equal(t, tc.expectedDue, approvalPath.GasEstimateDue(true, now))
			require.Equal(t, tc.expectedFresh, approvalPath.GasEstimateFresh(true, now))
			require.True(t, approvalPath.GasEstimateDue(false, now))
			require.False(t, approvalPath.GasEstimateFresh(false, now))
		})
	}
}

func TestGasEstimateFreshForTheSignedValue(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	value := func(v int64) *hexutil.Big { return (*hexutil.Big)(big.NewInt(v)) }

	// a full-balance send's value changes when fees are rebuilt out of it
	plain := &Path{TxGasEstimatedAt: now, AmountIn: value(974_800), TxGasEstimatedValue: value(974_800)}
	require.True(t, plain.GasEstimateFresh(false, now))
	plain.AmountIn = value(924_400)
	require.False(t, plain.GasEstimateFresh(false, now))
	require.False(t, plain.GasEstimateDue(false, now), "re-estimated at build, not every block")

	// packed data encodes its own amount
	packed := &Path{TxGasEstimatedAt: now, AmountIn: value(924_400), TxGasEstimatedValue: value(974_800), TxPackedData: []byte{0x01}}
	require.True(t, packed.GasEstimateFresh(false, now))

	// the approval doesn't carry the value
	approval := &Path{ApprovalGasEstimatedAt: now, AmountIn: value(924_400)}
	require.True(t, approval.GasEstimateFresh(true, now))
}
