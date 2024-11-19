package connector

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/types"
	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	"github.com/status-im/status-go/services/connector/chainutils"
	"github.com/status-im/status-go/services/connector/commands"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
)

func TestRequestAccountsSwitchChainAndSendTransactionFlow(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

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

			err = state.api.RequestAccountsAccepted(commands.RequestAccountsAcceptedArgs{
				RequestID: ev.RequestID,
				Account:   accountAddress,
				ChainID:   0x1,
			})
			assert.NoError(t, err)
		case signal.EventConnectorSendTransaction:
			var ev signal.ConnectorSendTransactionSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = state.api.SendTransactionAccepted(commands.SendTransactionAcceptedArgs{
				RequestID: ev.RequestID,
				Hash:      expectedHash,
			})
			assert.NoError(t, err)
		case signal.EventConnectorSign:
			var ev signal.ConnectorSignSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = state.api.SignAccepted(commands.SignAcceptedArgs{
				RequestID: ev.RequestID,
				Signature: expectedSignature,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	// Request accounts, now for real
	request := "{\"method\": \"eth_requestAccounts\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err := state.api.CallRPC(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, commands.FormatAccountAddressToResponse(accountAddress), response)
	assert.Equal(t, true, dAppPermissionGranted)
	assert.Equal(t, false, dAppPermissionRevoked)

	// Request to switch ethereum chain
	expectedChainID, err := chainutils.GetHexChainID(walletCommon.ChainID(walletCommon.EthereumMainnet).String())
	assert.NoError(t, err)
	request = fmt.Sprintf("{\"method\": \"wallet_switchEthereumChain\", \"params\": [{\"chainId\": \"%s\"}], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }", expectedChainID)
	response, err = state.api.CallRPC(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, expectedChainID, response)

	// Check if the chain was switched
	request = "{\"method\": \"eth_chainId\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err = state.api.CallRPC(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, expectedChainID, response)

	// Check the account after switching chain
	request = "{\"method\": \"eth_accounts\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err = state.api.CallRPC(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, commands.FormatAccountAddressToResponse(accountAddress), response)

	// Send transaction
	mockedChainClient := mock_client.NewMockClientInterface(state.mockCtrl)
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().SuggestGasPrice(state.ctx).Times(1).Return(big.NewInt(1), nil)
	mockedChainClient.EXPECT().SuggestGasTipCap(state.ctx).Times(1).Return(big.NewInt(0), errors.New("EIP-1559 is not enabled"))
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().PendingNonceAt(state.ctx, common.Address(accountAddress)).Times(1).Return(uint64(10), nil)

	request = fmt.Sprintf("{\"method\": \"eth_sendTransaction\", \"params\":[{\"from\":\"%s\",\"to\":\"0x0200000000000000000000000000000000000000\",\"value\":\"0x12345\",\"data\":\"0x307830\"}], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }", accountAddress.Hex())
	response, err = state.api.CallRPC(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, expectedHash.Hex(), response)

	// Personal sign
	request = "{\"method\": \"personal_sign\", \"params\":[\"0x506c65617365207369676e2074686973206d65737361676520746f20636f6e6669726d20796f7572206964656e746974792e\",\"0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7\"], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err = state.api.CallRPC(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, expectedSignature, response)

	// Revoke permissions
	request = "{\"method\": \"wallet_revokePermissions\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
	response, err = state.api.CallRPC(state.ctx, request)
	assert.NoError(t, err)
	assert.Empty(t, response)

	// Check if the account was revoked
	request = fmt.Sprintf("{\"method\": \"eth_sendTransaction\", \"params\":[{\"from\":\"%s\",\"to\":\"0x0200000000000000000000000000000000000000\",\"value\":\"0x12345\",\"data\":\"0x307830\"}], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }", accountAddress.Hex())
	response, err = state.api.CallRPC(state.ctx, request)
	assert.Empty(t, response)
	assert.Error(t, err)
	assert.Equal(t, commands.ErrDAppIsNotPermittedByUser, err)
	assert.Equal(t, true, dAppPermissionRevoked)
}

func TestForwardedRPCs(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	sharedAccount := types.BytesToAddress(types.FromHex("0x3d0ab2a774b74bb1d36f97700315adf962c69fct"))

	testDAppData := signal.ConnectorDApp{
		URL:     "https://app.test.org",
		Name:    "testDAppName",
		IconURL: "https://app.test.icon.org",
	}

	requestJson := "{\"method\": \"eth_blockNumber\", \"params\":[],\"url\":\"https://app.test.org\",\"name\":\"testDAppName\",\"iconUrl\":\"http://testDAppIconUrl\"}"
	request, err := commands.RPCRequestFromJSON(requestJson)
	assert.NoError(t, err)
	request.ChainID = 291
	byteRequest, err := json.Marshal(request)
	assert.NoError(t, err)

	expectedResponse := "0xaa37dc"

	state.rpcClient.EXPECT().CallRaw(string(byteRequest)).Times(0)

	_, err = state.api.CallRPC(state.ctx, requestJson)
	assert.Equal(t, commands.ErrDAppIsNotPermittedByUser, err)

	err = commands.PersistDAppData(state.walletDb, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)

	state.rpcClient.EXPECT().CallRaw(string(byteRequest)).Times(1).Return(fmt.Sprintf(`{"jsonrpc":"2.0","id":37,"result":"%s"}`, expectedResponse))

	response, err := state.api.CallRPC(state.ctx, requestJson)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func TestRequestAccountsAfterPermisasionsRevokeTest(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

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

			err = state.api.RequestAccountsAccepted(commands.RequestAccountsAcceptedArgs{
				RequestID: ev.RequestID,
				Account:   accountAddress,
				ChainID:   0x1,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	for range [10]int{} {
		dAppPermissionRevoked = false
		dAppPermissionGranted = false

		// Request accounts
		request := "{\"method\": \"eth_requestAccounts\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
		response, err := state.api.CallRPC(state.ctx, request)
		assert.NoError(t, err)
		assert.Equal(t, commands.FormatAccountAddressToResponse(accountAddress), response)
		assert.Equal(t, true, dAppPermissionGranted)
		assert.Equal(t, false, dAppPermissionRevoked)

		// Revoke permissions
		request = "{\"method\": \"wallet_revokePermissions\", \"params\": [], \"url\": \"http://testDAppURL123\", \"name\": \"testDAppName\", \"iconUrl\": \"http://testDAppIconUrl\" }"
		_, err = state.api.CallRPC(state.ctx, request)
		assert.NoError(t, err)
		assert.Equal(t, true, dAppPermissionRevoked)
	}
}
