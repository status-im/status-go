package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/status-im/status-go/eth-node/types"
	persistence "github.com/status-im/status-go/services/connector/database"
)

type AccountsCommand struct {
	Db *sql.DB
}

type AccountsResponse struct {
	Accounts []types.Address `json:"accounts"`
}

func (c *AccountsCommand) dAppToAccountsResponse(dApp *persistence.DApp) (string, error) {
	response := AccountsResponse{
		Accounts: []types.Address{dApp.SharedAccount},
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	return string(responseJSON), nil
}

func (c *AccountsCommand) Execute(request RPCRequest) (string, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}
	dAppData := request.GetDAppData()

	dApp, err := persistence.SelectDAppByUrl(c.Db, dAppData.Origin)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	return c.dAppToAccountsResponse(dApp)
}
