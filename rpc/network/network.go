package network

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"

	persistence "github.com/status-im/status-go/rpc/network/db"
)

//go:generate mockgen -package=mock -source=network.go -destination=mock/network.go
type ManagerInterface interface {
	InitEmbeddedNetworks(networks []params.Network) error

	Upsert(network *params.Network) error
	Delete(chainID uint64) error
	Find(chainID uint64) *params.Network

	Get(onlyEnabled bool) ([]*params.Network, error)
	GetAll() ([]*params.Network, error)
	GetActiveNetworks() ([]*params.Network, error)
	GetCombinedNetworks() ([]*CombinedNetwork, error)
	GetEmbeddedNetworks() []params.Network // Networks that are embedded in the app binary code
	GetTestNetworksEnabled() (bool, error)

	SetUserRpcProviders(chainID uint64, providers []params.RpcProvider) error
	SetEnabled(chainID uint64, enabled bool) error
}

type CombinedNetwork struct {
	Prod *params.Network
	Test *params.Network
}

type Manager struct {
	db                 *sql.DB
	accountsDB         *accounts.Database
	networkPersistence persistence.NetworksPersistenceInterface
	embeddedNetworks   []params.Network

	logger *zap.Logger
}

// NewManager creates a new instance of Manager.
func NewManager(db *sql.DB) *Manager {
	accountsDB, err := accounts.NewDB(db)
	if err != nil {
		return nil
	}

	logger := logutils.ZapLogger().Named("NetworkManager")

	return &Manager{
		db:                 db,
		accountsDB:         accountsDB,
		networkPersistence: persistence.NewNetworksPersistence(db),
		logger:             logger,
	}
}

// Init initializes the nets, merges them with existing ones, and wraps the operation in a transaction.
// We should store the following information in the DB:
// - User's RPC providers
// - Enabled state of the network
// Embedded RPC providers should only be stored in memory
func (nm *Manager) InitEmbeddedNetworks(embeddedNetworks []params.Network) error {
	if embeddedNetworks == nil {
		return nil
	}

	// Update embedded networks
	nm.embeddedNetworks = embeddedNetworks

	// Begin a transaction
	return persistence.ExecuteWithinTransaction(nm.db, func(tx *sql.Tx) error {
		// Create temporary persistence instances with the transaction
		txNetworksPersistence := persistence.NewNetworksPersistence(tx)

		currentNetworks, err := txNetworksPersistence.GetAllNetworks()
		if err != nil {
			return fmt.Errorf("error fetching current networks: %w", err)
		}

		// Create a map for quick access to current networks
		currentNetworskMap := make(map[uint64]params.Network)
		for _, currentNetwork := range currentNetworks {
			currentNetworskMap[currentNetwork.ChainID] = *currentNetwork
		}

		// Keep user's rpc providers and enabled state
		var updatedNetworks []params.Network
		for _, newNetwork := range embeddedNetworks {
			if existingNetwork, exists := currentNetworskMap[newNetwork.ChainID]; exists {
				newNetwork.RpcProviders = networkhelper.GetUserProviders(existingNetwork.RpcProviders)
				newNetwork.Enabled = existingNetwork.Enabled
			} else {
				newNetwork.RpcProviders = networkhelper.GetUserProviders(newNetwork.RpcProviders)
			}
			updatedNetworks = append(updatedNetworks, newNetwork)
		}

		// Use SetNetworks to replace all networks in the database without embedded RPC providers
		err = txNetworksPersistence.SetNetworks(updatedNetworks)
		if err != nil {
			return fmt.Errorf("error setting networks: %w", err)
		}

		return nil
	})
}

// GetEmbeddedProviders returns embedded providers for a given chainID.
func (nm *Manager) getEmbeddedProviders(chainID uint64) []params.RpcProvider {
	for _, network := range nm.embeddedNetworks {
		if network.ChainID == chainID {
			return networkhelper.GetEmbeddedProviders(network.RpcProviders)
		}
	}
	return nil
}

// setEmbeddedProviders adds embedded providers to a network.
func (nm *Manager) setNetworkEmbeddedProviders(network *params.Network) {
	network.RpcProviders = networkhelper.ReplaceEmbeddedProviders(
		network.RpcProviders, nm.getEmbeddedProviders(network.ChainID))
}

func (nm *Manager) setEmbeddedProviders(networks []*params.Network) {
	for _, network := range networks {
		nm.setNetworkEmbeddedProviders(network)
	}
}

// networkWithoutEmbeddedProviders returns a copy of the given network without embedded RPC providers.
func (nm *Manager) networkWithoutEmbeddedProviders(network *params.Network) *params.Network {
	networkCopy := network.DeepCopy()
	networkCopy.RpcProviders = networkhelper.GetUserProviders(network.RpcProviders)
	return &networkCopy
}

