package commands

import (
	"errors"

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
	NetworkManager NetworkManagerInterface
	ClientHandler  ClientSideHandlerInterface
	AccountsCommand
}

type RawAccountsResponse struct {
	JSONRPC string             `json:"jsonrpc"`
	ID      int                `json:"id"`
	Result  []accounts.Account `json:"result"`
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
	dAppData, err := request.getDAppData()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, dAppData.Origin)
	if err != nil {
		return "", err
	}

	// FIXME: this may have a security issue in case some malicious software tries to fake the origin
	if dApp == nil {
		account, err := c.ClientHandler.RequestShareAccountForDApp(dAppData)
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
