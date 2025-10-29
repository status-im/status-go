package commands

import (
	"context"
	"database/sql"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

type RevokePermissionsCommand struct {
	Db *sql.DB
}

func (c *RevokePermissionsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDApp(c.Db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	err = persistence.DeleteDApp(c.Db, dApp.URL, dApp.ClientID)
	if err != nil {
		return "", err
	}

	signal.SendConnectorDAppPermissionRevoked(signal.ConnectorDApp{
		URL:      request.URL,
		Name:     request.Name,
		IconURL:  request.IconURL,
		ClientID: request.ClientID,
	})

	return nil, nil
}
