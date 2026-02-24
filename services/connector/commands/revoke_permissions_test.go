package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	types2 "github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

func TestFailToRevokePermissionsWithMissingDAppFields(t *testing.T) {
	state, close := setupCommand(t, Method_RevokePermissions)
	t.Cleanup(close)

	// Missing DApp fields
	request, err := ConstructRPCRequest("wallet_revokePermissions", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToRevokePermissionsForUnpermittedDApp(t *testing.T) {
	state, close := setupCommand(t, Method_RevokePermissions)
	t.Cleanup(close)

	request, err := ConstructRPCRequest("wallet_revokePermissions", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
	assert.Empty(t, result)
}

func TestRevokePermissionsSucceeded(t *testing.T) {
	state, close := setupCommand(t, Method_RevokePermissions)
	t.Cleanup(close)

	sharedAccount := types2.BytesToAddress(types2.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
	dAppPermissionRevoked := false

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorDAppPermissionRevoked:
			dAppPermissionRevoked = true
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	err := PersistDAppData(state.walletDb, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)

	request, err := ConstructRPCRequest("wallet_revokePermissions", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Empty(t, result)

	dApp, err := persistence.SelectDApp(state.walletDb, testDAppData.URL, testDAppData.ClientID)
	assert.NoError(t, err)
	assert.Nil(t, dApp)

	assert.True(t, dAppPermissionRevoked)
}

func TestRevokePermissionsDeletesWCSessions(t *testing.T) {
	state, close := setupCommand(t, Method_RevokePermissions)
	t.Cleanup(close)

	sharedAccount := types2.BytesToAddress(types2.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
	wcDAppData := signal.ConnectorDApp{
		URL:      "https://wc-test-dapp.com",
		Name:     "WC Test DApp",
		IconURL:  "https://wc-test-icon.com",
		ClientID: persistence.WCClientID,
	}

	// Persist WalletConnect DApp
	err := PersistDAppData(state.walletDb, wcDAppData, sharedAccount, 0x1)
	assert.NoError(t, err)

	// Insert 2 WC sessions for this DApp
	err = persistence.UpsertWCSession(state.walletDb, "topic1", `{"session":"data1"}`, 9999999999, "pairing1", wcDAppData.URL, "", 100)
	assert.NoError(t, err)
	err = persistence.UpsertWCSession(state.walletDb, "topic2", `{"session":"data2"}`, 9999999999, "pairing2", wcDAppData.URL, "", 200)
	assert.NoError(t, err)

	// Verify sessions exist
	sessions, err := persistence.SelectWCSessionsByDAppURL(state.walletDb, wcDAppData.URL)
	assert.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Revoke permissions
	request, err := ConstructRPCRequest("wallet_revokePermissions", []interface{}{}, &wcDAppData)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// Verify DApp is deleted
	dApp, err := persistence.SelectDApp(state.walletDb, wcDAppData.URL, wcDAppData.ClientID)
	assert.NoError(t, err)
	assert.Nil(t, dApp)

	// Verify WC sessions are also deleted
	sessions, err = persistence.SelectWCSessionsByDAppURL(state.walletDb, wcDAppData.URL)
	assert.NoError(t, err)
	assert.Len(t, sessions, 0, "WC sessions should be deleted when DApp permissions are revoked")

	// Verify individual sessions are deleted
	session1, err := persistence.SelectWCSession(state.walletDb, "topic1")
	assert.NoError(t, err)
	assert.Nil(t, session1)

	session2, err := persistence.SelectWCSession(state.walletDb, "topic2")
	assert.NoError(t, err)
	assert.Nil(t, session2)
}
