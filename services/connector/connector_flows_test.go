package connector

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/connector/chainutils"
	"github.com/status-im/status-go/services/connector/commands"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
)

func TestRequestAccountsSwitchChainAndSendTransactionFlow(t *testing.T) {
	db, close := createDB(t)
	defer close()

	nm := commands.NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: walletCommon.EthereumMainnet,
			Layer:   1,
		},
		{
			ChainID: walletCommon.OptimismMainnet,
			Layer:   1,
		},
	})
	rpc := commands.RPCClientMock{}

	service := NewService(db, &rpc, &nm)

	api := NewAPI(service)

	accountAddress := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
	expectedHash := types.BytesToHash(types.FromHex("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"))
	expectedSignature := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	dAppPermissionRevoked := false
	dAppPermissionGranted := false

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt commands.EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorDAppPermissionRevoked:
			dAppPermissionRevoked = true
		case signal.EventConnectorDAppPermissionGranted:
			dAppPermissionGranted = true
		case signal.EventConnectorSendRequestAccounts:
			var ev signal.ConnectorSendRequestAccountsSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = api.RequestAccountsAccepted(commands.RequestAccountsAcceptedArgs{
				RequestID: ev.RequestID,
				Account:   accountAddress,
				ChainID:   0x1,
			})
			assert.NoError(t, err)
		case signal.EventConnectorSendTransaction:
			var ev signal.ConnectorSendTransactionSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = api.SendTransactionAccepted(commands.SendTransactionAcceptedArgs{
				RequestID: ev.RequestID,
				Hash:      expectedHash,
			})
			assert.NoError(t, err)
		case signal.EventConnectorPersonalSign:
			var ev signal.ConnectorPersonalSignSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = api.PersonalSignAccepted(commands.PersonalSignAcceptedArgs{
				RequestID: ev.RequestID,
				Signature: expectedSignature,
			})
			assert.NoError(t, err)
		}
	}))

	// Request accounts, now for real
	request := "{\"method\": \"eth_requestAccounts\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err := api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, commands.FormatAccountAddressToResponse(accountAddress), response)
	assert.Equal(t, true, dAppPermissionGranted)
	assert.Equal(t, false, dAppPermissionRevoked)

	// Request to switch ethereum chain
	expectedChainID, err := chainutils.GetHexChainID(walletCommon.ChainID(walletCommon.EthereumMainnet).String())
	assert.NoError(t, err)
	request = fmt.Sprintf("{\"method\": \"wallet_switchEthereumChain\", \"params\": [{\"chainId\": \"%s\"}], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }", expectedChainID)
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedChainID, response)

	// Check if the chain was switched
	request = "{\"method\": \"eth_chainId\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedChainID, response)

	// Check the account after switching chain
	request = "{\"method\": \"eth_accounts\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, commands.FormatAccountAddressToResponse(accountAddress), response)

	// Send transaction
	request = fmt.Sprintf("{\"method\": \"eth_sendTransaction\", \"params\":[{\"from\":\"%s\",\"to\":\"0x0200000000000000000000000000000000000000\",\"value\":\"0x12345\",\"data\":\"0x307830\"}], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }", accountAddress.Hex())
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedHash.Hex(), response)

	// Personal sign
	request = "{\"method\": \"personal_sign\", \"params\":[{\"challenge\": \"0x506c65617365207369676e2074686973206d65737361676520746f20636f6e6669726d20796f7572206964656e746974792e\",\"address\":\"0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7\"}], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err = api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedSignature, response)

	// Revoke permissions
	request = "{\"method\": \"wallet_revokePermissions\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	_, err = api.CallRPC(request)
	assert.NoError(t, err)

	// Check if the account was revoked
	request = fmt.Sprintf("{\"method\": \"eth_sendTransaction\", \"params\":[{\"from\":\"%s\",\"to\":\"0x0200000000000000000000000000000000000000\",\"value\":\"0x12345\",\"data\":\"0x307830\"}], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }", accountAddress.Hex())
	response, err = api.CallRPC(request)
	assert.Empty(t, response)
	assert.Error(t, err)
	assert.Equal(t, commands.ErrDAppIsNotPermittedByUser, err)
	assert.Equal(t, true, dAppPermissionRevoked)
}

func TestForwardedRPCs(t *testing.T) {
	db, close := createDB(t)
	defer close()

	rpc := commands.RPCClientMock{}
	service := NewService(db, &rpc, nil)

	api := NewAPI(service)

	sharedAccount := types.BytesToAddress(types.FromHex("0x3d0ab2a774b74bb1d36f97700315adf962c69fct"))

	testDAppData := signal.ConnectorDApp{
		URL:     "https://app.test.org",
		Name:    "testDAppName",
		IconURL: "https://app.test.icon.org",
	}

	request := "{\"method\": \"eth_blockNumber\", \"params\":[],\"url\":\"https://app.test.org\",\"name\":\"testDAppName\",\"iconUrl\":\"http://testDAppIconUrl\"}"
	_, err := api.CallRPC(request)
	assert.Equal(t, commands.ErrDAppIsNotPermittedByUser, err)

	err = commands.PersistDAppData(db, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)

	expectedResponse := "0xaa37dc"
	rpc.SetResponse(fmt.Sprintf(`{"jsonrpc":"2.0","id":37,"result":"%s"}`, expectedResponse))

	response, err := api.CallRPC(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func TestRequestAccountsAfterPermisasionsRevokeTest(t *testing.T) {
	db, close := createDB(t)
	defer close()

	nm := commands.NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: walletCommon.EthereumMainnet,
			Layer:   1,
		},
		{
			ChainID: walletCommon.OptimismMainnet,
			Layer:   1,
		},
	})
	rpc := commands.RPCClientMock{}

	service := NewService(db, &rpc, &nm)

	api := NewAPI(service)

	accountAddress := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
	dAppPermissionRevoked := false
	dAppPermissionGranted := false

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt commands.EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorDAppPermissionRevoked:
			dAppPermissionRevoked = true
		case signal.EventConnectorDAppPermissionGranted:
			dAppPermissionGranted = true
		case signal.EventConnectorSendRequestAccounts:
			var ev signal.ConnectorSendRequestAccountsSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = api.RequestAccountsAccepted(commands.RequestAccountsAcceptedArgs{
				RequestID: ev.RequestID,
				Account:   accountAddress,
				ChainID:   0x1,
			})
			assert.NoError(t, err)
		}
	}))

	for range [10]int{} {
		dAppPermissionRevoked = false
		dAppPermissionGranted = false

		// Request accounts
		request := "{\"method\": \"eth_requestAccounts\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
		response, err := api.CallRPC(request)
		assert.NoError(t, err)
		assert.Equal(t, commands.FormatAccountAddressToResponse(accountAddress), response)
		assert.Equal(t, true, dAppPermissionGranted)
		assert.Equal(t, false, dAppPermissionRevoked)

		// Revoke permissions
		request = "{\"method\": \"wallet_revokePermissions\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
		_, err = api.CallRPC(request)
		assert.NoError(t, err)
		assert.Equal(t, true, dAppPermissionRevoked)
	}
}
