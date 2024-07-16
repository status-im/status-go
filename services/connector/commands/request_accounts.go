package commands

import (
	"errors"

	"github.com/status-im/status-go/multiaccounts/accounts"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

// errors
var (
	ErrAccountsRequestDeniedByUser = errors.New("accounts request denied by user")
	ErrNoAccountsAvailable         = errors.New("no accounts available")
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

func (c *RequestAccountsCommand) Execute(request RPCRequest) (string, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.DAppUrl)
	if err != nil {
		return "", err
	}

	// FIXME: this may have a security issue in case some malicious software tries to fake the origin
	if dApp == nil {
		account, chainID, err := c.ClientHandler.RequestShareAccountForDApp(signal.ConnectorDApp{
			URL:     request.DAppUrl,
			Name:    request.DAppName,
			IconURL: request.DAppIconUrl,
		})
		if err != nil {
			return "", err
		}

		dApp = &persistence.DApp{
			URL:           request.DAppUrl,
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
