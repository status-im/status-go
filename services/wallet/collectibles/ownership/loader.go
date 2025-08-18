package ownership

//go:generate mockgen -package=mock_ownership -source=loader.go -destination=mock/loader.go

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"go.uber.org/zap"
)

var ErrLoadAlreadyInProgress = errors.New("load already in progress")

type CollectibleOwnershipFetcher interface {
	FetchCollectibleOwnershipByOwner(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error)
}

type CollectibleOwnershipStorage interface {
	GetOwnershipUpdateTimestamp(owner common.Address, chainID walletCommon.ChainID) (int64, error)
	Update(chainID walletCommon.ChainID, ownerAddress common.Address, balances thirdparty.TokenBalancesPerContractAddress, timestamp int64) (removedIDs, updatedIDs, insertedIDs []thirdparty.CollectibleUniqueID, err error)
}

type LoaderParams struct {
	LoadDelay  time.Duration // (Optional) Delay between notifying a Load start and actually triggering the first fetch, used to "debounce" fetches triggered by settings changes
	FetchLimit int           // (Optional) Limit the number of collectibles to fetch per page during the initial load
}

func DefaultLoaderParams() LoaderParams {
	return LoaderParams{
		LoadDelay:  10 * time.Second,
		FetchLimit: 50,
	}
}

type Loader struct {
	chainID   walletCommon.ChainID
	account   common.Address
	fetcher   CollectibleOwnershipFetcher
	storage   CollectibleOwnershipStorage
	publisher *pubsub.Publisher
	params    LoaderParams

	loading atomic.Bool

	logger *zap.Logger
}

func NewLoader(chainID walletCommon.ChainID, account common.Address, fetcher CollectibleOwnershipFetcher, storage CollectibleOwnershipStorage, publisher *pubsub.Publisher, params LoaderParams, logger *zap.Logger) *Loader {
	return &Loader{
		chainID:   chainID,
		account:   account,
		fetcher:   fetcher,
		storage:   storage,
		publisher: publisher,
		params:    params,
		logger:    logger.Named("Loader"),
	}
}

