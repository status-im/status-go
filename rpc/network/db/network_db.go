package db

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/sqlite"
)

// NetworksPersistenceInterface describes the interface for managing networks and providers.
type NetworksPersistenceInterface interface {
	// GetNetworks returns networks based on filters.
	GetNetworks(onlyEnabled bool, chainID *uint64) ([]*params.Network, error)
	// GetAllNetworks returns all networks.
	GetAllNetworks() ([]*params.Network, error)
	// GetNetworkByChainID returns a network by ChainID.
	GetNetworkByChainID(chainID uint64) ([]*params.Network, error)
	// GetEnabledNetworks returns enabled networks.
	GetEnabledNetworks() ([]*params.Network, error)
	// SetNetworks replaces all networks with new ones and their providers.
	SetNetworks(networks []params.Network) error
	// UpsertNetwork adds or updates a network and its providers.
	UpsertNetwork(network *params.Network) error
	// DeleteNetwork deletes a network by ChainID and its providers.
	DeleteNetwork(chainID uint64) error
	// DeleteAllNetworks deletes all networks and their providers.
	DeleteAllNetworks() error

	GetRpcPersistence() RpcProvidersPersistenceInterface
	SetActive(chainID uint64, active bool) error
	SetEnabled(chainID uint64, enabled bool) error
}

// NetworksPersistence manages networks and their providers.
type NetworksPersistence struct {
	db             sqlite.StatementExecutor
	rpcPersistence RpcProvidersPersistenceInterface
	validator      *validator.Validate
}

// NewNetworksPersistence creates a new instance of NetworksPersistence.
func NewNetworksPersistence(db sqlite.StatementExecutor) *NetworksPersistence {
	return &NetworksPersistence{
		db:             db,
		rpcPersistence: NewRpcProvidersPersistence(db),
		validator:      validator.New(),
	}
}

// GetRpcPersistence returns an instance of RpcProvidersPersistenceInterface.
func (n *NetworksPersistence) GetRpcPersistence() RpcProvidersPersistenceInterface {
	return n.rpcPersistence
}

