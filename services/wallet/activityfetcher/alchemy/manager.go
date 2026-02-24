// Package alchemy provides an integration manager that orchestrates the Alchemy API client
// with persistence and type conversions to implement the activity fetcher interface.
package alchemy

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	geth_rpc "github.com/ethereum/go-ethereum/rpc"

	ac "github.com/status-im/status-go/services/wallet/activity/common"
	wc "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	alchemy "github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"
)

type Manager struct {
	client      *alchemy.Client
	persistence *alchemy.Persistence
}

func NewManager(client *alchemy.Client, persistence *alchemy.Persistence) *Manager {
	return &Manager{
		client:      client,
		persistence: persistence,
	}
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

	items := alchemy.TransfersToThirdpartyActivityEntries(transfers, chainID, parameters.Address)

	return thirdparty.ActivityEntryContainer{
		Provider:       m.ID(),
		Items:          items,
		PreviousCursor: cursor,
		NextCursor:     nextCursor,
	}, nil
}

func (m *Manager) ClearAll(ctx context.Context) error {
	return m.persistence.ClearAll(ctx)
}

// GetActivityTokens returns unique tokens from activity transfers for a given account and chain
func (m *Manager) GetActivityTokens(ctx context.Context, chainID uint64, address common.Address) ([]ac.Token, error) {
	tokens, err := m.persistence.GetActivityTokens(chainID, address)
	if err != nil {
		return nil, err
	}

	// Convert alchemy.TokenInfo to activity common.Token
	result := make([]ac.Token, 0, len(tokens))
	for _, token := range tokens {
		var tokenType ac.TokenType
		if token.Category == alchemy.TransferCategoryExternal || token.Category == alchemy.TransferCategoryInternal {
			tokenType = ac.Native
		} else if token.Category == alchemy.TransferCategoryErc20 {
			tokenType = ac.Erc20
		} else {
			// Skip non-native, non-ERC20 tokens
			continue
		}

		result = append(result, ac.Token{
			TokenType: tokenType,
			ChainID:   wc.ChainID(chainID),
			Address:   token.Address,
		})
	}

	return result, nil
}