// Loads owned collectibles for a ChainID+OwnerAddress combination
// If there's no record of a successful load in the storage, fetches in chunks and updates
// the storage with the accumulated ownership after each fetch.
// Otherwise, fetches and updates the storage in a single call
func (l *Loader) Load(ctx context.Context) ([]thirdparty.CollectibleIDBalance, error) {
	if !l.loading.CompareAndSwap(false, true) {
		// Load already in progress
		return nil, ErrLoadAlreadyInProgress
	}
	defer l.loading.Store(false)

	l.logger.Debug("start collectibles Loader",
		zap.Uint64("chain", uint64(l.chainID)),
		zap.String("account", gocommon.TruncateWithDot(l.account.String())),
	)

	pubsub.Publish(l.publisher, EventOwnedCollectiblesLoadStarted{
		ChainID: l.chainID,
		Account: l.account,
	})

	var err error
	// Handle error at the end, if any
	defer func() {
		if err != nil {
			pubsub.Publish(l.publisher, EventOwnedCollectiblesLoadError{
				ChainID: l.chainID,
				Account: l.account,
				Error:   err,
			})
			return
		}
	}()

	var lastFetchTimestamp int64
	lastFetchTimestamp, err = l.storage.GetOwnershipUpdateTimestamp(l.account, l.chainID)
	if err != nil {
		return nil, err
	}
	isInitialLoad := lastFetchTimestamp == InvalidTimestamp

	pageSize := thirdparty.FetchNoLimit
	if isInitialLoad {
		pageSize = l.params.FetchLimit
	}

	// Start fetching collectibles in chunks
	pageNr := 0
	cursor := thirdparty.FetchFromStartCursor
	providerID := thirdparty.FetchFromAnyProvider

	accumulatedOwnership := make([]thirdparty.CollectibleIDBalance, 0)

	pageStart := time.Now()
	l.logger.Debug("start collectibles Loader",
		zap.Uint64("chain", uint64(l.chainID)),
		zap.String("account", gocommon.TruncateWithDot(l.account.String())),
		zap.Int("page", pageNr),
	)

	if l.params.LoadDelay.Seconds() > 0 {
		if l.waitFor(ctx, l.params.LoadDelay) {
			return nil, ctx.Err()
		}
	}

	start := time.Now()

	for {
		if walletCommon.ShouldCancel(ctx) {
			err = ctx.Err()
			l.logger.Error("cancelled collectibles Loader",
				zap.Uint64("chain", uint64(l.chainID)),
				zap.String("account", gocommon.TruncateWithDot(l.account.String())),
				zap.Int("page", pageNr),
				zap.Error(err),
			)
			return nil, err
		}

		ownershipPage, tmpErr := l.fetcher.FetchCollectibleOwnershipByOwner(ctx, l.chainID, l.account, cursor, pageSize, providerID)
		if tmpErr != nil {
			err = tmpErr
			l.logger.Error("failed fetching collectible ownership",
				zap.Uint64("chain", uint64(l.chainID)),
				zap.String("account", gocommon.TruncateWithDot(l.account.String())),
				zap.Int("page", pageNr),
				zap.Error(err),
			)
			return nil, err
		}

		accumulatedOwnership = append(accumulatedOwnership, ownershipPage.Items...)

		if isInitialLoad {
			l.logger.Debug("partial collectibles Loader",
				zap.Uint64("chain", uint64(l.chainID)),
				zap.String("account", gocommon.TruncateWithDot(l.account.String())),
				zap.Int("page", pageNr),
				zap.Duration("duration", time.Since(pageStart)),
				zap.Int("found", len(ownershipPage.Items)),
			)

			partialEvent := EventOwnedCollectiblesLoadPartial{
				ChainID:          l.chainID,
				Account:          l.account,
				PartialOwnership: accumulatedOwnership,
			}

			_, _, partialEvent.Added, tmpErr = l.storage.Update(l.chainID, l.account, ownedTokensToTokenBalancesPerContractAddress(accumulatedOwnership), InvalidTimestamp)
			if tmpErr != nil {
				err = tmpErr
				return nil, err
			}

			pubsub.Publish(l.publisher, partialEvent)
		}

		pageNr++
		cursor = ownershipPage.NextCursor
		providerID = ownershipPage.Provider

		finished := cursor == thirdparty.FetchFromStartCursor
		if finished {
			break
		}
	}

	l.logger.Debug("end collectibles Loader",
		zap.Uint64("chain", uint64(l.chainID)),
		zap.String("account", gocommon.TruncateWithDot(l.account.String())),
		zap.Duration("in", time.Since(start)),
		zap.Int("found", len(accumulatedOwnership)),
	)

	finishedEvent := EventOwnedCollectiblesLoadFinished{
		ChainID:      l.chainID,
		Account:      l.account,
		NewOwnership: accumulatedOwnership,
	}

	finishedEvent.Removed, finishedEvent.Updated, finishedEvent.Added, err = l.storage.Update(l.chainID, l.account, ownedTokensToTokenBalancesPerContractAddress(accumulatedOwnership), start.Unix())
	if err != nil {
		return nil, err
	}

	pubsub.Publish(l.publisher, finishedEvent)

	return accumulatedOwnership, nil
}

func (l *Loader) waitFor(ctx context.Context, delay time.Duration) (cancelled bool) {
	delayTimer := time.NewTimer(delay)
	defer delayTimer.Stop()
	select {
	case <-delayTimer.C:
		break
	case <-ctx.Done():
		return true
	}
	return false
}

func ownedTokensToTokenBalancesPerContractAddress(ownership []thirdparty.CollectibleIDBalance) thirdparty.TokenBalancesPerContractAddress {
	ret := make(thirdparty.TokenBalancesPerContractAddress)
	for _, idBalance := range ownership {
		balanceBigInt := idBalance.Balance
		if balanceBigInt == nil {
			balanceBigInt = &bigint.BigInt{Int: big.NewInt(1)}
		}
		balance := thirdparty.TokenBalance{
			TokenID: idBalance.ID.TokenID,
			Balance: balanceBigInt,
		}
		ret[idBalance.ID.ContractID.Address] = append(ret[idBalance.ID.ContractID.Address], balance)
	}
	return ret
}
