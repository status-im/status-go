package network

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/services/accounts/settingsevent"

	persistence "github.com/status-im/status-go/rpc/network/db"
	"github.com/status-im/status-go/rpc/network/networksevent"
)

//go:generate mockgen -package=mock -source=network.go -destination=mock/network.go

const MaxActiveNetworks = 5

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
	SetActive(chainID uint64, active bool) error
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

	accountFeed     *event.Feed
	settingsFeed    *event.Feed
	networksFeed    *event.Feed
	settingsWatcher *settingsevent.Watcher

	logger *zap.Logger
}

// NewManager creates a new instance of Manager.
func NewManager(db *sql.DB, accountFeed *event.Feed, settingsFeed *event.Feed, networksFeed *event.Feed) *Manager {
	accountsDB, err := accounts.NewDB(db)
	if err != nil {
		return nil
	}

	logger := logutils.ZapLogger().Named("NetworkManager")

	return &Manager{
		db:                 db,
		accountsDB:         accountsDB,
		networkPersistence: persistence.NewNetworksPersistence(db),
		accountFeed:        accountFeed,
		settingsFeed:       settingsFeed,
		networksFeed:       networksFeed,
		logger:             logger,
	}
}

func (nm *Manager) Start() {
	if nm.settingsWatcher == nil {
		settingChangeCb := func(setting settings.SettingField, value interface{}) {
			if setting.Equals(settings.TestNetworksEnabled) {
				nm.onTestNetworksEnabledChanged()
			}
		}
		nm.settingsWatcher = settingsevent.NewWatcher(nm.settingsFeed, settingChangeCb)
		nm.settingsWatcher.Start()
	}
}

func (nm *Manager) Stop() {
	if nm.settingsWatcher != nil {
		nm.settingsWatcher.Stop()
		nm.settingsWatcher = nil
	}
}

func (nm *Manager) onTestNetworksEnabledChanged() {
	nm.notifyActiveNetworksChange()
}

func (nm *Manager) notifyActiveNetworksChange() {
	if nm.networksFeed == nil {
		return
	}

	nm.networksFeed.Send(networksevent.Event{
		Type: networksevent.EventTypeActiveNetworksChanged,
	})
}

// Init initializes the nets, merges them with existing ones, and wraps the operation in a transaction.
// We should store the following information in the DB:
// - User's RPC providers
// - Enabled and Active state of the network
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
			return errors.CreateErrorResponseFromError(fmt.Errorf("error fetching current networks: %w", err))
		}

		// Create a map for quick access to current networks
		currentNetworskMap := make(map[uint64]params.Network)
		for _, currentNetwork := range currentNetworks {
			currentNetworskMap[currentNetwork.ChainID] = *currentNetwork
		}

		// Keep user's rpc providers, enabled and active state
		var updatedNetworks []params.Network
		for _, newNetwork := range embeddedNetworks {
			if existingNetwork, exists := currentNetworskMap[newNetwork.ChainID]; exists {
				newNetwork.RpcProviders = networkhelper.GetUserProviders(existingNetwork.RpcProviders)
				newNetwork.Enabled = existingNetwork.Enabled
				newNetwork.IsActive = existingNetwork.IsActive
			} else {
				newNetwork.RpcProviders = networkhelper.GetUserProviders(newNetwork.RpcProviders)
			}
			updatedNetworks = append(updatedNetworks, newNetwork)
		}

		// Use SetNetworks to replace all networks in the database without embedded RPC providers
		err = txNetworksPersistence.SetNetworks(updatedNetworks)
		if err != nil {
			return errors.CreateErrorResponseFromError(fmt.Errorf("error setting networks: %w", err))
		}

		return nil
	})
}

func (nm *Manager) getEmbeddedNetwork(chainID uint64) *params.Network {
	for _, network := range nm.embeddedNetworks {
		if network.ChainID == chainID {
			return &network
		}
	}
	return nil
}

