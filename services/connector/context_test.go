package connector

import (
	"context"
	"testing"
)

func TestConnectionType(t *testing.T) {
	tests := []struct {
		name          string
		connType      ConnectionType
		wantUntrusted bool
		wantTrusted   bool
	}{
		{
			name:          "HTTP connection is untrusted",
			connType:      ConnectionTypeHTTP,
			wantUntrusted: true,
			wantTrusted:   false,
		},
		{
			name:          "Internal connection is trusted",
			connType:      ConnectionTypeInternal,
			wantUntrusted: false,
			wantTrusted:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithConnectionType(context.Background(), tt.connType)

			if got := IsUntrustedConnection(ctx); got != tt.wantUntrusted {
				t.Errorf("IsUntrustedConnection() = %v, want %v", got, tt.wantUntrusted)
			}

			if got := IsTrustedConnection(ctx); got != tt.wantTrusted {
				t.Errorf("IsTrustedConnection() = %v, want %v", got, tt.wantTrusted)
			}

			if got := GetConnectionType(ctx); got != tt.connType {
				t.Errorf("GetConnectionType() = %v, want %v", got, tt.connType)
			}
		})
	}
}

func TestConnectionTypeDefault(t *testing.T) {
	// Context without connection type should default to Internal (trusted)
	ctx := context.Background()

	if !IsTrustedConnection(ctx) {
		t.Error("Default connection type should be trusted (Internal)")
	}

	if got := GetConnectionType(ctx); got != ConnectionTypeInternal {
		t.Errorf("Default GetConnectionType() = %v, want %v", got, ConnectionTypeInternal)
	}
}
