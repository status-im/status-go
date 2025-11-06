package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func TestNetVersionWithoutDApp(t *testing.T) {
	state, close := setupCommand(t, "net_version")
	t.Cleanup(close)

	request, err := ConstructRPCRequest("net_version", []interface{}{}, &testDAppData)
	require.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	require.NoError(t, err)

	// Should return a decimal string
	assert.Equal(t, "1", result)
}

func TestNetVersionWithExistingDApp(t *testing.T) {
	state, close := setupCommand(t, "net_version")
	t.Cleanup(close)

	dApp := &persistence.DApp{
		URL:           testDAppData.URL,
		Name:          testDAppData.Name,
		IconURL:       testDAppData.IconURL,
		ClientID:      testDAppData.ClientID,
		SharedAccount: [20]byte{0x01},
		ChainID:       walletCommon.OptimismMainnet,
	}
	err := persistence.UpsertDApp(state.walletDb, dApp)
	require.NoError(t, err)

	request, err := ConstructRPCRequest("net_version", []interface{}{}, &testDAppData)
	require.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	require.NoError(t, err)

	assert.Equal(t, "10", result)
}

func TestNetVersionValidationError(t *testing.T) {
	state, close := setupCommand(t, "net_version")
	t.Cleanup(close)

	// Missing DApp data
	request, err := ConstructRPCRequest("net_version", []interface{}{}, nil)
	require.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}
