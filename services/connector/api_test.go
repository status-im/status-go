package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/connector/commands"
)

func TestCallRPC(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	tests := []struct {
		request     string
		expectError error
	}{
		{
			request:     "{\"method\": \"eth_chainId\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
		{
			request:     "{\"method\": \"eth_accounts\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
		{
			request:     "{\"method\": \"eth_requestAccounts\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
		{
			request:     "{\"method\": \"eth_sendTransaction\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
		{
			request:     "{\"method\": \"wallet_switchEthereumChain\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.request, func(t *testing.T) {
			_, err := state.api.CallRPC(ctx, tt.request)
			require.Error(t, err)
			require.Equal(t, tt.expectError, err)
		})
	}
}
