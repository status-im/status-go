package networks

import (
	"context"

	"github.com/status-im/status-go/params"
)

// API is the JSON-RPC surface of the networks service.
type API struct {
	manager *Manager
}

func NewAPI(manager *Manager) *API {
	return &API{manager: manager}
}

// GetFlatEthereumChains returns all known networks.
func (api *API) GetFlatEthereumChains(ctx context.Context) ([]*params.Network, error) {
	return api.manager.GetAll()
}

// SetChainUserRpcProviders replaces the user-defined RPC providers of a chain.
func (api *API) SetChainUserRpcProviders(ctx context.Context, chainID uint64, rpcProviders []params.RpcProvider) error {
	return api.manager.SetUserRpcProviders(chainID, rpcProviders)
}

// SetChainActive marks a chain available for selection across the application.
// Providers are expected to be accessed only for active chains.
func (api *API) SetChainActive(ctx context.Context, chainID uint64, active bool) error {
	return api.manager.SetActive(chainID, active)
}

// SetChainEnabled marks a chain as taken into account when displaying balances,
// collectibles, activity, etc.
func (api *API) SetChainEnabled(ctx context.Context, chainID uint64, enabled bool) error {
	return api.manager.SetEnabled(chainID, enabled)
}

// AddEthereumChain adds or updates a network.
//
// Deprecated: custom networks are not currently supported. Change settings
// using specific API functions.
func (api *API) AddEthereumChain(ctx context.Context, network params.Network) error {
	return api.manager.Upsert(&network)
}

// DeleteEthereumChain removes a network.
//
// Deprecated: custom networks are not currently supported. Change settings
// using specific API functions.
func (api *API) DeleteEthereumChain(ctx context.Context, chainID uint64) error {
	return api.manager.Delete(chainID)
}

// GetEthereumChains returns networks paired by prod and test chain.
//
// Deprecated: combined networks are not used anymore, use
// GetFlatEthereumChains instead.
func (api *API) GetEthereumChains(ctx context.Context) ([]*CombinedNetwork, error) {
	return api.manager.GetCombinedNetworks()
}
