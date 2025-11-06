package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectionType(t *testing.T) {
	ctx := WithConnectionType(context.Background(), ConnectionTypeHTTP)
	require.True(t, IsUntrustedConnection(ctx))
	require.Equal(t, ConnectionTypeHTTP, GetConnectionType(ctx))

	ctx = WithConnectionType(context.Background(), ConnectionTypeInternal)
	require.False(t, IsUntrustedConnection(ctx))
	require.Equal(t, ConnectionTypeInternal, GetConnectionType(ctx))
}

func TestConnectionTypeDefault(t *testing.T) {
	// Context without connection type should default to untrusted
	ctx := context.Background()

	if IsUntrustedConnection(ctx) {
		t.Error("Default connection type should be untrusted (HTTP)")
	}

	if got := GetConnectionType(ctx); got != ConnectionTypeInternal {
		t.Errorf("Default GetConnectionType() = %v, want %v", got, ConnectionTypeInternal)
	}
}
