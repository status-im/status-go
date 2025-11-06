package commands

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/status-im/status-go/services/connector/chainutils"
	persistence "github.com/status-im/status-go/services/connector/database"
)

type NetVersionCommand struct {
	defaultChainIDGetter chainutils.DefaultChainIDGetter
	db                   *sql.DB
}

func NewNetVersionCommand(db *sql.DB, defaultChainIDGetter chainutils.DefaultChainIDGetter) *NetVersionCommand {
	return &NetVersionCommand{
		db:                   db,
		defaultChainIDGetter: defaultChainIDGetter,
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
		chainId, err = c.defaultChainIDGetter.GetDefaultChainID()
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
