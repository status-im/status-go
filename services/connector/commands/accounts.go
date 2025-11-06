package commands

import (
	"context"
	"database/sql"
	"strings"

	"github.com/status-im/status-go/crypto/types"
	persistence "github.com/status-im/status-go/services/connector/database"
)

type AccountsCommand struct {
	db *sql.DB
}

func NewAccountsCommand(db *sql.DB) *AccountsCommand {
	return &AccountsCommand{
		db: db,
	}
}

func FormatAccountAddressToResponse(address types.Address) []string {
	return []string{strings.ToLower(address.Hex())}
}

func (c *AccountsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
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

	return FormatAccountAddressToResponse(dApp.SharedAccount), nil
}
