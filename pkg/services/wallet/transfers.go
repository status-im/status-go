package wallet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/pkg/services/wallet/activityfetcher"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/activity/alchemy"
)

const (
	transfersDefaultLimit = 100
	transfersMaxLimit     = 1000
	transfersFetchTimeout = 30 * time.Second
)

// TransfersCoverage reports how fresh the history of one address on one chain is.
type TransfersCoverage struct {
	Address common.Address `json:"address"`
	ChainID uint64         `json:"chainId"`
	// False when the fetch before this call did not finish in time or failed.
	Complete bool `json:"complete"`
	// Block of the newest stored transfer, nil when none is stored.
	LastFetchedBlock *uint64 `json:"lastFetchedBlock"`
}

type TransfersResponse struct {
	Transfers []alchemy.HistoryTransfer `json:"transfers"`
	Coverage  []TransfersCoverage       `json:"coverage"`
}

type transferFetcher interface {
	IsChainSupported(chainID uint64) bool
	FetchAndWait(ctx context.Context, keys []activityfetcher.FetchKey) map[activityfetcher.FetchKey]bool
}

type transferStorage interface {
	GetHistory(ctx context.Context, chainIDs []uint64, addresses []common.Address, categories []alchemy.TransferCategory, limit, offset int) ([]alchemy.StoredTransfer, error)
	GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address common.Address) (*gethrpc.BlockNumber, *time.Time, error)
}

type transferHistory struct {
	fetcher      transferFetcher
	storage      transferStorage
	fetchTimeout time.Duration
}

func (h *transferHistory) getTransfers(ctx context.Context, addresses []common.Address, chainIDs []uint64, includeCollectibles bool, limit, offset int) (*TransfersResponse, error) {
	addresses = unique(addresses)
	chainIDs = unique(chainIDs)
	if len(addresses) == 0 {
		return nil, errors.New("no addresses")
	}
	if len(chainIDs) == 0 {
		return nil, errors.New("no chain ids")
	}
	if limit < 1 || limit > transfersMaxLimit {
		return nil, fmt.Errorf("limit must be between 1 and %d", transfersMaxLimit)
	}
	if offset < 0 {
		return nil, errors.New("offset must not be negative")
	}
	for _, chainID := range chainIDs {
		if !h.fetcher.IsChainSupported(chainID) {
			return nil, fmt.Errorf("no history provider for chain %d", chainID)
		}
	}

	keys := make([]activityfetcher.FetchKey, 0, len(addresses)*len(chainIDs))
	for _, address := range addresses {
		for _, chainID := range chainIDs {
			keys = append(keys, activityfetcher.FetchKey{Address: address, ChainID: chainID})
		}
	}
	fetchCtx, cancel := context.WithTimeout(ctx, h.fetchTimeout)
	complete := h.fetcher.FetchAndWait(fetchCtx, keys)
	cancel()

	categories := []alchemy.TransferCategory{alchemy.TransferCategoryExternal, alchemy.TransferCategoryErc20}
	if includeCollectibles {
		categories = append(categories, alchemy.TransferCategoryErc721, alchemy.TransferCategoryErc1155, alchemy.TransferCategorySpecialNft)
	}
	stored, err := h.storage.GetHistory(ctx, chainIDs, addresses, categories, limit, offset)
	if err != nil {
		return nil, err
	}

	response := &TransfersResponse{
		Transfers: make([]alchemy.HistoryTransfer, 0, len(stored)),
		Coverage:  make([]TransfersCoverage, 0, len(keys)),
	}
	for _, s := range stored {
		response.Transfers = append(response.Transfers, alchemy.ToHistoryTransfer(s))
	}
	for _, key := range keys {
		coverage := TransfersCoverage{Address: key.Address, ChainID: key.ChainID, Complete: complete[key]}
		block, _, err := h.storage.GetLastFetchedBlockAndTimestamp(ctx, key.ChainID, key.Address)
		if err != nil {
			return nil, err
		}
		if block != nil {
			b := uint64(block.Int64())
			coverage.LastFetchedBlock = &b
		}
		response.Coverage = append(response.Coverage, coverage)
	}
	return response, nil
}

func unique[T comparable](values []T) []T {
	seen := make(map[T]struct{}, len(values))
	result := make([]T, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}
