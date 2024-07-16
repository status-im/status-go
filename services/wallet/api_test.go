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

// TestAPI_HashMessageEIP191
func TestAPI_HashMessageEIP191(t *testing.T) {
	api := &API{}

	res := api.HashMessageEIP191(context.Background(), []byte("test"))
	require.Equal(t, "0x4a5c5d454721bbbb25540c3317521e71c373ae36458f960d2ad46ef088110e95", res.String())
}
