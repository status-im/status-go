package requests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/centralizedmetrics/common"
)

func TestValidateAddCentralizedMetrics(t *testing.T) {
	tests := []struct {
		name          string
		request       AddCentralizedMetric
		expectedError error
	}{
		{
			name: "valid metric",
			request: AddCentralizedMetric{
				Metric: &common.Metric{EventName: "event-name", Platform: "android", AppVersion: "version"},
			},
			expectedError: nil,
		},
		{
			name:          "empty metric",
			expectedError: ErrAddCentralizedMetricInvalidMetric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			require.Equal(t, tt.expectedError, err)
		})
	}
}
