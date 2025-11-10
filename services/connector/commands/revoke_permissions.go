package commands

import (
	"context"
	"database/sql"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

type RevokePermissionsCommand struct {
	db *sql.DB
}

func NewRevokePermissionsCommand(db *sql.DB) *RevokePermissionsCommand {
	return &RevokePermissionsCommand{
		db: db,
	}
}

func (c *RevokePermissionsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDApp(c.db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	// Delete the dApp entry (CASCADE will automatically delete permissions)
	err = persistence.DeleteDApp(c.db, dApp.URL, dApp.ClientID)
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
