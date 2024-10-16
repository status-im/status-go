package rpcstatus

import (
	"errors"
	"testing"
	"time"
)

func TestNewRpcProviderStatus(t *testing.T) {
	tests := []struct {
		name     string
		res      RpcProviderCallStatus
		expected ProviderStatus
	}{
		{
			name: "No error, should be up",
			res: RpcProviderCallStatus{
				Name:      "Provider1",
				Timestamp: time.Now(),
				Err:       nil,
			},
			expected: ProviderStatus{
				Name:   "Provider1",
				Status: StatusUp,
			},
		},
		{
			name: "Critical RPC error, should be down",
			res: RpcProviderCallStatus{
				Name:      "Provider1",
				Timestamp: time.Now(),
				Err:       errors.New("Some critical RPC error"),
			},
			expected: ProviderStatus{
				Name:      "Provider1",
				LastError: errors.New("Some critical RPC error"),
				Status:    StatusDown,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			got := NewRpcProviderStatus(tt.res)

			// Compare expected and got
			if got.Name != tt.expected.Name {
				t.Errorf("expected name %v, got %v", tt.expected.Name, got.Name)
			}

			// Check LastSuccessAt for StatusUp
			if tt.expected.Status == StatusUp {
				if got.LastSuccessAt.IsZero() {
					t.Errorf("expected LastSuccessAt to be set, but got zero value")
				}
				if !got.LastErrorAt.IsZero() {
					t.Errorf("expected LastErrorAt to be zero, but got %v", got.LastErrorAt)
				}
			} else if tt.expected.Status == StatusDown {
				if got.LastErrorAt.IsZero() {
					t.Errorf("expected LastErrorAt to be set, but got zero value")
				}
				if !got.LastSuccessAt.IsZero() {
					t.Errorf("expected LastSuccessAt to be zero, but got %v", got.LastSuccessAt)
				}
			}

			if got.Status != tt.expected.Status {
				t.Errorf("expected status %v, got %v", tt.expected.Status, got.Status)
			}

			if got.LastError != nil && tt.expected.LastError != nil && got.LastError.Error() != tt.expected.LastError.Error() {
				t.Errorf("expected last error %v, got %v", tt.expected.LastError, got.LastError)
			}
		})
	}
}

func TestNewProviderStatus(t *testing.T) {
	tests := []struct {
		name     string
		res      ProviderCallStatus
		expected ProviderStatus
	}{
		{
			name: "No error, should be up",
			res: ProviderCallStatus{
				Name:      "Provider1",
				Timestamp: time.Now(),
				Err:       nil,
			},
			expected: ProviderStatus{
				Name:   "Provider1",
				Status: StatusUp,
			},
		},
		{
			name: "Critical provider error, should be down",
			res: ProviderCallStatus{
				Name:      "Provider1",
				Timestamp: time.Now(),
				Err:       errors.New("Some critical provider error"),
			},
			expected: ProviderStatus{
				Name:      "Provider1",
				LastError: errors.New("Some critical provider error"),
				Status:    StatusDown,
			},
		},
		{
			name: "Non-critical provider error, should be up",
			res: ProviderCallStatus{
				Name:      "Provider2",
				Timestamp: time.Now(),
				Err:       errors.New("backoff_seconds"), // Assuming this is non-critical
			},
			expected: ProviderStatus{
				Name:   "Provider2",
				Status: StatusUp,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			got := NewProviderStatus(tt.res)

			// Compare expected and got
			if got.Name != tt.expected.Name {
				t.Errorf("expected name %v, got %v", tt.expected.Name, got.Name)
			}

			// Check LastSuccessAt for StatusUp
			if tt.expected.Status == StatusUp {
				if got.LastSuccessAt.IsZero() {
					t.Errorf("expected LastSuccessAt to be set, but got zero value")
				}
				if !got.LastErrorAt.IsZero() {
					t.Errorf("expected LastErrorAt to be zero, but got %v", got.LastErrorAt)
				}
			} else if tt.expected.Status == StatusDown {
				if got.LastErrorAt.IsZero() {
					t.Errorf("expected LastErrorAt to be set, but got zero value")
				}
				if !got.LastSuccessAt.IsZero() {
					t.Errorf("expected LastSuccessAt to be zero, but got %v", got.LastSuccessAt)
				}
			}

			if got.Status != tt.expected.Status {
				t.Errorf("expected status %v, got %v", tt.expected.Status, got.Status)
			}

			if got.LastError != nil && tt.expected.LastError != nil && got.LastError.Error() != tt.expected.LastError.Error() {
				t.Errorf("expected last error %v, got %v", tt.expected.LastError, got.LastError)
			}
		})
	}
}
