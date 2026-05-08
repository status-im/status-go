package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/services/connector/walletconnect"
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
			expectError: ErrCannotOverrideClientIDForUntrustedConnection,
		},
	}

	ctx := WithConnectionType(context.Background(), ConnectionTypeUntrusted)
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
	ctx := WithConnectionType(context.Background(), ConnectionTypeTrusted)

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

	ctx := WithConnectionType(context.Background(), ConnectionTypeTrusted)

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

func TestCallRPC_WalletGetCapabilities(t *testing.T) {
	state := setupTests(t)
	ctx := WithConnectionType(context.Background(), ConnectionTypeTrusted)

	res, err := state.api.CallRPC(ctx, `{
		"method": "wallet_getCapabilities",
		"params": [],
		"url": "https://example.com",
		"name": "Example DApp",
		"iconUrl": "https://example.com/icon.png",
		"clientId": "status-desktop"
	}`)
	require.NoError(t, err)
	m, ok := res.(map[string]any)
	require.True(t, ok)
	require.Empty(t, m)

	_, err = state.api.CallRPC(ctx, `{
		"method": "wallet_sendCalls",
		"params": [],
		"url": "https://example.com",
		"name": "Example DApp",
		"iconUrl": "https://example.com/icon.png",
		"clientId": "status-desktop"
	}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestChangeAccount_UntrustedConnection(t *testing.T) {
	state := setupTests(t)

	// Test untrusted connection (HTTP)
	ctx := WithConnectionType(context.Background(), ConnectionTypeUntrusted)

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

	ctx := WithConnectionType(context.Background(), ConnectionTypeUntrusted)

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

	ctx := WithConnectionType(context.Background(), ConnectionTypeUntrusted)

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

func TestDeleteEphemeralDApps(t *testing.T) {
	state := setupTests(t)

	normal := persistence.DApp{
		URL:           "https://normal-dapp.com",
		Name:          "Normal",
		IconURL:       "",
		ClientID:      "status-desktop/dapp-browser",
		SharedAccount: types2.HexToAddress("0x1111"),
		ChainID:       0x1,
	}
	ephemeral := persistence.DApp{
		URL:           "https://ephemeral-dapp.com",
		Name:          "Ephemeral",
		IconURL:       "",
		ClientID:      "status-desktop/dapp-browser#ephemeral",
		SharedAccount: types2.HexToAddress("0x2222"),
		ChainID:       0x1,
	}
	require.NoError(t, persistence.UpsertDApp(state.walletDb, &normal))
	require.NoError(t, persistence.UpsertDApp(state.walletDb, &ephemeral))

	require.NoError(t, state.api.DeleteEphemeralDApps())

	gotNormal, err := persistence.SelectDApp(state.walletDb, normal.URL, normal.ClientID)
	require.NoError(t, err)
	require.NotNil(t, gotNormal)

	gotEphemeral, err := persistence.SelectDApp(state.walletDb, ephemeral.URL, ephemeral.ClientID)
	require.NoError(t, err)
	require.Nil(t, gotEphemeral)
}

func TestGetWCActiveSessions(t *testing.T) {
	state := setupTests(t)

	// Empty initially
	sessions, err := state.api.GetWCActiveSessions(state.ctx, 0)
	require.NoError(t, err)
	require.Empty(t, sessions)

	// Insert a WC session
	dappURL := "https://wc-test-dapp.com"
	require.NoError(t, persistence.UpsertDApp(state.walletDb, &persistence.DApp{
		URL: dappURL, Name: "WC Test", IconURL: "", ClientID: persistence.WCClientID,
		SharedAccount: types2.Address{}, ChainID: 0x1,
	}))
	require.NoError(t, persistence.UpsertWCSession(state.walletDb, "topic1", `{"session":"data"}`, 9999999999, "pairing1", dappURL, "symkey", 100))

	// Now should return the session
	sessions, err = state.api.GetWCActiveSessions(state.ctx, 0)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, "topic1", sessions[0].Topic)
}

func TestDisconnectWCSession(t *testing.T) {
	state := setupTests(t)

	// Disconnect non-existent session is idempotent
	err := state.api.DisconnectWCSession(state.ctx, "non-existent-topic")
	require.NoError(t, err)

	// Insert WC session and DApp
	dappURL := "https://wc-disconnect-test.com"
	require.NoError(t, persistence.UpsertDApp(state.walletDb, &persistence.DApp{
		URL: dappURL, Name: "WC Disconnect Test", IconURL: "", ClientID: persistence.WCClientID,
		SharedAccount: types2.Address{}, ChainID: 0x1,
	}))
	require.NoError(t, persistence.UpsertWCSession(state.walletDb, "topic-disconnect", `{"session":"data"}`, 9999999999, "pairing1", dappURL, "symkey", 100))

	session, err := persistence.SelectWCSession(state.walletDb, "topic-disconnect")
	require.NoError(t, err)
	require.NotNil(t, session)

	err = state.api.DisconnectWCSession(state.ctx, "topic-disconnect")
	require.NoError(t, err)

	session, err = persistence.SelectWCSession(state.walletDb, "topic-disconnect")
	require.NoError(t, err)
	require.Nil(t, session)
}

func TestUpdateWCSessionChains_EmptyChains(t *testing.T) {
	state := setupTests(t)

	err := state.api.UpdateWCSessionChains(state.ctx, "some-topic", "0x1234", []uint64{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "chains must not be empty")
}

func TestUpdateWCSessionChains_SessionNotFound(t *testing.T) {
	state := setupTests(t)

	err := state.api.UpdateWCSessionChains(state.ctx, "non-existent-topic", "0x1234", []uint64{1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "session not found")
}

func TestUpdateWCSessionChains_InvalidSessionJSON(t *testing.T) {
	state := setupTests(t)

	dappURL := "https://wc-update-test.com"
	require.NoError(t, persistence.UpsertDApp(state.walletDb, &persistence.DApp{
		URL: dappURL, Name: "WC Update Test", IconURL: "", ClientID: persistence.WCClientID,
		SharedAccount: types2.Address{}, ChainID: 0x1,
	}))
	// Store invalid JSON as session data
	require.NoError(t, persistence.UpsertWCSession(state.walletDb, "topic-update-invalid", `not-valid-json`, 9999999999, "pairing1", dappURL, "symkey", 100))

	err := state.api.UpdateWCSessionChains(state.ctx, "topic-update-invalid", "0x1234", []uint64{1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse session")
}

func TestEmitWCSessionEvent_InvalidDataJSON(t *testing.T) {
	state := setupTests(t)

	err := state.api.EmitWCSessionEvent(state.ctx, "some-topic", "accountsChanged", "invalid-json", "eip155:1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid dataJSON")
}

func TestEmitWCSessionEvent_EmptyDataJSON(t *testing.T) {
	state := setupTests(t)

	// Empty dataJSON should not fail on JSON parsing, but session won't be found
	err := state.api.EmitWCSessionEvent(state.ctx, "non-existent-topic", "accountsChanged", "", "eip155:1")
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrSessionNotFound)
}

func TestEmitWCSessionEvent_SessionNotFound(t *testing.T) {
	state := setupTests(t)

	err := state.api.EmitWCSessionEvent(state.ctx, "non-existent-topic", "chainChanged", `{"chainId":"0x1"}`, "eip155:1")
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrSessionNotFound)
}

func TestApproveWCSession_InvalidAccount(t *testing.T) {
	state := setupTests(t)

	_, err := state.api.ApproveWCSession(state.ctx, "proposal1", "not-an-address", "https://dapp.com", "DApp", "", []uint64{1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid account address")
}

func TestApproveWCSession_EmptyChains(t *testing.T) {
	state := setupTests(t)

	_, err := state.api.ApproveWCSession(state.ctx, "proposal1", "0x1234567890abcdef1234567890abcdef12345678", "https://dapp.com", "DApp", "", []uint64{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "supportedChains must not be empty")
}

func TestApproveWCSession_ProposalNotFound(t *testing.T) {
	state := setupTests(t)

	_, err := state.api.ApproveWCSession(state.ctx, "non-existent-proposal", "0x1234567890abcdef1234567890abcdef12345678", "https://dapp.com", "DApp", "", []uint64{1})
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrProposalNotFound)
}

func TestRejectWCSession_ProposalNotFound(t *testing.T) {
	state := setupTests(t)

	err := state.api.RejectWCSession(state.ctx, "non-existent-proposal")
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrProposalNotFound)
}

func TestApproveWCSessionRequest_InvalidRequestID(t *testing.T) {
	state := setupTests(t)

	err := state.api.ApproveWCSessionRequest(state.ctx, "some-topic", "not-a-number", "0xsignature")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request ID")
}

func TestApproveWCSessionRequest_SessionNotFound(t *testing.T) {
	state := setupTests(t)

	err := state.api.ApproveWCSessionRequest(state.ctx, "non-existent-topic", "12345", "0xsignature")
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrSessionNotFound)
}

func TestRejectWCSessionRequest_InvalidRequestID(t *testing.T) {
	state := setupTests(t)

	err := state.api.RejectWCSessionRequest(state.ctx, "some-topic", "not-a-number", 4001, "User rejected")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request ID")
}

func TestRejectWCSessionRequest_SessionNotFound(t *testing.T) {
	state := setupTests(t)

	err := state.api.RejectWCSessionRequest(state.ctx, "non-existent-topic", "12345", 4001, "User rejected")
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrSessionNotFound)
}

func TestPairWalletConnect_InvalidURI(t *testing.T) {
	state := setupTests(t)

	err := state.api.PairWalletConnect(state.ctx, "not-a-valid-wc-uri")
	require.Error(t, err)
}

func TestUpdateWCSessionChains_NilClient(t *testing.T) {
	state := setupTests(t)
	state.api.wcClient = nil

	err := state.api.UpdateWCSessionChains(state.ctx, "some-topic", "0x1234", []uint64{1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "WalletConnect client not initialized")
}

func TestEmitWCSessionEvent_NilClient(t *testing.T) {
	state := setupTests(t)
	state.api.wcClient = nil

	err := state.api.EmitWCSessionEvent(state.ctx, "some-topic", "accountsChanged", "", "eip155:1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "WalletConnect client not initialized")
}

func TestNewAPI_WithRestoredSessions(t *testing.T) {
	// Pre-insert a WC session so NewAPI enters the session restoration path.
	// ConnectAndResubscribe will fail to reach the relay (no server in tests)
	// but the error is non-fatal (logged as a warning).
	walletDb := createWalletDB(t)
	db := createDB(t)

	dappURL := "https://wc-restore-test.com"
	require.NoError(t, persistence.UpsertDApp(walletDb, &persistence.DApp{
		URL: dappURL, Name: "WC Restore Test", IconURL: "", ClientID: persistence.WCClientID,
		SharedAccount: types2.Address{}, ChainID: 0x1,
	}))
	require.NoError(t, persistence.UpsertWCSession(walletDb, "restore-topic-1", `{"session":"data"}`, 9999999999, "pairing1", dappURL, "symkey1", 100))

	networkManager := network.NewManager(db, nil)
	service := NewService(
		zap.NewNop(),
		walletDb,
		nil,
		nil,
		networkManager,
		&Config{},
	)

	api := NewAPI(service)
	require.NotNil(t, api)
}
