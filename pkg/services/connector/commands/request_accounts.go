package commands

import (
	"context"
	"database/sql"
	"time"

	"github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/signal"
	persistence "github.com/status-im/status-go/pkg/services/connector/database"
)

type RequestAccountsCommand struct {
	clientHandler ClientSideHandlerInterface
	db            *sql.DB
}

func NewRequestAccountsCommand(db *sql.DB, clientHandler ClientSideHandlerInterface) *RequestAccountsCommand {
	return &RequestAccountsCommand{
		db:            db,
		clientHandler: clientHandler,
	}
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

	dApp, err := persistence.SelectDApp(c.db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}

	hasEthAccounts := false
	if dApp != nil {
		hasEthAccounts, err = persistence.PermissionExists(c.db, request.URL, request.ClientID, Method_EthAccounts)
		if err != nil {
			return "", err
		}
	}

	// FIXME: this may have a security issue in case some malicious software tries to fake the origin
	if dApp == nil || !hasEthAccounts {
		dApp, err = shareAndUpsertDApp(c.db, c.clientHandler, request)
		if err != nil {
			return "", err
		}

		// Store eth_accounts permission (EIP-2255)
		createdAt := time.Now().Unix()
		emptyCaveats := []persistence.Caveat{}
		err = persistence.InsertPermission(c.db, dApp.URL, dApp.ClientID, "eth_accounts", emptyCaveats, createdAt)
		if err != nil {
			return "", err
		}

		signal.SendConnectorDAppPermissionGranted(connectorDAppFromRequest(request), dApp.SharedAccount, []uint64{dApp.ChainID})
	}

	return FormatAccountAddressToResponse(dApp.SharedAccount), nil
}
