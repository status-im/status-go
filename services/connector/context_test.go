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

	ctx = WithConnectionType(context.Background(), ConnectionTypeWS)
	require.True(t, IsUntrustedConnection(ctx))
	require.Equal(t, ConnectionTypeWS, GetConnectionType(ctx))

	ctx = WithConnectionType(context.Background(), ConnectionTypeInternal)
	require.False(t, IsUntrustedConnection(ctx))
	require.Equal(t, ConnectionTypeInternal, GetConnectionType(ctx))
}

func TestConnectionTypeDefault(t *testing.T) {
	// This is the case for in-process RPC calls from status-desktop
	ctx := context.Background()

	if IsUntrustedConnection(ctx) {
		t.Error("Default connection type should be trusted (internal) for in-process calls")
	}

	if got := GetConnectionType(ctx); got != ConnectionTypeInternal {
		t.Errorf("Default GetConnectionType() = %v, want %v", got, ConnectionTypeInternal)
	}
}
