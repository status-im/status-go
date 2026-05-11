package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
)

func TestFailToRequestAccountsWithMissingDAppFields(t *testing.T) {
	state, close := setupCommand(t, Method_EthRequestAccounts)
	t.Cleanup(close)

	// Missing DApp fields
	request, err := ConstructRPCRequest("eth_requestAccounts", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestRequestAccountsWithSignalTimeout(t *testing.T) {
	state, close := setupCommand(t, Method_EthRequestAccounts)
	t.Cleanup(close)

	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x01})
	assert.NoError(t, err)

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrWalletResponseTimeout, err)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestRequestAccountsAcceptedAndRequestAgain(t *testing.T) {
	state, close := setupCommand(t, Method_EthRequestAccounts)
	t.Cleanup(close)

	request, err := ConstructRPCRequest("eth_requestAccounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	accountAddress := types.Address{0x03}
	dAppPermissionGranted := false

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSendRequestAccounts:
			var ev signal.ConnectorSendRequestAccountsSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = state.handler.RequestAccountsAccepted(RequestAccountsAcceptedArgs{
				RequestID: ev.RequestID,
				Account:   accountAddress,
				ChainID:   walletCommon.EthereumMainnet,
			})
			assert.NoError(t, err)
		case signal.EventConnectorDAppPermissionGranted:
			dAppPermissionGranted = true
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	expectedResponse := FormatAccountAddressToResponse(accountAddress)
	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

	// Check dApp in the database
	dApp, err := persistence.SelectDApp(state.walletDb, request.URL, request.ClientID)
	assert.NoError(t, err)
	assert.Equal(t, request.Name, dApp.Name)
	assert.Equal(t, request.IconURL, dApp.IconURL)
	assert.Equal(t, request.ClientID, dApp.ClientID)
	assert.Equal(t, accountAddress, dApp.SharedAccount)
	assert.Equal(t, walletCommon.EthereumMainnet, dApp.ChainID)

	// This should not invoke UI side
	response, err = state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

	assert.True(t, dAppPermissionGranted)
}

func TestRequestAccountsRejected(t *testing.T) {
	state, close := setupCommand(t, Method_EthRequestAccounts)
	t.Cleanup(close)

	request, err := ConstructRPCRequest("eth_requestAccounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSendRequestAccounts:
			var ev signal.ConnectorSendRequestAccountsSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = state.handler.RequestAccountsRejected(RejectedArgs{
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestAccountsRejectedByUser, err)
}

// TestRequestAccounts_EphemeralDoesNotReuseNormalSession verifies that an
// ephemeral (incognito) clientID never silently reuses a permission that was
// granted to the normal-session clientID for the same dApp origin.
func TestRequestAccounts_EphemeralDoesNotReuseNormalSession(t *testing.T) {
	state, close := setupCommand(t, Method_EthRequestAccounts)
	t.Cleanup(close)

	normalClientID := "status-desktop/dapp-browser"
	ephemeralClientID := "status-desktop/dapp-browser" + persistence.EphemeralClientIDSuffix

	origin := "https://some-dapp.com"
	normalAccount := types.Address{0xAA}
	ephemeralAccount := types.Address{0xBB}

	// Pre-populate a normal-session dApp + eth_accounts permission.
	normalDApp := &persistence.DApp{
		URL: origin, Name: "SomeDApp", IconURL: "",
		ClientID: normalClientID, SharedAccount: normalAccount, ChainID: 1,
	}
	require.NoError(t, persistence.UpsertDApp(state.walletDb, normalDApp))
	require.NoError(t, persistence.InsertPermission(state.walletDb, origin, normalClientID, Method_EthAccounts, []persistence.Caveat{}, 1))

	// Now call eth_requestAccounts with the ephemeral clientID for the same origin.
	// This MUST NOT silently reuse the normal-session permission — it must trigger the share UI.
	sharePromptInvoked := false
	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		require.NoError(t, json.Unmarshal(s, &evt))
		if evt.Type == signal.EventConnectorSendRequestAccounts {
			sharePromptInvoked = true
			var ev signal.ConnectorSendRequestAccountsSignal
			require.NoError(t, json.Unmarshal(evt.Event, &ev))
			// Simulate user approving with the ephemeral account.
			require.NoError(t, state.handler.RequestAccountsAccepted(RequestAccountsAcceptedArgs{
				RequestID: ev.RequestID,
				Account:   ephemeralAccount,
				ChainID:   1,
			}))
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	ephemeralDApp := signal.ConnectorDApp{
		URL: origin, Name: "SomeDApp", IconURL: "", ClientID: ephemeralClientID,
	}
	request, err := ConstructRPCRequest(Method_EthRequestAccounts, []interface{}{}, &ephemeralDApp)
	require.NoError(t, err)

	response, err := state.cmd.Execute(state.ctx, request)
	require.NoError(t, err)
	require.True(t, sharePromptInvoked, "share prompt must fire for ephemeral clientID even when normal session has permission")

	// The returned account must be the ephemeral one, not the normal-session one.
	require.Equal(t, FormatAccountAddressToResponse(ephemeralAccount), response)

	// Normal-session dApp must be completely untouched.
	got, err := persistence.SelectDApp(state.walletDb, origin, normalClientID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, normalAccount, got.SharedAccount)
}

func TestRequestAccountsWithExistingDApp(t *testing.T) {
	state, close := setupCommand(t, Method_EthRequestAccounts)
	t.Cleanup(close)

	// Pre-create a dApp
	accountAddress := types.Address{0x05}
	dApp := &persistence.DApp{
		URL:           testDAppData.URL,
		Name:          testDAppData.Name,
		IconURL:       testDAppData.IconURL,
		ClientID:      testDAppData.ClientID,
		SharedAccount: accountAddress,
		ChainID:       walletCommon.EthereumMainnet,
	}
	err := persistence.UpsertDApp(state.walletDb, dApp)
	assert.NoError(t, err)
	err = persistence.InsertPermission(state.walletDb, testDAppData.URL, testDAppData.ClientID, Method_EthAccounts, []persistence.Caveat{}, time.Now().Unix())
	assert.NoError(t, err)

	request, err := ConstructRPCRequest("eth_requestAccounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	// Should not trigger signal since dApp already exists
	expectedResponse := FormatAccountAddressToResponse(accountAddress)
	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}
