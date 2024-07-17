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
	ErrNoActiveNetworks     = errors.New("no active networks")
	ErrUnsupportedNetwork   = errors.New("unsupported network")
	ErrNoChainIDParamsFound = errors.New("no chain id in params found")
)

type SwitchEthereumChainCommand struct {
	NetworkManager NetworkManagerInterface
	Db             *sql.DB
}

func (r *RPCRequest) getChainID() (uint64, error) {
	if r.Params == nil || len(r.Params) == 0 {
		return 0, ErrEmptyRPCParams
	}

	switch v := r.Params[0].(type) {
	case float64:
		return uint64(v), nil
	case int:
		return uint64(v), nil
	case int64:
		return uint64(v), nil
	case uint64:
		return v, nil
	default:
		return 0, ErrNoChainIDParamsFound
	}
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
	err := request.Validate()
	if err != nil {
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

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.URL)
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
