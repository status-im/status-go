package provider_errors

import (
	"errors"
	"testing"
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
			if got != tt.wantResult {
				t.Errorf("IsRpsLimitError(%v) = %v; want %v", tt.err, got, tt.wantResult)
			}
		})
	}
}
