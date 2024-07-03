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
	ErrNoDefaultNetworkAvailable   = errors.New("no default network available")
)

type RequestAccountsCommand struct {
	RpcClient      RPCClientInterface
	NetworkManager NetworkManagerInterface
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

func (c *RequestAccountsCommand) getDefaultChainID() (uint64, error) {
	networks, err := c.NetworkManager.GetActiveNetworks()
	if err != nil {
		return 0, err
	}

	for _, network := range networks {
		if network.Layer == 1 {
			return network.ChainID, nil
		}
	}
	return 0, ErrNoDefaultNetworkAvailable
}

func (c *RequestAccountsCommand) Execute(request RPCRequest) (string, error) {
	if err := request.checkDAppData(); err != nil {
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

		chainID, err := c.getDefaultChainID()
		if err != nil {
			return "", err
		}

		dApp = &persistence.DApp{
			URL:           request.Origin,
			Name:          request.DAppName,
			IconURL:       request.DAppIconUrl,
			SharedAccount: account,
			ChainID:       chainID,
		}

		err = persistence.UpsertDApp(c.Db, dApp)
		if err != nil {
			return "", err
		}
	}

	return c.dAppToAccountsResponse(dApp)
}
