package rpcstatus

import (
	"errors"
	"testing"
	"time"

	"github.com/status-im/status-go/rpc/chain/rpclimiter"
)

func TestNewRpcProviderStatus(t *testing.T) {
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
				Timestamp: time.Now(),
				Err:       nil,
			},
			expectSuccess: true,
		},
		{
			name: "Critical RPC error, should be down",
			res: RpcProviderCallStatus{
				Name:      "Provider1",
				Timestamp: time.Now(),
				Err:       errors.New("Some critical RPC error"),
			},
			expectSuccess: false,
			expectedError: errors.New("Some critical RPC error"),
		},
		{
			name: "Non-critical RPC error, should be up",
			res: RpcProviderCallStatus{
				Name:      "Provider2",
				Timestamp: time.Now(),
				Err:       rpclimiter.ErrRequestsOverLimit,
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectSuccess {
				if tt.res.Err != nil && tt.res.Err != rpclimiter.ErrRequestsOverLimit {
					t.Errorf("expected success but got error: %v", tt.res.Err)
				}
			} else {
				if tt.res.Err == nil {
					t.Error("expected error but got success")
				}
				if tt.expectedError != nil && tt.res.Err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedError, tt.res.Err)
				}
			}
		})
	}
}
