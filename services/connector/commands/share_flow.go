package commands

import (
	"database/sql"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

func connectorDAppFromRequest(request RPCRequest) signal.ConnectorDApp {
	return signal.ConnectorDApp{
		URL:      request.URL,
		Name:     request.Name,
		IconURL:  request.IconURL,
		ClientID: request.ClientID,
	}
}

// shareAndUpsertDApp runs the connector share-account UI flow and persists the dApp row.
// It does not insert EIP-2255 permission rows or emit ConnectorDAppPermissionGranted; callers do that.
// Begin/EndPendingShareAccount bracket the UI wait and UpsertDApp so parallel wallet_requestPermissions
// does not observe an empty DB row after the wait unblocks.
func shareAndUpsertDApp(db *sql.DB, h ClientSideHandlerInterface, request RPCRequest) (*persistence.DApp, error) {
	h.BeginPendingShareAccount(request.URL, request.ClientID)
	defer h.EndPendingShareAccount(request.URL, request.ClientID)

	connectorDApp := connectorDAppFromRequest(request)
	account, chainID, err := h.RequestShareAccountForDApp(connectorDApp)
	if err != nil {
		return nil, err
	}

	dApp := &persistence.DApp{
		URL:           request.URL,
		Name:          request.Name,
		IconURL:       request.IconURL,
		ClientID:      request.ClientID,
		SharedAccount: account,
		ChainID:       chainID,
	}

	if err := persistence.UpsertDApp(db, dApp); err != nil {
		return nil, err
	}
	return dApp, nil
}
