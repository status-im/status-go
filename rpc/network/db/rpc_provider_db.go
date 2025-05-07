package db

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/sqlite"
)

// Interface for managing RPC providers
type RpcProvidersPersistenceInterface interface {
	GetRpcProviders(chainID uint64) ([]params.RpcProvider, error)
	GetRpcProvidersByType(chainID uint64, providerType params.RpcProviderType) ([]params.RpcProvider, error)
	AddRpcProvider(provider params.RpcProvider) error
	DeleteRpcProviders(chainID uint64) error
	DeleteAllRpcProviders() error
	UpdateRpcProvider(provider params.RpcProvider) error
	SetRpcProviders(chainID uint64, newProviders []params.RpcProvider) error
}

// Struct for managing RPC providers
type RpcProvidersPersistence struct {
	db        sqlite.StatementExecutor
	validator *validator.Validate
}

// Constructor for RpcProvidersPersistence with validator
func NewRpcProvidersPersistence(db sqlite.StatementExecutor) *RpcProvidersPersistence {
	return &RpcProvidersPersistence{
		db:        db,
		validator: validator.New(),
	}
}

// Retrieve all providers for a specific ChainID
func (p *RpcProvidersPersistence) GetRpcProviders(chainID uint64) ([]params.RpcProvider, error) {
	q := sq.Select(
		"id",
		"chain_id",
		"name",
		"url",
		"enable_rps_limiter",
		"type",
		"enabled",
		"auth_type",
		"auth_login",
		"auth_password",
		"auth_token",
	).
		From("rpc_providers").
		Where(sq.Eq{"chain_id": chainID}).
		OrderBy("id ASC")

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var providers []params.RpcProvider
	for rows.Next() {
		var provider params.RpcProvider

		err := rows.Scan(
			&provider.ID,
			&provider.ChainID,
			&provider.Name,
			&provider.URL,
			&provider.EnableRPSLimiter,
			&provider.Type,
			&provider.Enabled,
			&provider.AuthType,
			&provider.AuthLogin,
			&provider.AuthPassword,
			&provider.AuthToken,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		providers = append(providers, provider)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return providers, nil
}

// Retrieve providers of a specific type
func (p *RpcProvidersPersistence) GetRpcProvidersByType(chainID uint64, providerType params.RpcProviderType) ([]params.RpcProvider, error) {
	allProviders, err := p.GetRpcProviders(chainID)
	if err != nil {
		return nil, err
	}

	result := make([]params.RpcProvider, 0, len(allProviders))
	for _, provider := range allProviders {
		if provider.Type == providerType {
			result = append(result, provider)
		}
	}

	return result, nil
}

// Add a new provider
func (p *RpcProvidersPersistence) AddRpcProvider(provider params.RpcProvider) error {
	// Validate the provider
	if err := p.validator.Struct(provider); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Proceed with adding the provider to the database
	q := sq.Insert("rpc_providers").
		Columns(
			"chain_id",
			"name",
			"url",
			"enable_rps_limiter",
			"type",
			"enabled",
			"auth_type",
			"auth_login",
			"auth_password",
			"auth_token",
		).
		Values(
			provider.ChainID,
			provider.Name,
			provider.URL,
			provider.EnableRPSLimiter,
			provider.Type,
			provider.Enabled,
			provider.AuthType,
			provider.AuthLogin,
			provider.AuthPassword,
			provider.AuthToken,
		)

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute insert query: %w", err)
	}

	return nil
}

// Delete providers for a specific ChainID
func (p *RpcProvidersPersistence) DeleteRpcProviders(chainID uint64) error {
	q := sq.Delete("rpc_providers").
		Where(sq.Eq{"chain_id": chainID})

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute delete query: %w", err)
	}

	return nil
}

// Delete all providers
func (p *RpcProvidersPersistence) DeleteAllRpcProviders() error {
	q := sq.Delete("rpc_providers")

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute delete query: %w", err)
	}

	return nil
}

// Update an existing provider
func (p *RpcProvidersPersistence) UpdateRpcProvider(provider params.RpcProvider) error {
	// Validate the provider
	if err := p.validator.Struct(provider); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Proceed with updating the provider in the database
	q := sq.Update("rpc_providers").
		SetMap(sq.Eq{
			"url":                provider.URL,
			"enable_rps_limiter": provider.EnableRPSLimiter,
			"type":               provider.Type,
			"enabled":            provider.Enabled,
			"auth_type":          provider.AuthType,
			"auth_login":         provider.AuthLogin,
			"auth_password":      provider.AuthPassword,
			"auth_token":         provider.AuthToken,
		}).
		Where(sq.Eq{"id": provider.ID})

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	_, err = p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute update query: %w", err)
	}

	return nil
}

// Set the list of providers for a ChainID, replacing any existing providers
func (p *RpcProvidersPersistence) SetRpcProviders(chainID uint64, newProviders []params.RpcProvider) error {
	if err := p.DeleteRpcProviders(chainID); err != nil {
		return fmt.Errorf("failed to delete existing providers: %w", err)
	}

	for _, provider := range newProviders {
		if err := p.AddRpcProvider(provider); err != nil {
			return fmt.Errorf("failed to add new provider: %w", err)
		}
	}

	return nil
}