// Upsert adds or updates a network, synchronizing RPC providers, wrapped in a transaction.
func (nm *Manager) Upsert(network *params.Network) error {
	return persistence.ExecuteWithinTransaction(nm.db, func(tx *sql.Tx) error {
		txNetworksPersistence := persistence.NewNetworksPersistence(tx)
		err := txNetworksPersistence.UpsertNetwork(nm.networkWithoutEmbeddedProviders(network))
		if err != nil {
			return fmt.Errorf("failed to upsert network: %w", err)
		}
		return nil
	})
}

// Delete removes a network by ChainID, wrapped in a transaction.
func (nm *Manager) Delete(chainID uint64) error {
	return persistence.ExecuteWithinTransaction(nm.db, func(tx *sql.Tx) error {
		txNetworksPersistence := persistence.NewNetworksPersistence(tx)
		err := txNetworksPersistence.DeleteNetwork(chainID)
		if err != nil {
			return fmt.Errorf("failed to delete network: %w", err)
		}
		return nil
	})
}

// SetUserRpcProviders updates user RPC providers, wrapped in a transaction.
func (nm *Manager) SetUserRpcProviders(chainID uint64, userProviders []params.RpcProvider) error {
	rpcPersistence := nm.networkPersistence.GetRpcPersistence()
	return rpcPersistence.SetRpcProviders(chainID, networkhelper.GetUserProviders(userProviders))
}

// SetEnabled updates the enabled status of a network
func (nm *Manager) SetEnabled(chainID uint64, enabled bool) error {
	err := nm.networkPersistence.SetEnabled(chainID, enabled)
	if err != nil {
		return fmt.Errorf("failed to set enabled status: %w", err)
	}
	return nil
}

// Find locates a network by ChainID.
func (nm *Manager) Find(chainID uint64) *params.Network {
	networks, err := nm.networkPersistence.GetNetworkByChainID(chainID)
	if len(networks) != 1 || err != nil {
		nm.logger.Warn("Failed to find network", zap.Uint64("chainID", chainID), zap.Error(err))
		return nil
	}
	result := networks[0]
	nm.setNetworkEmbeddedProviders(result)
	return result
}

// GetAll returns all networks.
func (nm *Manager) GetAll() ([]*params.Network, error) {
	networks, err := nm.networkPersistence.GetAllNetworks()
	if err != nil {
		return nil, err
	}
	nm.setEmbeddedProviders(networks)
	return networks, nil
}

// Get returns networks filtered by the enabled status.
func (nm *Manager) Get(onlyEnabled bool) ([]*params.Network, error) {
	networks, err := nm.networkPersistence.GetNetworks(onlyEnabled, nil)
	if err != nil {
		return nil, err
	}
	nm.setEmbeddedProviders(networks)
	return networks, nil
}

// GetConfiguredNetworks returns the configured networks.
func (nm *Manager) GetEmbeddedNetworks() []params.Network {
	return nm.embeddedNetworks
}

// GetTestNetworksEnabled checks if test networks are enabled.
func (nm *Manager) GetTestNetworksEnabled() (result bool, err error) {
	return nm.accountsDB.GetTestNetworksEnabled()
}

// GetActiveNetworks returns active networks based on the current mode (test/prod).
func (nm *Manager) GetActiveNetworks() ([]*params.Network, error) {
	areTestNetworksEnabled, err := nm.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	networks, err := nm.GetAll()
	if err != nil {
		return nil, err
	}

	var availableNetworks []*params.Network
	for _, network := range networks {
		if network.IsTest == areTestNetworksEnabled {
			availableNetworks = append(availableNetworks, network)
		}
	}

	return availableNetworks, nil
}

func (nm *Manager) GetCombinedNetworks() ([]*CombinedNetwork, error) {
	networks, err := nm.Get(false)
	if err != nil {
		return nil, err
	}

	combinedNetworksMap := make(map[uint64]*CombinedNetwork)
	combinedNetworksSlice := make([]*CombinedNetwork, 0)

	for _, network := range networks {
		combinedNetwork, exists := combinedNetworksMap[network.RelatedChainID]

		if !exists {
			combinedNetwork = &CombinedNetwork{}
			combinedNetworksMap[network.ChainID] = combinedNetwork
			combinedNetworksSlice = append(combinedNetworksSlice, combinedNetwork)
		}

		if network.IsTest {
			combinedNetwork.Test = network
		} else {
			combinedNetwork.Prod = network
		}
	}

	return combinedNetworksSlice, nil
}
