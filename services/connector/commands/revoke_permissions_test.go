package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/signal"
	persistence "github.com/status-im/status-go/services/connector/database"
)

func trackRevokedSignal(t *testing.T) *bool {
	t.Helper()
	revoked := false
	signal.SetHandler(signal.Handler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)
		if evt.Type == signal.EventConnectorDAppPermissionRevoked {
			revoked = true
		}
	}))
	t.Cleanup(signal.ResetHandler)
	return &revoked
}

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

	sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcf"))
	dAppPermissionRevoked := false

	signal.SetHandler(signal.Handler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorDAppPermissionRevoked:
			dAppPermissionRevoked = true
		}
	}))
	t.Cleanup(signal.ResetHandler)

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

	sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcf"))
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

func TestRevokePermissions_TargetedEthAccounts_KeepsDApp_NoRevokedSignal(t *testing.T) {
	tests := []struct {
		name         string
		insertedPerm []string
		expectedPerm []string
	}{
		{
			name:         "keeps non-revoked permissions",
			insertedPerm: []string{"eth_accounts", "personal_sign"},
			expectedPerm: []string{"personal_sign"},
		},
		{
			name:         "removes only permission when single capability exists",
			insertedPerm: []string{"eth_accounts"},
			expectedPerm: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, close := setupCommand(t, Method_RevokePermissions)
			t.Cleanup(close)

			sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcf"))
			revoked := trackRevokedSignal(t)

			err := PersistDAppData(state.walletDb, testDAppData, sharedAccount, 0x123)
			assert.NoError(t, err)
			for i, perm := range tt.insertedPerm {
				err = persistence.InsertPermission(state.walletDb, testDAppData.URL, testDAppData.ClientID, perm, []persistence.Caveat{}, time.Now().Unix()+int64(i))
				assert.NoError(t, err)
			}

			params := []interface{}{
				map[string]interface{}{
					"eth_accounts": map[string]interface{}{},
				},
			}
			request, err := ConstructRPCRequest("wallet_revokePermissions", params, &testDAppData)
			assert.NoError(t, err)

			_, err = state.cmd.Execute(state.ctx, request)
			assert.NoError(t, err)

			dApp, err := persistence.SelectDApp(state.walletDb, testDAppData.URL, testDAppData.ClientID)
			assert.NoError(t, err)
			assert.NotNil(t, dApp)

			perms, err := persistence.SelectPermissions(state.walletDb, testDAppData.URL, testDAppData.ClientID)
			assert.NoError(t, err)
			assert.Len(t, perms, len(tt.expectedPerm))
			actualPerms := make([]string, 0, len(perms))
			for _, perm := range perms {
				actualPerms = append(actualPerms, perm.ParentCapability)
			}
			assert.ElementsMatch(t, tt.expectedPerm, actualPerms)
			assert.False(t, *revoked)
		})
	}
}

func TestRevokePermissions_MultiElementParams_RemovesAllCapabilities(t *testing.T) {
	state, close := setupCommand(t, Method_RevokePermissions)
	t.Cleanup(close)

	sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcf"))
	revoked := trackRevokedSignal(t)

	err := PersistDAppData(state.walletDb, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)
	err = persistence.InsertPermission(state.walletDb, testDAppData.URL, testDAppData.ClientID, "eth_accounts", []persistence.Caveat{}, time.Now().Unix())
	assert.NoError(t, err)
	err = persistence.InsertPermission(state.walletDb, testDAppData.URL, testDAppData.ClientID, "personal_sign", []persistence.Caveat{}, time.Now().Unix()+1)
	assert.NoError(t, err)

	params := []interface{}{
		map[string]interface{}{"eth_accounts": map[string]interface{}{}},
		map[string]interface{}{"personal_sign": map[string]interface{}{}},
	}
	request, err := ConstructRPCRequest("wallet_revokePermissions", params, &testDAppData)
	assert.NoError(t, err)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)

	dApp, err := persistence.SelectDApp(state.walletDb, testDAppData.URL, testDAppData.ClientID)
	assert.NoError(t, err)
	assert.NotNil(t, dApp)

	perms, err := persistence.SelectPermissions(state.walletDb, testDAppData.URL, testDAppData.ClientID)
	assert.NoError(t, err)
	assert.Empty(t, perms)
	assert.False(t, *revoked)
}

func TestRevokePermissions_InvalidParamsType(t *testing.T) {
	state, close := setupCommand(t, Method_RevokePermissions)
	t.Cleanup(close)

	sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcf"))
	err := PersistDAppData(state.walletDb, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)
	err = persistence.InsertPermission(state.walletDb, testDAppData.URL, testDAppData.ClientID, "eth_accounts", []persistence.Caveat{}, time.Now().Unix())
	assert.NoError(t, err)

	request, err := ConstructRPCRequest("wallet_revokePermissions", []interface{}{"not-a-map"}, &testDAppData)
	assert.NoError(t, err)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidParamType, err)
}
