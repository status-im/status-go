package commands

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
)

func TestFailToSwitchEthereumChainWithMissingDAppFields(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &SwitchEthereumChainCommand{Db: db}

	// Missing DApp fields
	request, err := ConstructRPCRequest("wallet_switchEthereumChain", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToSwitchEthereumChainWithNoChainId(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &SwitchEthereumChainCommand{Db: db}

	request, err := ConstructRPCRequest("wallet_switchEthereumChain", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrEmptyRPCParams, err)
}

func TestFailToSwitchEthereumChainWithUnsupportedChainId(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	nm := NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: walletCommon.EthereumMainnet,
		},
	})

	cmd := &SwitchEthereumChainCommand{
		Db:             db,
		NetworkManager: &nm,
	}

	params := make([]interface{}, 1)
	params[0] = map[string]interface{}{
		"chainId": "0x1a343",
	} // some unrecoginzed chain id

	request, err := ConstructRPCRequest("wallet_switchEthereumChain", params, &testDAppData)
	assert.NoError(t, err)

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrUnsupportedNetwork, err)
}

func TestSwitchEthereumChainSuccess(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	nm := NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: walletCommon.EthereumMainnet,
		},
		{
			ChainID: walletCommon.EthereumGoerli,
		},
	})

	chainId := fmt.Sprintf(`0x%s`, walletCommon.ChainID(walletCommon.EthereumMainnet).String())
	chainIdSwitched := false

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
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
	t.Cleanup(signal.ResetMobileSignalHandler)

	cmd := &SwitchEthereumChainCommand{
		Db:             db,
		NetworkManager: &nm,
	}

	params := make([]interface{}, 1)
	params[0] = map[string]interface{}{
		"chainId": "0x1",
	}

	request, err := ConstructRPCRequest("wallet_switchEthereumChain", params, &testDAppData)
	assert.NoError(t, err)

	err = PersistDAppData(db, testDAppData, types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"), walletCommon.EthereumMainnet)
	assert.NoError(t, err)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, chainId, response)
	assert.True(t, chainIdSwitched)
}
