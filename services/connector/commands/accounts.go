package commands

import (
	"context"
	"database/sql"
	"strings"

	"github.com/status-im/status-go/eth-node/types"
	persistence "github.com/status-im/status-go/services/connector/database"
)

type AccountsCommand struct {
	Db *sql.DB
}

func FormatAccountAddressToResponse(address types.Address) []string {
	return []string{strings.ToLower(address.Hex())}
}

func (c *AccountsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
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

	return FormatAccountAddressToResponse(dApp.SharedAccount), nil
}
