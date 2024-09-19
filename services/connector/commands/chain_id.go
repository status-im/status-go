package commands

import (
	"context"
	"database/sql"

	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/connector/chainutils"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

type ChainIDCommand struct {
	NetworkManager *network.Manager
	Db             *sql.DB
}

func (c *ChainIDCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.URL)
	if err != nil {
		return "", err
	}

	var chainId uint64
	if dApp == nil {
		chainId, err = chainutils.GetDefaultChainID(c.NetworkManager)
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
