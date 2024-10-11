package rpcstatus

import (
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
	Name          string     `json:"name"`
	LastSuccessAt time.Time  `json:"last_success_at"`
	LastErrorAt   time.Time  `json:"last_error_at"`
	LastError     error      `json:"last_error"`
	Status        StatusType `json:"status"`
}

// ProviderCallStatus represents the result of an arbitrary provider call.
type ProviderCallStatus struct {
	Name      string
	Timestamp time.Time
	Err       error
}

// RpcProviderCallStatus represents the result of an RPC provider call.
type RpcProviderCallStatus struct {
	Name      string
	Timestamp time.Time
	Err       error
}

// NewRpcProviderStatus processes RpcProviderCallStatus and returns a new ProviderStatus.
func NewRpcProviderStatus(res RpcProviderCallStatus) ProviderStatus {
	status := ProviderStatus{
		Name: res.Name,
	}

	// Determine if the error is critical
	if res.Err == nil || provider_errors.IsNonCriticalRpcError(res.Err) || provider_errors.IsNonCriticalProviderError(res.Err) {
		status.LastSuccessAt = res.Timestamp
		status.Status = StatusUp
	} else {
		status.LastErrorAt = res.Timestamp
		status.LastError = res.Err
		status.Status = StatusDown
	}

	return status
}

// NewProviderStatus processes ProviderCallStatus and returns a new ProviderStatus.
func NewProviderStatus(res ProviderCallStatus) ProviderStatus {
	status := ProviderStatus{
		Name: res.Name,
	}

	// Determine if the error is critical
	if res.Err == nil || provider_errors.IsNonCriticalProviderError(res.Err) {
		status.LastSuccessAt = res.Timestamp
		status.Status = StatusUp
	} else {
		status.LastErrorAt = res.Timestamp
		status.LastError = res.Err
		status.Status = StatusDown
	}

	return status
}
