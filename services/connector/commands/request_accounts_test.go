package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
)

func TestFailToRequestAccountsWithMissingDAppFields(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &RequestAccountsCommand{Db: db}

	// Missing DApp fields
	request, err := ConstructRPCRequest("eth_requestAccounts", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestRequestAccountsWithSignalTimeout(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	clientHandler := NewClientSideHandler()

	cmd := &RequestAccountsCommand{
		ClientHandler: clientHandler,
		Db:            db,
	}

	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x01})
	assert.NoError(t, err)

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrWalletResponseTimeout, err)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestRequestAccountsAcceptedAndRequestAgain(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	clientHandler := NewClientSideHandler()

	cmd := &RequestAccountsCommand{
		ClientHandler: clientHandler,
		Db:            db,
	}

	request, err := ConstructRPCRequest("eth_requestAccounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	accountAddress := types.Address{0x03}
	dAppPermissionGranted := false

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSendRequestAccounts:
			var ev signal.ConnectorSendRequestAccountsSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = clientHandler.RequestAccountsAccepted(RequestAccountsAcceptedArgs{
				RequestID: ev.RequestID,
				Account:   accountAddress,
				ChainID:   walletCommon.EthereumMainnet,
			})
			assert.NoError(t, err)
		case signal.EventConnectorDAppPermissionGranted:
			dAppPermissionGranted = true
		}
	}))

	expectedResponse := FormatAccountAddressToResponse(accountAddress)
	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

	// Check dApp in the database
	dApp, err := persistence.SelectDAppByUrl(db, request.URL)
	assert.NoError(t, err)
	assert.Equal(t, request.Name, dApp.Name)
	assert.Equal(t, request.IconURL, dApp.IconURL)
	assert.Equal(t, accountAddress, dApp.SharedAccount)
	assert.Equal(t, walletCommon.EthereumMainnet, dApp.ChainID)

	// This should not invoke UI side
	response, err = cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

	assert.True(t, dAppPermissionGranted)
}

func TestRequestAccountsRejected(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	clientHandler := NewClientSideHandler()

	cmd := &RequestAccountsCommand{
		ClientHandler: clientHandler,
		Db:            db,
	}

	request, err := ConstructRPCRequest("eth_requestAccounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSendRequestAccounts:
			var ev signal.ConnectorSendRequestAccountsSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = clientHandler.RequestAccountsRejected(RejectedArgs{
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrRequestAccountsRejectedByUser, err)
}
