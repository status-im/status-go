package commands

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/connector/chainutils"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func TestFailToGetChainIdWithMissingDAppFields(t *testing.T) {
	state, close := setupCommand(t, Method_EthChainId)
	t.Cleanup(close)

	// Missing DApp fields
	request, err := ConstructRPCRequest("eth_chainId", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestGetDefaultChainIdForUnpermittedDApp(t *testing.T) {
	state, close := setupCommand(t, Method_EthChainId)
	t.Cleanup(close)

	request, err := ConstructRPCRequest("eth_chainId", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	chainId, err := chainutils.GetHexChainID(strconv.FormatUint(walletCommon.EthereumMainnet, 16))
	assert.NoError(t, err)
	assert.Equal(t, result, chainId)
}

func TestGetChainIdForPermittedDApp(t *testing.T) {
	state, close := setupCommand(t, Method_EthChainId)
	t.Cleanup(close)

	sharedAccount := types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg")
	chainID := uint64(0x123)

	err := PersistDAppData(state.walletDb, testDAppData, sharedAccount, chainID)
	assert.NoError(t, err)

	request, err := ConstructRPCRequest("eth_chainId", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	chainId := fmt.Sprintf(`0x%s`, strconv.FormatUint(chainID, 16))
	assert.Equal(t, chainId, response)
}
