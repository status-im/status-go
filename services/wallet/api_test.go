package wallet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/walletconnect"
)

// TestAPI_GetWalletConnectActiveSessions tames coverage
func TestAPI_GetWalletConnectActiveSessions(t *testing.T) {
	db, close := walletconnect.SetupTestDB(t)
	defer close()
	api := &API{
		s: &Service{db: db},
	}

	sessions, err := api.GetWalletConnectActiveSessions(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, 0, len(sessions))
}
