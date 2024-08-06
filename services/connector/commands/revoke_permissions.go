package commands

import (
	"database/sql"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

type RevokePermissionsCommand struct {
	Db *sql.DB
}

func (c *RevokePermissionsCommand) Execute(request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.URL)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	err = persistence.DeleteDApp(c.Db, dApp.URL)
	if err != nil {
		return "", err
	}

	signal.SendConnectorDAppPermissionRevoked(signal.ConnectorDApp{
		URL:     request.URL,
		Name:    request.Name,
		IconURL: request.IconURL,
	})

	return nil, nil
}
