package commands

import (
	"errors"
	"slices"

	"github.com/status-im/status-go/multiaccounts/accounts"
	persistence "github.com/status-im/status-go/services/connector/database"
)

// errors
var (
	ErrAccountsRequestDeniedByUser = errors.New("accounts request denied by user")
	ErrNoAccountsAvailable         = errors.New("no accounts available")
	ErrNotSupportedNetwork         = errors.New("not supported network")
)

type RequestAccountsCommand struct {
	NetworkManager NetworkManagerInterface
	ClientHandler  ClientSideHandlerInterface
	AccountsCommand
}

type RawAccountsResponse struct {
	JSONRPC string             `json:"jsonrpc"`
	ID      int                `json:"id"`
	Result  []accounts.Account `json:"result"`
}

func (c *RequestAccountsCommand) getActiveChainIDs() ([]uint64, error) {
	networks, err := c.NetworkManager.GetActiveNetworks()
	if err != nil {
		return []uint64{}, err
	}

	chainIDs := make([]uint64, len(networks))
	for i, network := range networks {
		chainIDs[i] = network.ChainID
	}
	return chainIDs, nil
}

func (c *RequestAccountsCommand) Execute(request RPCRequest) (string, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}
	dAppData := request.GetDAppData()

	dApp, err := persistence.SelectDAppByUrl(c.Db, dAppData.Origin)
	if err != nil {
		return "", err
	}

	// FIXME: this may have a security issue in case some malicious software tries to fake the origin
	if dApp == nil {
		chainIDs, err := c.getActiveChainIDs()
		if err != nil {
			return "", err
		}

		account, chainID, err := c.ClientHandler.RequestShareAccountForDApp(dAppData, chainIDs)
		if err != nil {
			return "", err
		}

		if !slices.Contains(chainIDs, chainID) {
			return "", ErrNotSupportedNetwork
		}

		dApp = &persistence.DApp{
			URL:           request.Origin,
			Name:          request.DAppName,
			IconURL:       request.DAppIconUrl,
			SharedAccount: account,
			ChainID:       chainID,
		}

		err = persistence.UpsertDApp(c.Db, dApp)
		if err != nil {
			return "", err
		}
	}

	return c.dAppToAccountsResponse(dApp)
}
