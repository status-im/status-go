// Package alchemy provides an integration service that orchestrates the Alchemy API client
// with persistence and type conversions to implement the activity fetcher interface.
package alchemy

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	geth_rpc "github.com/ethereum/go-ethereum/rpc"

	wc "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	alchemy "github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"
)

type Service struct {
	client      *alchemy.Client
	persistence *alchemy.Persistence
}

func NewService(client *alchemy.Client, persistence *alchemy.Persistence) *Service {
	return &Service{
		client:      client,
		persistence: persistence,
	}
}

func (s *Service) ID() string {
	return alchemy.AlchemyID
}

func (s *Service) IsConnected() bool {
	return s.client.IsConnected()
}

func (s *Service) IsChainSupported(chainID wc.ChainID) bool {
	return s.client.IsChainSupported(chainID)
}

func (s *Service) GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address common.Address) (*geth_rpc.BlockNumber, *time.Time, error) {
	return s.persistence.GetLastFetchedBlockAndTimestamp(ctx, chainID, address)
}

// FetchActivity orchestrates fetching, persistence, and type conversion.
func (s *Service) FetchActivity(ctx context.Context, chainID uint64, parameters thirdparty.ActivityFetchParameters, cursor string, limit int) (thirdparty.ActivityEntryContainer, error) {

	transfers, nextCursor, err := s.client.FetchTransfers(ctx, chainID, parameters, cursor, limit)
	if err != nil {
		return thirdparty.ActivityEntryContainer{}, err
	}

	err = s.persistence.SaveTransfers(transfers, chainID, parameters.Address)
	if err != nil {
		return thirdparty.ActivityEntryContainer{}, err
	}

	items := alchemy.TransfersToThirdpartyActivityEntries(transfers, chainID, parameters.Address)

	return thirdparty.ActivityEntryContainer{
		Provider:       s.ID(),
		Items:          items,
		PreviousCursor: cursor,
		NextCursor:     nextCursor,
	}, nil
}
