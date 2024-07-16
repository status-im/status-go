package commands

import (
	"database/sql"

	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

type ChainIDCommand struct {
	Db *sql.DB
}

func (c *ChainIDCommand) Execute(request RPCRequest) (string, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.DAppUrl)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	return walletCommon.ChainID(dApp.ChainID).String(), nil
}
