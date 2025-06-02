package rpcstatus

import (
	"encoding/json"
	"time"

	"github.com/status-im/status-go/healthmanager/provider_errors"
)

// StatusType represents the possible status values for a provider.
type StatusType string

const (
	StatusUnknown StatusType = "unknown"
	StatusUp      StatusType = "up"
	StatusDown    StatusType = "down"
)

// ProviderStatus holds the status information for a single provider.
type ProviderStatus struct {
	Name              string        `json:"name"`
	LastSuccessAt     time.Time     `json:"last_success_at"`
	LastErrorAt       time.Time     `json:"last_error_at"`
	LastError         error         `json:"-"` // ignore this field during standard marshaling
	Status            StatusType    `json:"status"`
	TotalDuration     time.Duration `json:"-"` // ignore this field during standard marshaling
	TotalRequests     int64         `json:"total_requests"`
	TotalTimeoutCount int64         `json:"total_timeout_count"`
	TotalErrorCount   int64         `json:"total_error_count"`
}

// MarshalJSON implements custom JSON marshaling for ProviderStatus
func (ps ProviderStatus) MarshalJSON() ([]byte, error) {
	type Alias ProviderStatus // prevent recursive MarshalJSON calls

	// Create a new struct for JSON marshaling
	return json.Marshal(&struct {
		Alias
		LastError       string `json:"last_error,omitempty"`
		TotalDurationMs int64  `json:"total_duration_ms"` // Include duration as milliseconds
	}{
		Alias: Alias(ps),
		LastError: func() string {
			if ps.LastError != nil {
				return ps.LastError.Error()
			}
			return ""
		}(),
		TotalDurationMs: ps.TotalDuration.Milliseconds(), // Convert duration to milliseconds
	})
}

// RpcProviderCallStatus represents the result of an RPC provider call.
type RpcProviderCallStatus struct {
	Name      string
	Timestamp time.Time
	Method    string
	Err       error
	StartTime time.Time
}

// NewRpcProviderStatus processes RpcProviderCallStatus and returns a new ProviderStatus.
func NewRpcProviderStatus(res RpcProviderCallStatus) ProviderStatus {
	status := ProviderStatus{
		Name:          res.Name,
		TotalRequests: 1,
		TotalDuration: res.Timestamp.Sub(res.StartTime),
	}

	if res.Err == nil || provider_errors.IsNonCriticalRpcError(res.Err) || provider_errors.IsNonCriticalProviderError(res.Err) {
		status.LastSuccessAt = res.Timestamp
		status.Status = StatusUp
	} else {
		status.LastErrorAt = res.Timestamp
		status.LastError = res.Err
		status.Status = StatusDown
		status.TotalErrorCount = 1
		if provider_errors.IsTimeoutErr(res.Err) {
			status.TotalTimeoutCount = 1
		}
	}

	return status
}
