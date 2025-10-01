package commands

import (
	"context"
	"database/sql"

	"github.com/status-im/status-go/accounts-management/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

type RequestAccountsCommand struct {
	ClientHandler ClientSideHandlerInterface
	Db            *sql.DB
}

type RawAccountsResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  []types.Account `json:"result"`
}

func (c *RequestAccountsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrlAndClientID(c.Db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}

	// FIXME: this may have a security issue in case some malicious software tries to fake the origin
	if dApp == nil {
		connectorDApp := signal.ConnectorDApp{
			URL:      request.URL,
			Name:     request.Name,
			IconURL:  request.IconURL,
			ClientID: request.ClientID,
		}
		account, chainID, err := c.ClientHandler.RequestShareAccountForDApp(connectorDApp)
		if err != nil {
			return "", err
		}

		dApp = &persistence.DApp{
			URL:           request.URL,
			Name:          request.Name,
			IconURL:       request.IconURL,
			ClientID:      request.ClientID,
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
