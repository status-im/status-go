package commands

import (
	"database/sql"
	"errors"
	"slices"
	"strconv"

	"github.com/status-im/status-go/services/connector/chainutils"
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

func hexStringToUint64(s string) (uint64, error) {
	if len(s) > 2 && s[:2] == "0x" {
		value, err := strconv.ParseUint(s[2:], 16, 64)
		if err != nil {
			return 0, err
		}
		return value, nil
	}
	return 0, ErrUnsupportedNetwork
}

func (r *RPCRequest) getChainID() (uint64, error) {
	if r.Params == nil || len(r.Params) == 0 {
		return 0, ErrEmptyRPCParams
	}

	chainIds := r.Params[0].(map[string]interface{})

	for _, chainId := range chainIds {
		return hexStringToUint64(chainId.(string))
	}

	return 0, nil
}

func (c *SwitchEthereumChainCommand) getSupportedChainIDs() ([]uint64, error) {
	return chainutils.GetSupportedChainIDs(c.NetworkManager)
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

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	dApp.ChainID = requestedChainID

	err = persistence.UpsertDApp(c.Db, dApp)
	if err != nil {
		return "", err
	}

	chainId, err := chainutils.GetHexChainID(walletCommon.ChainID(dApp.ChainID).String())
	if err != nil {
		return "", err
	}

	return chainId, nil
}
