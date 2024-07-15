package connector

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/connector/commands"
	"github.com/status-im/status-go/signal"
)

func TestRequestAccountsSwitchChainAndSendTransactionFlow(t *testing.T) {
	db, close := createDB(t)
	defer close()

	nm := commands.NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: 0x1,
			Layer:   1,
		},
		{
			ChainID: 0x5,
			Layer:   1,
		},
	})
	rpc := commands.RPCClientMock{}

	service := NewService(db, &rpc, &nm)

	api := NewAPI(service)

	// Try to request accounts without permission
	request := "{\"method\":\"eth_accounts\",\"params\":[],\"origin\":\"http://testDAppURL123\",\"dAppName\":\"testDAppName\",\"dAppIconUrl\":\"http://testDAppIconUrl\"}"
	response, err := api.CallRPC(request)
	assert.Empty(t, response)
	assert.Error(t, err)
	assert.Equal(t, commands.ErrDAppIsNotPermittedByUser, err)

	accountAddress := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
	expectedHash := types.BytesToHash(types.FromHex("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"))

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt commands.EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSendRequestAccounts:
			var ev signal.ConnectorSendRequestAccounts
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = api.RequestAccountsFinished(commands.RequestAccountsFinishedArgs{
				Account: accountAddress,
				ChainID: 0x1,
				Error:   nil,
			})
			assert.NoError(t, err)
		case signal.EventConnectorSendTransaction:
			var ev signal.ConnectorSendTransactionSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = api.SendTransactionFinished(commands.SendTransactionFinishedArgs{
				Hash:  expectedHash,
				Error: nil,
			})
			assert.NoError(t, err)
		}
	}))

	// Request accounts, now for real
	request = "{\"method\": \"eth_requestAccounts\", \"params\": [], \"origin\": \"http://testDAppURL123\", \"dAppName\": \"testDAppName\", \"dAppIconUrl\": \"http://testDAppIconUrl\" }"
	expectedResponse := strings.ToLower(fmt.Sprintf(`{"accounts":["%s"]}`, accountAddress.Hex()))
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

	// Request to switch ethereum chain
	expectedChainId := 0x5
	request = fmt.Sprintf("{\"method\": \"wallet_switchEthereumChain\", \"params\": [%d], \"origin\": \"http://testDAppURL123\", \"dAppName\": \"testDAppName\", \"dAppIconUrl\": \"http://testDAppIconUrl\" }", expectedChainId)
	expectedResponse = fmt.Sprintf(`%d`, expectedChainId)
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

	// Check if the chain was switched
	request = "{\"method\": \"eth_chainId\", \"params\": [], \"origin\": \"http://testDAppURL123\", \"dAppName\": \"testDAppName\", \"dAppIconUrl\": \"http://testDAppIconUrl\" }"
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

	// Check the account after switching chain
	request = "{\"method\": \"eth_accounts\", \"params\": [], \"origin\": \"http://testDAppURL123\", \"dAppName\": \"testDAppName\", \"dAppIconUrl\": \"http://testDAppIconUrl\" }"
	expectedResponse = strings.ToLower(fmt.Sprintf(`{"accounts":["%s"]}`, accountAddress.Hex()))
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

	// Send transaction
	request = fmt.Sprintf("{\"method\": \"eth_sendTransaction\", \"params\":[{\"from\":\"%s\",\"to\":\"0x0200000000000000000000000000000000000000\",\"gas\":null,\"gasPrice\":null,\"value\":\"0x0\",\"nonce\":null,\"maxFeePerGas\":null,\"maxPriorityFeePerGas\":null,\"input\":\"0x\",\"data\":\"0x307830\",\"MultiTransactionID\":0,\"Symbol\":\"\"}], \"origin\": \"http://testDAppURL123\", \"dAppName\": \"testDAppName\", \"dAppIconUrl\": \"http://testDAppIconUrl\" }", accountAddress.Hex())
	expectedResponse = expectedHash.Hex()
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}