// Set fields that must be taken from embedded networks.
func (nm *Manager) setEmbeddedFields(networks []*params.Network) {
	for _, network := range networks {
		embeddedNetwork := nm.getEmbeddedNetwork(network.ChainID)
		if embeddedNetwork != nil {
			network.IsDeactivatable = embeddedNetwork.IsDeactivatable
			// Append embedded providers to the user providers
			network.RpcProviders = networkhelper.ReplaceEmbeddedProviders(
				network.RpcProviders, embeddedNetwork.RpcProviders)
		}
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
	// New networks are deactivated by default. They are also always deactivatable.
	network.IsActive = false
	network.IsDeactivatable = true

	return persistence.ExecuteWithinTransaction(nm.db, func(tx *sql.Tx) error {
		txNetworksPersistence := persistence.NewNetworksPersistence(tx)
		err := txNetworksPersistence.UpsertNetwork(nm.networkWithoutEmbeddedProviders(network))
		if err != nil {
			return errors.CreateErrorResponseFromError(fmt.Errorf("failed to upsert network: %w", err))
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
			return errors.CreateErrorResponseFromError(fmt.Errorf("failed to delete network: %w", err))
		}
		return nil
	})
}

// SetUserRpcProviders updates user RPC providers, wrapped in a transaction.
func (nm *Manager) SetUserRpcProviders(chainID uint64, userProviders []params.RpcProvider) error {
	rpcPersistence := nm.networkPersistence.GetRpcPersistence()
	return rpcPersistence.SetRpcProviders(chainID, networkhelper.GetUserProviders(userProviders))
}

// SetActive updates the active status of a network
func (nm *Manager) SetActive(chainID uint64, active bool) error {
	network := nm.Find(chainID)
	if network == nil {
		return ErrUnsupportedChainId
	}

	// If trying to deactivate, check that it's deactivatable
	if !active && !network.IsDeactivatable {
		return ErrNetworkNotDeactivatable
	}

	// If trying to activate, check that we haven't reached the limit for the corresponding mode
	activeNetworks, err := nm.getActiveNetworksForTestMode(network.IsTest)
	if err != nil {
		return errors.CreateErrorResponseFromError(fmt.Errorf("failed to get active networks: %w", err))
	}
	if active && len(activeNetworks) >= MaxActiveNetworks {
		return ErrActiveNetworksLimitReached
	}

	err = nm.networkPersistence.SetActive(chainID, active)
	if err != nil {
		return errors.CreateErrorResponseFromError(fmt.Errorf("failed to persist active status: %w", err))
	}

	nm.notifyActiveNetworksChange()

	return nil
}

// SetEnabled updates the enabled status of a network
func (nm *Manager) SetEnabled(chainID uint64, enabled bool) error {
	err := nm.networkPersistence.SetEnabled(chainID, enabled)
	if err != nil {
		return errors.CreateErrorResponseFromError(fmt.Errorf("failed to set enabled status: %w", err))
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
	nm.setEmbeddedFields([]*params.Network{result})
	return result
}

// GetAll returns all networks.
func (nm *Manager) GetAll() ([]*params.Network, error) {
	networks, err := nm.networkPersistence.GetAllNetworks()
	if err != nil {
		return nil, err
	}
	nm.setEmbeddedFields(networks)
	return networks, nil
}

// Get returns networks filtered by the enabled status.
func (nm *Manager) Get(onlyEnabled bool) ([]*params.Network, error) {
	networks, err := nm.networkPersistence.GetNetworks(onlyEnabled, nil)
	if err != nil {
		return nil, err
	}
	nm.setEmbeddedFields(networks)
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

func (nm *Manager) getActiveNetworksForTestMode(areTestNetworksEnabled bool) ([]*params.Network, error) {
	networks, err := nm.GetAll()
	if err != nil {
		return nil, err
	}

	var availableNetworks []*params.Network
	for _, network := range networks {
		if network.IsActive && network.IsTest == areTestNetworksEnabled {
			availableNetworks = append(availableNetworks, network)
		}
	}

	return availableNetworks, nil
}

// GetActiveNetworks returns active networks based on the current mode (test/prod).
func (nm *Manager) GetActiveNetworks() ([]*params.Network, error) {
	areTestNetworksEnabled, err := nm.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	return nm.getActiveNetworksForTestMode(areTestNetworksEnabled)
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
