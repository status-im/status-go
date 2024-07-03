package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"

	persistence "github.com/status-im/status-go/services/connector/database"
)

type AccountsCommand struct {
	Db *sql.DB
}

type AccountsResponse struct {
	Accounts []string `json:"accounts"`
}

func (c *AccountsCommand) dAppToAccountsResponse(dApp *persistence.DApp) (string, error) {
	response := AccountsResponse{
		Accounts: []string{dApp.SharedAccount},
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	return string(responseJSON), nil
}

func (c *AccountsCommand) Execute(request RPCRequest) (string, error) {
	if err := request.checkDAppData(); err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.Origin)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	return c.dAppToAccountsResponse(dApp)
}
