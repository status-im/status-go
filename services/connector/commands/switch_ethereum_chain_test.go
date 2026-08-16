package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto/types"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
)

func TestFailToSwitchEthereumChainWithMissingDAppFields(t *testing.T) {
	state, close := setupCommand(t, Method_SwitchEthereumChain)
	t.Cleanup(close)

	// Missing DApp fields
	request, err := ConstructRPCRequest("wallet_switchEthereumChain", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToSwitchEthereumChainWithNoChainId(t *testing.T) {
	state, close := setupCommand(t, Method_SwitchEthereumChain)
	t.Cleanup(close)

	request, err := ConstructRPCRequest("wallet_switchEthereumChain", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrEmptyRPCParams, err)
}

func TestFailToSwitchEthereumChainWithUnsupportedChainId(t *testing.T) {
	state, close := setupCommand(t, Method_SwitchEthereumChain)
	t.Cleanup(close)

	params := make([]interface{}, 1)
	params[0] = map[string]interface{}{
		"chainId": "0x1a343",
	} // some unrecoginzed chain id

	request, err := ConstructRPCRequest("wallet_switchEthereumChain", params, &testDAppData)
	assert.NoError(t, err)

	_, err = state.cmd.Execute(state.ctx, request)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnsupportedNetwork),
		"unknown chain error must still satisfy errors.Is(ErrUnsupportedNetwork) for backward compatibility, got %v", err)
}

func TestSwitchEthereumChainSuccess(t *testing.T) {
	state, close := setupCommand(t, Method_SwitchEthereumChain)
	t.Cleanup(close)

	chainId := fmt.Sprintf(`0x%s`, walletCommon.ChainID(walletCommon.EthereumMainnet).String())
	chainIdSwitched := false

	signal.SetHandler(signal.Handler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorDAppChainIdSwitched:
			var ev signal.ConnectorDAppChainIdSwitchedSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			assert.Equal(t, chainId, ev.ChainId)
			assert.Equal(t, testDAppData.URL, ev.URL)
			chainIdSwitched = true
		}
	}))
	t.Cleanup(signal.ResetHandler)

	params := make([]interface{}, 1)
	params[0] = map[string]interface{}{
		"chainId": "0x1",
	}

	request, err := ConstructRPCRequest("wallet_switchEthereumChain", params, &testDAppData)
	assert.NoError(t, err)

	err = PersistDAppData(state.walletDb, testDAppData, types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"), walletCommon.EthereumMainnet)
	assert.NoError(t, err)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, chainId, response)
	assert.True(t, chainIdSwitched)
}
