package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
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

	sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
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

	dApp, err := persistence.SelectDAppByUrl(state.walletDb, testDAppData.URL)
	assert.NoError(t, err)
	assert.Nil(t, dApp)

	assert.True(t, dAppPermissionRevoked)
}
