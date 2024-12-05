package network

import (
	"database/sql"
	"fmt"

	"github.com/status-im/status-go/multiaccounts/accounts"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	persistence "github.com/status-im/status-go/rpc/network/db"
)

type CombinedNetwork struct {
	Prod *params.Network
	Test *params.Network
}

type ManagerInterface interface {
	InitEmbeddedNetworks(networks []params.Network) error

	Upsert(network *params.Network) error
	Delete(chainID uint64) error
	Find(chainID uint64) *params.Network

	Get(onlyEnabled bool) ([]*params.Network, error)
	GetAll() ([]*params.Network, error)
	GetActiveNetworks() ([]*params.Network, error)
	GetCombinedNetworks() ([]*CombinedNetwork, error)
	GetConfiguredNetworks() []params.Network
	GetTestNetworksEnabled() (bool, error)

	SetUserRpcProviders(chainID uint64, providers []params.RpcProvider) error
}

type Manager struct {
	db                 *sql.DB
	accountsDB         *accounts.Database
	networkPersistence persistence.NetworksPersistenceInterface
	configuredNetworks []params.Network
}

// NewManager creates a new instance of Manager.
func NewManager(db *sql.DB) *Manager {
	accountsDB, err := accounts.NewDB(db)
	if err != nil {
		return nil
	}
	return &Manager{
		db:                 db,
		accountsDB:         accountsDB,
		networkPersistence: persistence.NewNetworksPersistence(db),
	}
}

// Init initializes the networks, merging with existing ones and wrapping the operation in a transaction.
func (nm *Manager) InitEmbeddedNetworks(networks []params.Network) error {
	if networks == nil {
		return nil
	}

	// Begin a transaction
	return persistence.ExecuteWithinTransaction(nm.db, func(tx *sql.Tx) error {
		// Create temporary persistence instances with the transaction
		txNetworksPersistence := persistence.NewNetworksPersistence(tx)

		currentNetworks, err := txNetworksPersistence.GetAllNetworks()
		if err != nil {
			return fmt.Errorf("error fetching current networks: %w", err)
		}

		// Create a map for quick access to current networks
		currentNetworkMap := make(map[uint64]params.Network)
		for _, currentNetwork := range currentNetworks {
			currentNetworkMap[currentNetwork.ChainID] = *currentNetwork
		}

		// Process new networks
		var updatedNetworks []params.Network
		for _, newNetwork := range networks {
			if existingNetwork, exists := currentNetworkMap[newNetwork.ChainID]; exists {
				// If network already exists, merge providers
				newNetwork.RpcProviders = networkhelper.ReplaceEmbeddedProviders(existingNetwork.RpcProviders, newNetwork.RpcProviders)
			}
			updatedNetworks = append(updatedNetworks, newNetwork)
		}

		// Use SetNetworks to replace all networks in the database
		err = txNetworksPersistence.SetNetworks(updatedNetworks)
		if err != nil {
			return fmt.Errorf("error setting networks: %w", err)
		}

		// Update configured networks
		nm.configuredNetworks = networks

		return nil
	})
}

// Upsert adds or updates a network, synchronizing RPC providers, wrapped in a transaction.
func (nm *Manager) Upsert(network *params.Network) error {
	return persistence.ExecuteWithinTransaction(nm.db, func(tx *sql.Tx) error {
		txNetworksPersistence := persistence.NewNetworksPersistence(tx)
		err := txNetworksPersistence.UpsertNetwork(network)
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
	return persistence.ExecuteWithinTransaction(nm.db, func(tx *sql.Tx) error {
		// Create temporary persistence instances with the transaction
		txRpcPersistence := persistence.NewRpcProvidersPersistence(tx)

		// Get all providers using the transactional RPC persistence
		allProviders, err := txRpcPersistence.GetRpcProviders(chainID)
		if err != nil {
			return fmt.Errorf("failed to get all providers: %w", err)
		}

		// Replace user providers
		providers := networkhelper.ReplaceUserProviders(allProviders, userProviders)

		// Set RPC providers using the transactional RPC persistence
		err = txRpcPersistence.SetRpcProviders(chainID, providers)
		if err != nil {
			return fmt.Errorf("failed to set RPC providers: %w", err)
		}

		return nil
	})
}

// Find locates a network by ChainID.
func (nm *Manager) Find(chainID uint64) *params.Network {
	networks, err := nm.networkPersistence.GetNetworkByChainID(chainID)
	if len(networks) != 1 || err != nil {
		return nil
	}
	return networks[0]
}

// GetAll returns all networks.
func (nm *Manager) GetAll() ([]*params.Network, error) {
	return nm.networkPersistence.GetAllNetworks()
}

// Get returns networks filtered by the enabled status.
func (nm *Manager) Get(onlyEnabled bool) ([]*params.Network, error) {
	return nm.networkPersistence.GetNetworks(onlyEnabled, nil)
}

// GetConfiguredNetworks returns the configured networks.
func (nm *Manager) GetConfiguredNetworks() []params.Network {
	return nm.configuredNetworks
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
