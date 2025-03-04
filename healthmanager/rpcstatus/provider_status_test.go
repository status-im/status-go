package rpcstatus

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/healthmanager/provider_errors"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
)

func TestNewRpcProviderStatus(t *testing.T) {
	now := time.Now()
	duration := 100 * time.Millisecond

	tests := []struct {
		name          string
		res           RpcProviderCallStatus
		expectSuccess bool
		expectedError error
	}{
		{
			name: "No error, should be up",
			res: RpcProviderCallStatus{
				Name:      "Provider1",
				Timestamp: now,
				Err:       nil,
				StartTime: now.Add(-duration),
			},
			expectSuccess: true,
		},
		{
			name: "Critical RPC error, should be down",
			res: RpcProviderCallStatus{
				Name:      "Provider1",
				Timestamp: now,
				Err:       errors.New("Some critical RPC error"),
				StartTime: now.Add(-duration),
			},
			expectSuccess: false,
			expectedError: errors.New("Some critical RPC error"),
		},
		{
			name: "Non-critical RPC error, should be up",
			res: RpcProviderCallStatus{
				Name:      "Provider2",
				Timestamp: now,
				Err:       rpclimiter.ErrRequestsOverLimit,
				StartTime: now.Add(-duration),
			},
			expectSuccess: true,
		},
		{
			name: "Timeout error, should be down but tracked",
			res: RpcProviderCallStatus{
				Name:      "Provider3",
				Timestamp: now,
				Err:       errors.New("context deadline exceeded"),
				StartTime: now.Add(-duration),
			},
			expectSuccess: false,
			expectedError: errors.New("context deadline exceeded"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			status := NewRpcProviderStatus(tt.res)

			if tt.expectSuccess {
				if tt.res.Err != nil && tt.res.Err != rpclimiter.ErrRequestsOverLimit {
					t.Errorf("expected success but got error: %v", tt.res.Err)
				}
				assert.Equal(t, StatusUp, status.Status, "Status should be up")
			} else {
				if tt.res.Err == nil {
					t.Error("expected error but got success")
				}
				if tt.expectedError != nil && tt.res.Err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedError, tt.res.Err)
				}
				assert.Equal(t, StatusDown, status.Status, "Status should be down")
			}

			// Verify the new fields are correctly transferred
			expectedDuration := tt.res.Timestamp.Sub(tt.res.StartTime)
			assert.Equal(t, expectedDuration, status.TotalDuration, "Duration should be calculated as Timestamp - StartTime")
			assert.Equal(t, int64(1), status.TotalRequests, "TotalRequests should be initialized to 1")

			if provider_errors.IsTimeoutErr(tt.res.Err) {
				assert.Equal(t, int64(1), status.TotalTimeoutCount, "TotalTimeoutCount should be 1 for timeout errors")
			} else {
				assert.Equal(t, int64(0), status.TotalTimeoutCount, "TotalTimeoutCount should be 0 for non-timeout errors")
			}
		})
	}
}
