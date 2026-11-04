package routes

import (
	"testing"
	"time"

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
