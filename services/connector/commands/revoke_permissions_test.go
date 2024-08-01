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
	cmd := &RequestPermissionsCommand{}

	// Missing DApp fields
	request, err := ConstructRPCRequest("wallet_revokePermissions", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToRevokePermissionsForUnpermittedDApp(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &RevokePermissionsCommand{Db: db}

	request, err := ConstructRPCRequest("wallet_revokePermissions", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
	assert.Empty(t, result)
}

func TestRevokePermissionsSucceeded(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &RevokePermissionsCommand{Db: db}

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

	err := PersistDAppData(db, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)

	request, err := ConstructRPCRequest("wallet_revokePermissions", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Empty(t, result)

	dApp, err := persistence.SelectDAppByUrl(db, testDAppData.URL)
	assert.NoError(t, err)
	assert.Nil(t, dApp)

	assert.True(t, dAppPermissionRevoked)
}
