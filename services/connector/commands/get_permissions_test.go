package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	persistence "github.com/status-im/status-go/services/connector/database"
)

func TestGetPermissionsWithNoPermissions(t *testing.T) {
	state, close := setupCommand(t, Method_RequestPermissions)
	t.Cleanup(close)

	getPermissionsCmd := NewGetPermissionsCommand(state.walletDb)

	request, err := ConstructRPCRequest("wallet_getPermissions", []interface{}{}, &testDAppData)
	require.NoError(t, err)

	result, err := getPermissionsCmd.Execute(state.ctx, request)
	require.NoError(t, err)

	permissions, ok := result.([]persistence.Permission)
	assert.True(t, ok)
	assert.Empty(t, permissions)
}

func TestGetPermissionsWithExistingPermissions(t *testing.T) {
	state, close := setupCommand(t, Method_RequestPermissions)
	t.Cleanup(close)

	dApp := &persistence.DApp{
		URL:           testDAppData.URL,
		Name:          testDAppData.Name,
		IconURL:       testDAppData.IconURL,
		ClientID:      testDAppData.ClientID,
		SharedAccount: [20]byte{0x01},
		ChainID:       1,
	}
	err := persistence.UpsertDApp(state.walletDb, dApp)
	require.NoError(t, err)

	// Insert a permission
	caveats := []persistence.Caveat{{Type: "test", Value: "value"}}
	err = persistence.InsertPermission(state.walletDb, testDAppData.URL, testDAppData.ClientID, "eth_accounts", caveats, 123)
	require.NoError(t, err)

	getPermissionsCmd := NewGetPermissionsCommand(state.walletDb)

	request, err := ConstructRPCRequest("wallet_getPermissions", []interface{}{}, &testDAppData)
	require.NoError(t, err)

	result, err := getPermissionsCmd.Execute(state.ctx, request)
	require.NoError(t, err)

	permissions, ok := result.([]persistence.Permission)
	assert.True(t, ok)
	assert.Len(t, permissions, 1)
	assert.Equal(t, "eth_accounts", permissions[0].ParentCapability)
}

func TestGetPermissionsValidationError(t *testing.T) {
	state, close := setupCommand(t, Method_RequestPermissions)
	t.Cleanup(close)

	getPermissionsCmd := NewGetPermissionsCommand(state.walletDb)

	// Missing DApp data
	request, err := ConstructRPCRequest("wallet_getPermissions", []interface{}{}, nil)
	require.NoError(t, err)

	result, err := getPermissionsCmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}
