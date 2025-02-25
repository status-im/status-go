package rpcstatus

import (
	"encoding/json"
	"fmt"
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
	ChainID       uint64     `json:"chain_id"`
	LastSuccessAt time.Time  `json:"last_success_at"`
	LastErrorAt   time.Time  `json:"last_error_at"`
	LastError     error      `json:"-"` // ignore this field during standard marshaling
	Status        StatusType `json:"status"`
}

// MarshalJSON implements custom JSON marshaling for ProviderStatus
func (ps ProviderStatus) MarshalJSON() ([]byte, error) {
	type Alias ProviderStatus // prevent recursive MarshalJSON calls

	// Create a new struct for JSON marshaling
	return json.Marshal(&struct {
		Alias
		LastError string `json:"last_error,omitempty"`
	}{
		Alias: Alias(ps),
		LastError: func() string {
			if ps.LastError != nil {
				return ps.LastError.Error()
			}
			return ""
		}(),
	})
}

// RpcProviderCallStatus represents the result of an RPC provider call.
type RpcProviderCallStatus struct {
	Name      string
	ChainID   uint64
	Timestamp time.Time
	Method    string
	Err       error
}

// NewRpcProviderStatus processes RpcProviderCallStatus and returns a new ProviderStatus.
func NewRpcProviderStatus(res RpcProviderCallStatus) ProviderStatus {
	status := ProviderStatus{
		Name:    res.Name,
		ChainID: res.ChainID,
	}

	// Determine if the error is critical
	if res.Err == nil || provider_errors.IsNonCriticalRpcError(res.Err) || provider_errors.IsNonCriticalProviderError(res.Err) {
		status.LastSuccessAt = res.Timestamp
		status.Status = StatusUp
	} else {
		status.LastErrorAt = res.Timestamp
		status.LastError = fmt.Errorf("method: %s, error: %w", res.Method, res.Err)
		status.Status = StatusDown
	}

	return status
}
