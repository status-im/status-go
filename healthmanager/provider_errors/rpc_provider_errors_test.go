package provider_errors

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsRpsLimitError tests the IsRpsLimitError function.
func TestIsRpsLimitError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantResult bool
	}{
		{
			name:       "Error contains 'backoff_seconds'",
			err:        errors.New("Error: backoff_seconds: 30"),
			wantResult: true,
		},
		{
			name:       "Error contains 'has exceeded its throughput limit'",
			err:        errors.New("Your application has exceeded its throughput limit."),
			wantResult: true,
		},
		{
			name:       "Error contains 'request rate exceeded'",
			err:        errors.New("Request rate exceeded. Please try again later."),
			wantResult: true,
		},
		{
			name:       "Error does not contain any matching phrases",
			err:        errors.New("Some other error occurred."),
			wantResult: false,
		},
		{
			name:       "Error is nil",
			err:        nil,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture the variable
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitError(tt.err)
			require.Equal(t, tt.wantResult, got)
		})
	}
}

// mockRPCError implements the rpc.Error interface for testing
type mockRPCError struct {
	code    int
	message string
}

func (e *mockRPCError) Error() string {
	return e.message
}

func (e *mockRPCError) ErrorCode() int {
	return e.code
}

// TestNilRPCErrorHandling tests handling of nil RPC errors in various wrapping scenarios
func TestNilRPCErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType RpcProviderErrorType
	}{
		{
			name: "direct nil RPC error in url.Error",
			err: &url.Error{
				Op:  "Post",
				URL: "http://localhost:8545",
				Err: (*mockRPCError)(nil),
			},
			wantType: RpcErrorTypeRPCOther,
		},
		{
			name: "wrapped nil RPC error in url.Error",
			err: fmt.Errorf("outer error: %w",
				&url.Error{
					Op:  "Post",
					URL: "http://localhost:8545",
					Err: (*mockRPCError)(nil),
				}),
			wantType: RpcErrorTypeRPCOther,
		},
		{
			name: "double wrapped nil RPC error",
			err: fmt.Errorf("outer error: %w",
				fmt.Errorf("inner error: %w",
					&url.Error{
						Op:  "Post",
						URL: "http://localhost:8545",
						Err: (*mockRPCError)(nil),
					})),
			wantType: RpcErrorTypeRPCOther,
		},
		{
			name: "direct nil error in url.Error",
			err: &url.Error{
				Op:  "Post",
				URL: "http://localhost:8545",
				Err: nil,
			},
			wantType: RpcErrorTypeRPCOther,
		},
		{
			name:     "direct nil error",
			err:      nil,
			wantType: RpcErrorTypeNone,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Test for panics in all error handling functions
			testFuncs := []struct {
				name string
				fn   func(error)
			}{
				{"IsRPCError", func(err error) { _, _ = IsRPCError(err) }},
				{"IsMethodNotFoundError", func(err error) { _ = IsMethodNotFoundError(err) }},
				{"determineRpcErrorType", func(err error) { _ = determineRpcErrorType(err) }},
				{"IsNonCriticalRpcError", func(err error) { _ = IsNonCriticalRpcError(err) }},
			}

			// Verify no panics
			for _, tf := range testFuncs {
				tf := tf
				require.NotPanics(t, func() { tf.fn(tt.err) })
			}

			// Test error type determination
			errType := determineRpcErrorType(tt.err)
			require.Equal(t, tt.wantType, errType)

			if tt.err != nil {
				// Check IsMethodNotFoundError behavior
				require.False(t, IsMethodNotFoundError(tt.err),
					"IsMethodNotFoundError() should return false for nil RPC error")

				// Check IsNonCriticalRpcError behavior
				if tt.wantType == RpcErrorTypeRPCOther {
					require.False(t, IsNonCriticalRpcError(tt.err),
						"IsNonCriticalRpcError() should return false for RpcErrorTypeRPCOther")
				}
			}
		})
	}
}
