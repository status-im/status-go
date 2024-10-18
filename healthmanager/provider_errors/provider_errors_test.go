package provider_errors

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"testing"
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
