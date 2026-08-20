package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/pkg/services/connector/database"
)

func TestFailToGetAccountWithMissingDAppFields(t *testing.T) {
	state, close := setupCommand(t, Method_EthAccounts)
	t.Cleanup(close)

	// Missing DApp fields
	request, err := ConstructRPCRequest("eth_accounts", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToGetAccountForUnpermittedDApp(t *testing.T) {
	state, close := setupCommand(t, Method_EthAccounts)
	t.Cleanup(close)

	request, err := ConstructRPCRequest("eth_accounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, result)
}

func TestGetAccountForPermittedDApp(t *testing.T) {
	state, close := setupCommand(t, Method_EthAccounts)
	t.Cleanup(close)

	sharedAccount := types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg")

	err := PersistDAppData(state.walletDb, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)
	err = persistence.InsertPermission(state.walletDb, testDAppData.URL, testDAppData.ClientID, Method_EthAccounts, []persistence.Caveat{}, time.Now().Unix())
	assert.NoError(t, err)

	request, err := ConstructRPCRequest("eth_accounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	expectedResponse := FormatAccountAddressToResponse(sharedAccount)
	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}
