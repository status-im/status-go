package commands

import (
	"context"
	"database/sql"
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
	ClientHandler ClientSideHandlerInterface
	Db            *sql.DB
}

type RawAccountsResponse struct {
	JSONRPC string             `json:"jsonrpc"`
	ID      int                `json:"id"`
	Result  []accounts.Account `json:"result"`
}

func (c *RequestAccountsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.URL)
	if err != nil {
		return "", err
	}

	// FIXME: this may have a security issue in case some malicious software tries to fake the origin
	if dApp == nil {
		connectorDApp := signal.ConnectorDApp{
			URL:     request.URL,
			Name:    request.Name,
			IconURL: request.IconURL,
		}
		account, chainID, err := c.ClientHandler.RequestShareAccountForDApp(connectorDApp)
		if err != nil {
			return "", err
		}

		dApp = &persistence.DApp{
			URL:           request.URL,
			Name:          request.Name,
			IconURL:       request.IconURL,
			SharedAccount: account,
			ChainID:       chainID,
		}

		err = persistence.UpsertDApp(c.Db, dApp)
		if err != nil {
			return "", err
		}
		signal.SendConnectorDAppPermissionGranted(connectorDApp, account, []uint64{chainID})
	}

	return FormatAccountAddressToResponse(dApp.SharedAccount), nil
}
