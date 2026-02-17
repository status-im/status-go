package commands

import (
	"context"
	"database/sql"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

type RevokePermissionsCommand struct {
	db                    *sql.DB
	wcSessionDisconnector WCSessionDisconnector
}

func NewRevokePermissionsCommand(db *sql.DB, wcSessionDisconnector WCSessionDisconnector) *RevokePermissionsCommand {
	return &RevokePermissionsCommand{
		db:                    db,
		wcSessionDisconnector: wcSessionDisconnector,
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

	// If this is a WalletConnect DApp, disconnect all WC sessions first
	if c.wcSessionDisconnector != nil && dApp.ClientID == persistence.WCClientID {
		sessions, err := persistence.SelectWCSessionsByDAppURL(c.db, dApp.URL)
		if err == nil && len(sessions) > 0 {
			for _, session := range sessions {
				_ = c.wcSessionDisconnector.DisconnectSession(ctx, session.Topic)
			}
		}
	}

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
