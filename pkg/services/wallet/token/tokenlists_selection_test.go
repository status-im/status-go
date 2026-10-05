package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenManagerStartStopIdempotent(t *testing.T) {
	m, _, cleanup := setupTestTokenManager(t)
	defer cleanup()
	require.NoError(t, m.Start(context.Background()))
	first := m.stopCh
	require.NoError(t, m.Start(context.Background()))
	require.Equal(t, first, m.stopCh, "second Start must not leak watchers")
	m.Stop()
	m.Stop()
}
