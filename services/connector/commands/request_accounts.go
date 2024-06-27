package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/status-im/status-go/multiaccounts/accounts"
	persistence "github.com/status-im/status-go/services/connector/database"
)

// errors
var (
	ErrAccountsRequestDeniedByUser = errors.New("accounts request denied by user")
	ErrNoAccountsAvailable         = errors.New("no accounts available")
)

type RequestAccountsCommand struct {
	RpcClient RPCClientInterface
	AccountsCommand
}

type RawAccountsResponse struct {
	JSONRPC string             `json:"jsonrpc"`
	ID      int                `json:"id"`
	Result  []accounts.Account `json:"result"`
}

func (c *RequestAccountsCommand) requestAccountForDApp() (string, error) {
	// NOTE: this is temporary implementation, actual code should invoke popup on the UI

	// TODO: emit a request accounts signal and hang on wallet response
	if false {
		return "", ErrAccountsRequestDeniedByUser
	}

	accountsRequest := RPCRequest{
		Method: "accounts_getAccounts",
		Params: []interface{}{},
	}

	requestJSON, err := json.Marshal(accountsRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	responseJSON := c.RpcClient.CallRaw(string(requestJSON))
	var rawResponse RawAccountsResponse
	err = json.Unmarshal([]byte(responseJSON), &rawResponse)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(rawResponse.Result) < 1 {
		return "", ErrNoAccountsAvailable
	}
	return rawResponse.Result[0].Address.Hex(), nil
}

func (c *RequestAccountsCommand) Execute(request RPCRequest) (string, error) {
	if err := c.checkDAppData(request); err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.Origin)
	if err != nil {
		return "", err
	}

	// FIXME: this may have a security issue in case some malicious software tries to fake the origin
	if dApp == nil {
		account, err := c.requestAccountForDApp()
		if err != nil {
			return "", err
		}

		dApp = &persistence.DApp{
			URL:           request.Origin,
			Name:          request.DAppName,
			IconURL:       request.DAppIconUrl,
			SharedAccount: account,
		}

		err = persistence.UpsertDApp(c.Db, dApp)
		if err != nil {
			return "", err
		}
	}

	return c.dAppToAccountsResponse(dApp)
}
