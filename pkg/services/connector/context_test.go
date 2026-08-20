package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectionType(t *testing.T) {
	ctx := WithConnectionType(context.Background(), ConnectionTypeUntrusted)
	require.True(t, IsUntrustedConnection(ctx))
	require.Equal(t, ConnectionTypeUntrusted, GetConnectionType(ctx))

	ctx = WithConnectionType(context.Background(), ConnectionTypeTrusted)
	require.False(t, IsUntrustedConnection(ctx))
	require.Equal(t, ConnectionTypeTrusted, GetConnectionType(ctx))
}

func TestConnectionTypeDefault(t *testing.T) {
	// Default should be untrusted for safety
	ctx := context.Background()

	require.True(t, IsUntrustedConnection(ctx))
	require.Equal(t, ConnectionTypeUntrusted, GetConnectionType(ctx))
}
