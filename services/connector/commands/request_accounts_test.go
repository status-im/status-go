package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
)

func TestFailToRequestAccountsWithMissingDAppFields(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &RequestAccountsCommand{AccountsCommand: AccountsCommand{Db: db}}

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

	nm := NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{})

	cmd := &RequestAccountsCommand{
		ClientHandler:   clientHandler,
		AccountsCommand: AccountsCommand{Db: db},
		NetworkManager:  &nm,
	}

	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x01})
	assert.NoError(t, err)

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrWalletResponseTimeout, err)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestRequestAccountsTwoTimes(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	nm := NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: walletCommon.OptimismMainnet,
			Layer:   2,
		},
		{
			ChainID: walletCommon.EthereumMainnet,
			Layer:   1,
		},
	})

	clientHandler := NewClientSideHandler()

	cmd := &RequestAccountsCommand{
		ClientHandler:   clientHandler,
		NetworkManager:  &nm,
		AccountsCommand: AccountsCommand{Db: db},
	}

	request, err := ConstructRPCRequest("eth_requestAccounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	accountAddress := types.Address{0x03}

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
		}
	}))

	response, err := cmd.Execute(request)
	assert.NoError(t, err)

	result := &AccountsResponse{}
	err = json.Unmarshal([]byte(response), result)

	assert.NoError(t, err)
	assert.Len(t, result.Accounts, 1)
	assert.Equal(t, accountAddress, result.Accounts[0])

	// Check dApp in the database
	dApp, err := persistence.SelectDAppByUrl(db, request.DAppUrl)
	assert.NoError(t, err)
	assert.Equal(t, request.DAppName, dApp.Name)
	assert.Equal(t, request.DAppIconUrl, dApp.IconURL)
	assert.Equal(t, accountAddress, dApp.SharedAccount)
	assert.Equal(t, walletCommon.EthereumMainnet, dApp.ChainID)

	// This should not invoke UI side
	response, err = cmd.Execute(request)
	assert.NoError(t, err)

	err = json.Unmarshal([]byte(response), result)

	assert.NoError(t, err)
	assert.Len(t, result.Accounts, 1)
	assert.Equal(t, accountAddress, result.Accounts[0])
}
