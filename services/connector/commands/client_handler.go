package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

var (
	WalletResponseMaxInterval = 20 * time.Minute

	ErrWalletResponseTimeout = fmt.Errorf("timeout waiting for wallet response")
)

type ClientSideHandler struct {
	RpcClient RPCClientInterface

	sendTransactionResponseChannel chan ConnectorSendTransactionFinishedArgs
}

func NewClientSideHandler(rpcClient RPCClientInterface) *ClientSideHandler {
	return &ClientSideHandler{
		RpcClient:                      rpcClient,
		sendTransactionResponseChannel: make(chan ConnectorSendTransactionFinishedArgs, 1), // Buffer of 1 to avoid blocking
	}
}

func (c *ClientSideHandler) RequestShareAccountForDApp(dApp *DAppData) (types.Address, error) {
	// NOTE: this is temporary implementation, actual code should invoke popup on the UI

	// TODO: emit a request accounts signal and hang on wallet response
	if false {
		return types.Address{}, ErrAccountsRequestDeniedByUser
	}

	accountsRequest := RPCRequest{
		Method: "accounts_getAccounts",
		Params: []interface{}{},
	}

	requestJSON, err := json.Marshal(accountsRequest)
	if err != nil {
		return types.Address{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	responseJSON := c.RpcClient.CallRaw(string(requestJSON))
	var rawResponse RawAccountsResponse
	err = json.Unmarshal([]byte(responseJSON), &rawResponse)
	if err != nil {
		return types.Address{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(rawResponse.Result) < 1 {
		return types.Address{}, ErrNoAccountsAvailable
	}
	return rawResponse.Result[0].Address, nil
}

func (c *ClientSideHandler) RequestSendTransaction(dApp *DAppData, chainID uint64, txArgs *transactions.SendTxArgs) (types.Hash, error) {
	txArgsJson, err := json.Marshal(txArgs)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to marshal txArgs: %v", err)
	}

	signal.SendConnectorSendTransaction(dApp.Origin, chainID, string(txArgsJson))

	select {
	case response := <-c.sendTransactionResponseChannel:
		if response.Error != nil {
			return types.Hash{}, *response.Error
		}
		return response.Hash, nil
	case <-time.After(WalletResponseMaxInterval):
		return types.Hash{}, ErrWalletResponseTimeout
	}
}

func (c *ClientSideHandler) ConnectorSendTransactionFinished(args ConnectorSendTransactionFinishedArgs) error {
	// Notify RequestSendTransaction with hash or error
	c.sendTransactionResponseChannel <- args
	return nil
}
