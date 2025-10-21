package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/connector/commands"
)

func TestCallRPC_UntrustedConnection(t *testing.T) {
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
		{
			request: `{
				"method": "eth_chainId",
				"params": [],
				"url": "https://example.com",
				"name": "Example DApp",
				"iconUrl": "https://example.com/icon.png",
				"clientId": "wallet-connect"
			}`,
			expectError: ErrCannotOverrideClientIDForHttpConnection,
		},
	}

	ctx := WithConnectionType(context.Background(), ConnectionTypeHTTP)
	for _, tt := range tests {
		t.Run(tt.request, func(t *testing.T) {
			_, err := state.api.CallRPC(ctx, tt.request)
			require.Error(t, err)
			require.Equal(t, tt.expectError, err)
		})
	}
}

func TestCallRPC_TrustedConnectionRequiresClientID(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	// Trusted connection (Internal) without ClientID should fail
	ctx := WithConnectionType(context.Background(), ConnectionTypeInternal)

	request := `{
		"method": "eth_chainId",
		"params": [],
		"url": "https://example.com",
		"name": "Example DApp",
		"iconUrl": "https://example.com/icon.png"
	}`

	_, err := state.api.CallRPC(ctx, request)
	require.Error(t, err)
	require.Equal(t, ErrEmptyClientIDFromTrustedConnection, err)
}

func TestCallRPC_TrustedConnectionWithClientID(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	ctx := WithConnectionType(context.Background(), ConnectionTypeInternal)

	request := `{
		"method": "eth_chainId",
		"params": [],
		"url": "https://example.com",
		"name": "Example DApp",
		"iconUrl": "https://example.com/icon.png",
		"clientId": "status-desktop"
	}`

	_, err := state.api.CallRPC(ctx, request)
	require.Error(t, err)
	require.NotEqual(t, ErrEmptyClientIDFromTrustedConnection, err)
}
