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

func TestShadowRequiresNim(t *testing.T) {
	_, err := NewTokenManager(nil, nil, nil, nil, nil, nil, nil, nil, nil, 0, 0, ManagerOptions{Shadow: true})
	require.ErrorContains(t, err, "requires TokenListsUseNim")
}
