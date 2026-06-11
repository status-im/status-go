package provider_errors

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/status-im/status-go/services/wallet/puzzleauth"
)

// TestIsConnectionError tests the IsConnectionError function.
func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantResult bool
	}{
		{
			name:       "nil error",
			err:        nil,
			wantResult: false,
		},
		{
			name:       "net.DNSError with timeout",
			err:        &net.DNSError{IsTimeout: true},
			wantResult: true,
		},
		{
			name:       "DNS error without timeout",
			err:        &net.DNSError{},
			wantResult: true,
		},
		{
			name:       "net.OpError",
			err:        &net.OpError{},
			wantResult: true,
		},
		{
			name:       "tls.RecordHeaderError",
			err:        &tls.RecordHeaderError{},
			wantResult: true,
		},
		{
			name:       "tls.CertificateVerificationError",
			err:        &tls.CertificateVerificationError{},
			wantResult: true,
		},
		{
			name:       "tls.AlertError",
			err:        tls.AlertError(0),
			wantResult: true,
		},
		{
			name:       "context.DeadlineExceeded",
			err:        context.DeadlineExceeded,
			wantResult: true,
		},
		{
			name:       "http.ErrServerClosed",
			err:        http.ErrServerClosed,
			wantResult: true,
		},
		{
			name:       "i/o timeout error message",
			err:        errors.New("i/o timeout"),
			wantResult: true,
		},
		{
			name:       "connection refused error message",
			err:        errors.New("connection refused"),
			wantResult: true,
		},
		{
			name:       "network is unreachable error message",
			err:        errors.New("network is unreachable"),
			wantResult: true,
		},
		{
			name:       "no such host error message",
			err:        errors.New("no such host"),
			wantResult: true,
		},
		{
			name:       "tls handshake timeout error message",
			err:        errors.New("tls handshake timeout"),
			wantResult: true,
		},
		{
			name:       "rps limit error 1",
			err:        errors.New("backoff_seconds"),
			wantResult: false,
		},
		{
			name:       "rps limit error 2",
			err:        errors.New("has exceeded its throughput limit"),
			wantResult: false,
		},
		{
			name:       "rps limit error 3",
			err:        errors.New("request rate exceeded"),
			wantResult: false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture the variable
		t.Run(tt.name, func(t *testing.T) {
			got := IsConnectionError(tt.err)
			if got != tt.wantResult {
				t.Errorf("IsConnectionError(%v) = %v; want %v", tt.err, got, tt.wantResult)
			}
		})
	}
}

func TestDetermineProviderErrorType_AuthRotating(t *testing.T) {
	if got := determineProviderErrorType(puzzleauth.ErrAuthRotating); got != ProviderErrorTypeAuthRotating {
		t.Errorf("determineProviderErrorType(ErrAuthRotating) = %q, want %q", got, ProviderErrorTypeAuthRotating)
	}
	wrapped := fmt.Errorf("wrapped: %w", puzzleauth.ErrAuthRotating)
	if got := determineProviderErrorType(wrapped); got != ProviderErrorTypeAuthRotating {
		t.Errorf("determineProviderErrorType(wrapped ErrAuthRotating) = %q, want %q", got, ProviderErrorTypeAuthRotating)
	}
}

func TestIsNonCriticalProviderError_PuzzleAuthRotating(t *testing.T) {
	if !IsNonCriticalProviderError(fmt.Errorf("wrapped: %w", puzzleauth.ErrAuthRotating)) {
		t.Error("IsNonCriticalProviderError should be true for ErrAuthRotating (wrapped)")
	}
	if !IsNonCriticalProviderError(puzzleauth.ErrAuthRotating) {
		t.Error("IsNonCriticalProviderError should be true for bare ErrAuthRotating")
	}
	other := errors.New("not auth rotating")
	if IsNonCriticalProviderError(other) {
		t.Error("IsNonCriticalProviderError should be false for unrelated errors")
	}
}

func TestDetermineProviderErrorType_DataUnavailable(t *testing.T) {
	if got := determineProviderErrorType(ErrDataUnavailable); got != ProviderErrorTypeDataUnavailable {
		t.Errorf("determineProviderErrorType(ErrDataUnavailable) = %q, want %q", got, ProviderErrorTypeDataUnavailable)
	}

	wrapped := fmt.Errorf("wrapped: %w", ErrDataUnavailable)
	if got := determineProviderErrorType(wrapped); got != ProviderErrorTypeDataUnavailable {
		t.Errorf("determineProviderErrorType(wrapped ErrDataUnavailable) = %q, want %q", got, ProviderErrorTypeDataUnavailable)
	}
}

func TestIsNonCriticalProviderError_DataUnavailable(t *testing.T) {
	requireErr := fmt.Errorf("wrapped: %w", ErrDataUnavailable)
	if !IsNonCriticalProviderError(requireErr) {
		t.Error("IsNonCriticalProviderError should be true for wrapped ErrDataUnavailable")
	}
	if !IsNonCriticalProviderError(ErrDataUnavailable) {
		t.Error("IsNonCriticalProviderError should be true for bare ErrDataUnavailable")
	}
}

func TestIsIgnorableForConnectivity(t *testing.T) {
	if !IsIgnorableForConnectivity(fmt.Errorf("wrapped: %w", ErrDataUnavailable)) {
		t.Error("IsIgnorableForConnectivity should be true for wrapped ErrDataUnavailable")
	}
	if IsIgnorableForConnectivity(errors.New("fatal provider error")) {
		t.Error("IsIgnorableForConnectivity should be false for unrelated errors")
	}
}

func TestIsIgnorableForConnectivity_JoinedErrors(t *testing.T) {
	connectionErr := &net.OpError{Op: "dial", Err: errors.New("connection refused")}

	// Joined errors are ignorable only when every component is ignorable.
	mixed := fmt.Errorf("%w, provider2.error: %w", ErrDataUnavailable, connectionErr)
	if IsIgnorableForConnectivity(mixed) {
		t.Error("IsIgnorableForConnectivity should be false when a joined error contains a connectivity failure")
	}

	allIgnorable := fmt.Errorf("%w, provider2.error: %w", ErrDataUnavailable, context.Canceled)
	if !IsIgnorableForConnectivity(allIgnorable) {
		t.Error("IsIgnorableForConnectivity should be true when all joined errors are ignorable")
	}

	// Same shape as circuitbreaker.accumulateCommandError: leaves are single-wrapped.
	nested := fmt.Errorf("%w, provider2.error: %w",
		fmt.Errorf("provider1.error: %w", ErrDataUnavailable),
		fmt.Errorf("call failed: %w", connectionErr))
	if IsIgnorableForConnectivity(nested) {
		t.Error("IsIgnorableForConnectivity should be false for nested joined errors with a connectivity failure")
	}

	if IsIgnorableForConnectivity(errors.Join()) {
		t.Error("IsIgnorableForConnectivity should be false for nil joined error")
	}
}
