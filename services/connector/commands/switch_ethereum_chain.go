package commands

import (
	"database/sql"
	"errors"
	"slices"

	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

// errors
var (
	ErrNoActiveNetworks   = errors.New("no active networks")
	ErrUnsupportedNetwork = errors.New("unsupported network")
)

type SwitchEthereumChainCommand struct {
	NetworkManager NetworkManagerInterface
	Db             *sql.DB
}

func (c *SwitchEthereumChainCommand) getSupportedChainIDs() ([]uint64, error) {
	activeNetworks, err := c.NetworkManager.GetActiveNetworks()
	if err != nil {
		return nil, err
	}

	if len(activeNetworks) < 1 {
		return nil, ErrNoActiveNetworks
	}

	chainIDs := make([]uint64, len(activeNetworks))
	for i, network := range activeNetworks {
		chainIDs[i] = network.ChainID
	}

	return chainIDs, nil
}

func (c *SwitchEthereumChainCommand) Execute(request RPCRequest) (string, error) {
	if err := request.checkDAppData(); err != nil {
		return "", err
	}

	requestedChainID, err := request.getChainID()
	if err != nil {
		return "", err
	}

	chainIDs, err := c.getSupportedChainIDs()
	if err != nil {
		return "", err
	}

	if !slices.Contains(chainIDs, requestedChainID) {
		return "", ErrUnsupportedNetwork
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.Origin)
	if err != nil {
		return "", err
	}

	dApp.ChainID = requestedChainID

	err = persistence.UpsertDApp(c.Db, dApp)
	if err != nil {
		return "", err
	}

	return walletCommon.ChainID(dApp.ChainID).String(), nil
}
