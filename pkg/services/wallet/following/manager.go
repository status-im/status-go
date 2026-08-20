package following

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/efp"
)

// Manager handles following address operations using EFP provider
type Manager struct {
	provider efp.FollowingDataProvider
	logger   *zap.Logger
}

// NewManager creates a new following manager with the provided EFP provider
func NewManager(provider efp.FollowingDataProvider, logger *zap.Logger) *Manager {
	return &Manager{
		provider: provider,
		logger:   logger,
	}
}

// FetchFollowingAddresses fetches the list of addresses that the given user is following
func (m *Manager) FetchFollowingAddresses(ctx context.Context, userAddress common.Address, search string, limit, offset int) ([]efp.FollowingAddress, error) {
	m.logger.Debug("following.Manager.FetchFollowingAddresses",
		zap.String("userAddress", userAddress.Hex()),
		zap.String("search", search),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	if m.provider == nil {
		return nil, errors.New("EFP provider not initialized")
	}

	if !m.provider.IsConnected() {
		m.logger.Warn("EFP provider not connected", zap.String("providerID", m.provider.ID()))
		return []efp.FollowingAddress{}, nil
	}

	startTime := time.Now()
	addresses, err := m.provider.FetchFollowingAddresses(ctx, userAddress, search, limit, offset)
	duration := time.Since(startTime)

	m.logger.Debug("following.Manager.FetchFollowingAddresses completed",
		zap.String("userAddress", userAddress.Hex()),
		zap.String("providerID", m.provider.ID()),
		zap.Int("addresses.len", len(addresses)),
		zap.Duration("duration", duration),
		zap.Error(err),
	)

	if err != nil {
		return nil, err
	}

	return addresses, nil
}

// FetchFollowingStats fetches the stats (following count) for a user
func (m *Manager) FetchFollowingStats(ctx context.Context, userAddress common.Address) (int, error) {
	m.logger.Debug("following.Manager.FetchFollowingStats",
		zap.String("userAddress", userAddress.Hex()),
	)

	if m.provider == nil {
		return 0, errors.New("EFP provider not initialized")
	}

	if !m.provider.IsConnected() {
		m.logger.Warn("EFP provider not connected", zap.String("providerID", m.provider.ID()))
		return 0, nil
	}

	count, err := m.provider.FetchFollowingStats(ctx, userAddress)
	if err != nil {
		m.logger.Error("following.Manager.FetchFollowingStats error", zap.Error(err))
		return 0, err
	}

	m.logger.Debug("following.Manager.FetchFollowingStats completed",
		zap.String("userAddress", userAddress.Hex()),
		zap.Int("count", count),
	)

	return count, nil
}
