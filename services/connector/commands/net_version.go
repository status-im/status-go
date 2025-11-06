package commands

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/connector/chainutils"
	persistence "github.com/status-im/status-go/services/connector/database"
)

type NetVersionCommand struct {
	networkManager *network.Manager
	db             *sql.DB
}

func NewNetVersionCommand(db *sql.DB, networkManager *network.Manager) *NetVersionCommand {
	return &NetVersionCommand{
		db:             db,
		networkManager: networkManager,
	}
}

func (c *NetVersionCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
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
		chainId, err = chainutils.GetDefaultChainID(c.networkManager)
		if err != nil {
			return "", err
		}
	} else {
		chainId = dApp.ChainID
	}

	// net_version returns the network ID as a decimal string
	// (unlike eth_chainId which returns a hex string)
	networkIdStr := strconv.FormatUint(chainId, 10)

	return networkIdStr, nil
}
