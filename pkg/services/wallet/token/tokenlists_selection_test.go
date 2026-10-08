package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
)

func TestTokenManagerStartStopIdempotent(t *testing.T) {
	m, _, cleanup := setupTestTokenManager(t)
	defer cleanup()
	require.IsType(t, &tklmanager.Manager{}, m.tokensManager, "the native catalogue must be the only backend")
	require.NoError(t, m.Start(context.Background()))
	first := m.stopCh
	require.NoError(t, m.Start(context.Background()))
	require.Equal(t, first, m.stopCh, "second Start must not leak watchers")
	m.Stop()
	m.Stop()
}
