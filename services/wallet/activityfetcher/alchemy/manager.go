// Package alchemy provides an integration manager that orchestrates the Alchemy API client
// with persistence and type conversions to implement the activity fetcher interface.
package alchemy

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	geth_rpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet/activityfetcher"
	wc "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	alchemy "github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"
)

type Manager struct {
	client            *alchemy.Client
	persistence       *alchemy.Persistence
	activityPublisher *pubsub.Publisher
}

func NewManager(client *alchemy.Client, persistence *alchemy.Persistence) *Manager {
	return &Manager{
		client:      client,
		persistence: persistence,
	}
}

func (m *Manager) SetActivityPublisher(publisher *pubsub.Publisher) {
	m.activityPublisher = publisher
}

func (m *Manager) ID() string {
	return alchemy.AlchemyID
}

func (m *Manager) IsConnected() bool {
	return m.client.IsConnected()
}

func (m *Manager) IsChainSupported(chainID wc.ChainID) bool {
	return m.client.IsChainSupported(chainID)
}

func (m *Manager) GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address common.Address) (*geth_rpc.BlockNumber, *time.Time, error) {
	return m.persistence.GetLastFetchedBlockAndTimestamp(ctx, chainID, address)
}

// FetchActivity orchestrates fetching, persistence, and type conversion.
func (m *Manager) FetchActivity(ctx context.Context, chainID uint64, parameters thirdparty.ActivityFetchParameters, cursor string, limit int) (thirdparty.ActivityEntryContainer, error) {
	transfers, nextCursor, err := m.client.FetchTransfers(ctx, chainID, parameters, cursor, limit)
	if err != nil {
		return thirdparty.ActivityEntryContainer{}, err
	}

	err = m.persistence.SaveTransfers(transfers, chainID, parameters.Address)
	if err != nil {
		return thirdparty.ActivityEntryContainer{}, err
	}

	// Emit event about ERC20 activity being fetched so interested components can react
	if m.activityPublisher != nil {
		for _, transfer := range transfers {
			if transfer.Category == alchemy.TransferCategoryErc20 && transfer.RawContract.Address != nil {
				pubsub.Publish(m.activityPublisher, activityfetcher.EventERC20ActivityFetched{
					ChainID: chainID,
					Address: *transfer.RawContract.Address,
				})
			}
		}
	}

	items := alchemy.TransfersToThirdpartyActivityEntries(transfers, chainID, parameters.Address)

	return thirdparty.ActivityEntryContainer{
		Provider:       m.ID(),
		Items:          items,
		PreviousCursor: cursor,
		NextCursor:     nextCursor,
	}, nil
}
