package following

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/thirdparty/efp"
)

// Manager handles following address operations using EFP providers
type Manager struct {
	providers []efp.FollowingDataProvider
}

// NewManager creates a new following manager with the provided EFP providers
func NewManager(providers []efp.FollowingDataProvider) *Manager {
	return &Manager{
		providers: providers,
	}
}

// FetchFollowingAddresses fetches the list of addresses that the given user is following
// Uses the first available provider (can be enhanced later with fallback logic)
func (m *Manager) FetchFollowingAddresses(ctx context.Context, userAddress common.Address, search string, limit, offset int) ([]efp.FollowingAddress, error) {
	logutils.ZapLogger().Debug("following.Manager.FetchFollowingAddresses",
		zap.String("userAddress", userAddress.Hex()),
		zap.String("search", search),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("providers.len", len(m.providers)),
	)

	if len(m.providers) == 0 {
		return []efp.FollowingAddress{}, nil
	}

	// Use the first provider (EFP client)
	provider := m.providers[0]
	if !provider.IsConnected() {
		logutils.ZapLogger().Warn("EFP provider not connected", zap.String("providerID", provider.ID()))
		return []efp.FollowingAddress{}, nil
	}

	startTime := time.Now()
	addresses, err := provider.FetchFollowingAddresses(ctx, userAddress, search, limit, offset)
	duration := time.Since(startTime)

	logutils.ZapLogger().Debug("following.Manager.FetchFollowingAddresses completed",
		zap.String("userAddress", userAddress.Hex()),
		zap.String("providerID", provider.ID()),
		zap.Int("addresses.len", len(addresses)),
		zap.Duration("duration", duration),
		zap.Error(err),
	)

	if err != nil {
		return nil, err
	}

	return addresses, nil
}
