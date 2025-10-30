package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/connector/commands"
)

func TestCallRPC_UntrustedConnection(t *testing.T) {
	state := setupTests(t)

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
	state := setupTests(t)

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
	state := setupTests(t)

	ctx := WithConnectionType(context.Background(), ConnectionTypeInternal)

	request := `{
		"method": "eth_chainId",
		"params": [],
		"url": "https://example.com",
		"name": "Example DApp",
		"iconUrl": "https://example.com/icon.png",
		"clientId": "status-desktop"
	}`

	result, err := state.api.CallRPC(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestChangeAccount_UntrustedConnection(t *testing.T) {
	state := setupTests(t)

	// Test untrusted connection (HTTP)
	ctx := WithConnectionType(context.Background(), ConnectionTypeHTTP)

	args := commands.ChangeAccountArgs{
		URL:      "https://example.com",
		ClientID: "test-client",
	}

	err := state.api.ChangeAccount(ctx, args)
	require.Error(t, err)
	require.Equal(t, ErrNotAllowedForUntrustedConnection, err)
}

func TestCallRPC_MethodNotAllowed(t *testing.T) {
	state := setupTests(t)

	ctx := WithConnectionType(context.Background(), ConnectionTypeHTTP)

	// Test a method that's not in the allowed list
	request := `{
		"method": "eth_subscribe",
		"params": [],
		"url": "https://example.com",
		"name": "Example DApp",
		"iconUrl": "https://example.com/icon.png"
	}`

	result, err := state.api.CallRPC(ctx, request)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "not allowed")
}

func TestCallRPC_InvalidJSON(t *testing.T) {
	state := setupTests(t)

	ctx := WithConnectionType(context.Background(), ConnectionTypeHTTP)

	request := `invalid json`

	result, err := state.api.CallRPC(ctx, request)
	require.Error(t, err)
	require.Equal(t, "", result)
}

func TestRecallDAppPermission_Deprecated(t *testing.T) {
	state := setupTests(t)

	err := state.api.RecallDAppPermission("https://example.com")
	// Error is expected when dApp doesn't exist
	require.Error(t, err)
}

func TestGetPermittedDAppsList(t *testing.T) {
	state := setupTests(t)

	dapps, err := state.api.GetPermittedDAppsList()
	require.NoError(t, err)
	require.Empty(t, dapps)
}
