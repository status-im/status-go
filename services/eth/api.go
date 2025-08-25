package eth

import (
	"context"

	accounts "github.com/status-im/status-go/accounts-management"

	"github.com/status-im/status-go/rpc"
)

type API struct {
	client          *rpc.Client
	accountsManager *accounts.AccountsManager
}

func NewAPI(client *rpc.Client, accountsManager *accounts.AccountsManager) *API {
	return &API{
		client:          client,
		accountsManager: accountsManager,
	}
}

// EstimateGas estimates gas for a given payload on a chain.
func (a *API) EstimateGas(ctx context.Context, chainID uint64, payload any) (result string, err error) {
	err = a.client.CallContextIgnoringLocalHandlers(ctx, &result, chainID, "eth_estimateGas", payload)
	return
}
