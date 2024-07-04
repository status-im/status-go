package commands

import (
	"encoding/json"
	"fmt"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/transactions"
)

type ClientSideHandler struct {
	RpcClient RPCClientInterface
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

func (c *ClientSideHandler) RequestSendTransaction(dApp *DAppData, txArgs *transactions.SendTxArgs) (types.Hash, error) {
	// TODO: implement popup on the UI
	return types.Hash{}, nil
}
