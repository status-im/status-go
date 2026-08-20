package commands

import (
	"context"
	"database/sql"

	"github.com/status-im/status-go/internal/signal"
	persistence "github.com/status-im/status-go/pkg/services/connector/database"
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

// parseWalletRevokeParams mirrors EIP-2255-style params: [{ "eth_accounts": {} }, ...] or one or more maps
// with capability keys. Empty params slice means full disconnect (legacy behaviour). All-empty maps
// (e.g. [{}] or [{}, {}]) also mean full revoke. Any nil or non-map element is invalid.
func parseWalletRevokeParams(params []interface{}) (fullRevoke bool, capabilities []string, err error) {
	if len(params) == 0 {
		return true, nil, nil
	}
	for _, p := range params {
		if p == nil {
			return false, nil, ErrInvalidParamType
		}
		m, ok := p.(map[string]interface{})
		if !ok {
			return false, nil, ErrInvalidParamType
		}
		for cap := range m {
			capabilities = append(capabilities, cap)
		}
	}
	if len(capabilities) == 0 {
		return true, nil, nil
	}
	return false, capabilities, nil
}

func (c *RevokePermissionsCommand) fullRevoke(ctx context.Context, request RPCRequest, dApp *persistence.DApp) (interface{}, error) {
	if c.wcSessionDisconnector != nil && dApp.ClientID == persistence.WCClientID {
		sessions, err := persistence.SelectWCSessionsByDAppURL(c.db, dApp.URL)
		if err == nil && len(sessions) > 0 {
			for _, session := range sessions {
				_ = c.wcSessionDisconnector.DisconnectSession(ctx, session.Topic)
			}
		}
	}

	err := persistence.DeleteDApp(c.db, dApp.URL, dApp.ClientID)
	if err != nil {
		return "", err
	}

	signal.SendConnectorDAppPermissionRevoked(connectorDAppFromRequest(request))

	return nil, nil
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

	// WalletConnect sessions are keyed to the whole dApp row; keep full teardown only.
	if dApp.ClientID == persistence.WCClientID {
		return c.fullRevoke(ctx, request, dApp)
	}

	fullRevoke, caps, err := parseWalletRevokeParams(request.Params)
	if err != nil {
		return "", err
	}
	if fullRevoke {
		return c.fullRevoke(ctx, request, dApp)
	}

	for _, cap := range caps {
		if err := persistence.DeletePermission(c.db, dApp.URL, dApp.ClientID, cap); err != nil {
			return "", err
		}
	}

	// Intentionally keep the connector_dapps row when params name specific capabilities (e.g. Velora's
	// wallet_revokePermissions [{"eth_accounts":{}}]) so forwarded chain RPCs still work; use empty
	// params for a full disconnect (DeleteDApp + ConnectorDAppPermissionRevoked).
	return nil, nil
}
