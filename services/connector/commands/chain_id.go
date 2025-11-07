package commands

import (
	"context"
	"database/sql"

	"github.com/status-im/status-go/services/connector/chainutils"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

type ChainIDCommand struct {
	defaultChainIDGetter chainutils.DefaultChainIDGetter
	db                   *sql.DB
}

func NewChainIDCommand(db *sql.DB, defaultChainIDGetter chainutils.DefaultChainIDGetter) *ChainIDCommand {
	return &ChainIDCommand{
		db:                   db,
		defaultChainIDGetter: defaultChainIDGetter,
	}
}

func (c *ChainIDCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDApp(c.db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}

	var chainId uint64
	if dApp == nil {
		chainId, err = c.defaultChainIDGetter.GetDefaultChainID()
		if err != nil {
			return "", err
		}
	} else {
		chainId = dApp.ChainID
	}

	chainIdHex, err := chainutils.GetHexChainID(walletCommon.ChainID(chainId).String())
	if err != nil {
		return "", err
	}

	return chainIdHex, nil
}