// getNetworksWithoutProviders returns networks based on filters without populating providers.
func (n *NetworksPersistence) getNetworksWithoutProviders(onlyEnabled bool, chainID *uint64) ([]*params.Network, error) {
	q := sq.Select(
		"chain_id", "chain_name", "rpc_url", "fallback_url",
		"block_explorer_url", "icon_url", "native_currency_name", "native_currency_symbol", "native_currency_decimals",
		"is_test", "layer", "enabled", "chain_color", "short_name", "related_chain_id", "is_active", "is_deactivatable",
	).
		From("networks").
		OrderBy("chain_id ASC")

	if onlyEnabled {
		q = q.Where(sq.Eq{"enabled": true})
	}
	if chainID != nil {
		q = q.Where(sq.Eq{"chain_id": *chainID})
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := n.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	result := make([]*params.Network, 0, 10)
	for rows.Next() {
		network := &params.Network{}
		var relatedChainID sql.NullInt64
		err := rows.Scan(
			&network.ChainID, &network.ChainName, &network.RPCURL, &network.FallbackURL,
			&network.BlockExplorerURL, &network.IconURL, &network.NativeCurrencyName, &network.NativeCurrencySymbol,
			&network.NativeCurrencyDecimals, &network.IsTest, &network.Layer, &network.Enabled, &network.ChainColor,
			&network.ShortName, &relatedChainID, &network.IsActive, &network.IsDeactivatable,
		)
		if err != nil {
			return nil, err
		}

		// Convert sql.NullInt64 to uint64 only if it's valid
		if relatedChainID.Valid {
			network.RelatedChainID = uint64(relatedChainID.Int64)
		}

		result = append(result, network)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	return result, nil
}

// populateNetworkProviders populates RPC providers for the given networks.
func (n *NetworksPersistence) populateNetworkProviders(networks []*params.Network) error {
	for i := range networks {
		providers, err := n.rpcPersistence.GetRpcProviders(networks[i].ChainID)
		if err != nil {
			return fmt.Errorf("failed to fetch RPC providers for chain_id %d: %w", networks[i].ChainID, err)
		}
		networks[i].RpcProviders = providers
		FillDeprecatedURLs(networks[i], providers)
	}
	return nil
}

// GetNetworks returns networks based on filters.
func (n *NetworksPersistence) GetNetworks(onlyEnabled bool, chainID *uint64) ([]*params.Network, error) {
	networks, err := n.getNetworksWithoutProviders(onlyEnabled, chainID)
	if err != nil {
		return nil, err
	}

	if err := n.populateNetworkProviders(networks); err != nil {
		return nil, err
	}

	return networks, nil
}

// GetAllNetworks returns all networks.
func (n *NetworksPersistence) GetAllNetworks() ([]*params.Network, error) {
	return n.GetNetworks(false, nil)
}

// GetNetworkByChainID returns a network by ChainID.
func (n *NetworksPersistence) GetNetworkByChainID(chainID uint64) ([]*params.Network, error) {
	return n.GetNetworks(false, &chainID)
}

// GetEnabledNetworks returns enabled networks.
func (n *NetworksPersistence) GetEnabledNetworks() ([]*params.Network, error) {
	return n.GetNetworks(true, nil)
}

// SetNetworks replaces all networks with new ones and their providers.
// Note: Transaction management should be handled by the caller.
func (n *NetworksPersistence) SetNetworks(networks []params.Network) error {
	// Delete all networks and their providers
	err := n.DeleteAllNetworks()
	if err != nil {
		return err
	}

	// Upsert networks and their providers
	for i := range networks {
		err := n.UpsertNetwork(&networks[i])
		if err != nil {
			return fmt.Errorf("failed to upsert network with chain_id %d: %w", networks[i].ChainID, err)
		}
	}

	return nil
}

// DeleteAllNetworks deletes all networks and their RPC providers.
// Note: Transaction management should be handled by the caller.
func (n *NetworksPersistence) DeleteAllNetworks() error {
	// Delete all RPC providers
	err := n.rpcPersistence.DeleteAllRpcProviders()
	if err != nil {
		return fmt.Errorf("failed to delete all RPC providers: %w", err)
	}

	// Delete all networks
	q := sq.Delete("networks")

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = n.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute delete query: %w", err)
	}

	return nil
}

// UpsertNetwork adds or updates a network and its providers.
// Note: Transaction management should be handled by the caller.
func (n *NetworksPersistence) UpsertNetwork(network *params.Network) error {
	// Validate the network
	if err := n.validator.Struct(network); err != nil {
		return fmt.Errorf("network validation failed: %w", err)
	}

	// Upsert the network
	err := n.upsertNetwork(network)
	if err != nil {
		return fmt.Errorf("failed to upsert network for chain_id %d: %w", network.ChainID, err)
	}

	// Set the RPC providers
	err = n.rpcPersistence.SetRpcProviders(network.ChainID, network.RpcProviders)
	if err != nil {
		return fmt.Errorf("failed to set RPC providers for chain_id %d: %w", network.ChainID, err)
	}

	return nil
}

// upsertNetwork handles the logic for inserting or updating a network record.
func (n *NetworksPersistence) upsertNetwork(network *params.Network) error {
	q := sq.Insert("networks").
		Columns(
			"chain_id", "chain_name", "rpc_url", "original_rpc_url", "fallback_url", "original_fallback_url",
			"block_explorer_url", "icon_url", "native_currency_name", "native_currency_symbol", "native_currency_decimals",
			"is_test", "layer", "enabled", "chain_color", "short_name", "related_chain_id", "is_active", "is_deactivatable",
		).
		Values(
			network.ChainID, network.ChainName, network.RPCURL, network.OriginalRPCURL, network.FallbackURL, network.OriginalFallbackURL,
			network.BlockExplorerURL, network.IconURL, network.NativeCurrencyName, network.NativeCurrencySymbol, network.NativeCurrencyDecimals,
			network.IsTest, network.Layer, network.Enabled, network.ChainColor, network.ShortName, network.RelatedChainID, network.IsActive, network.IsDeactivatable,
		).
		Suffix("ON CONFLICT(chain_id) DO UPDATE SET "+
			"chain_name = excluded.chain_name, rpc_url = excluded.rpc_url, original_rpc_url = excluded.original_rpc_url, "+
			"fallback_url = excluded.fallback_url, original_fallback_url = excluded.original_fallback_url, "+
			"block_explorer_url = excluded.block_explorer_url, icon_url = excluded.icon_url, "+
			"native_currency_name = excluded.native_currency_name, native_currency_symbol = excluded.native_currency_symbol, "+
			"native_currency_decimals = excluded.native_currency_decimals, is_test = excluded.is_test, "+
			"layer = excluded.layer, enabled = excluded.enabled, chain_color = excluded.chain_color, "+
			"short_name = excluded.short_name, related_chain_id = excluded.related_chain_id, is_active = excluded.is_active", "is_deactivatable = excluded.is_deactivatable")

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build upsert query: %w", err)
	}

	_, err = n.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute upsert query: %w", err)
	}

	return nil
}

// DeleteNetwork deletes a network and its associated RPC providers.
// Note: Transaction management should be handled by the caller.
func (n *NetworksPersistence) DeleteNetwork(chainID uint64) error {
	// Delete RPC providers
	err := n.rpcPersistence.DeleteRpcProviders(chainID)
	if err != nil {
		return fmt.Errorf("failed to delete RPC providers for chain_id %d: %w", chainID, err)
	}

	// Delete the network record
	q := sq.Delete("networks").Where(sq.Eq{"chain_id": chainID})
	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = n.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute delete query for chain_id %d: %w", chainID, err)
	}

	return nil
}

// SetActive updates the active status of a network.
func (n *NetworksPersistence) SetActive(chainID uint64, active bool) error {
	q := sq.Update("networks").
		Set("is_active", active).
		Where(sq.Eq{"chain_id": chainID})

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	_, err = n.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute update query for chain_id %d: %w", chainID, err)
	}

	return nil
}

// SetEnabled updates the enabled status of a network.
func (n *NetworksPersistence) SetEnabled(chainID uint64, enabled bool) error {
	q := sq.Update("networks").
		Set("enabled", enabled).
		Where(sq.Eq{"chain_id": chainID})

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	_, err = n.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute update query for chain_id %d: %w", chainID, err)
	}

	return nil
}
