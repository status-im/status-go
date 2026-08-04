package collectibles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/pubsub"

	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/collectibles/ownership"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// These events are used to notify the UI of state changes
const (
	EventCollectiblesOwnershipUpdateStarted           walletevent.EventType = "wallet-collectibles-ownership-update-started"
	EventCollectiblesOwnershipUpdatePartial           walletevent.EventType = "wallet-collectibles-ownership-update-partial"
	EventCollectiblesOwnershipUpdateFinished          walletevent.EventType = "wallet-collectibles-ownership-update-finished"
	EventCollectiblesOwnershipUpdateFinishedWithError walletevent.EventType = "wallet-collectibles-ownership-update-finished-with-error"
	EventCollectiblesDataUpdated                      walletevent.EventType = "wallet-collectibles-data-updated"

	EventOwnedCollectiblesFilteringDone walletevent.EventType = "wallet-owned-collectibles-filtering-done"
	EventGetCollectiblesDetailsDone     walletevent.EventType = "wallet-get-collectibles-details-done"
	EventGetCollectionSocialsDone       walletevent.EventType = "wallet-get-collection-socials-done"
)

type OwnershipUpdateMessage struct {
	Added   []thirdparty.CollectibleUniqueID `json:"added"`
	Updated []thirdparty.CollectibleUniqueID `json:"updated"`
	Removed []thirdparty.CollectibleUniqueID `json:"removed"`
}

type CollectionSocialsMessage struct {
	ID      thirdparty.ContractID         `json:"id"`
	Socials *thirdparty.CollectionSocials `json:"socials"`
}

type CollectibleDataType byte

const (
	CollectibleDataTypeUniqueID CollectibleDataType = iota
	CollectibleDataTypeHeader
	CollectibleDataTypeDetails
	CollectibleDataTypeCommunityHeader
)

type FetchType byte

const (
	FetchTypeNeverFetch FetchType = iota
	FetchTypeAlwaysFetch
	FetchTypeFetchIfNotCached
	FetchTypeFetchIfCacheOld
)

type TxHashData struct {
	Hash common.Hash
	TxID common.Hash
}

type FetchCriteria struct {
	FetchType          FetchType `json:"fetch_type"`
	MaxCacheAgeSeconds int64     `json:"max_cache_age_seconds"`
}

var (
	filterOwnedCollectiblesTask = async.TaskType{
		ID:     1,
		Policy: async.ReplacementPolicyCancelOld,
	}
	getCollectiblesDataTask = async.TaskType{
		ID:     2,
		Policy: async.ReplacementPolicyCancelOld,
	}
)

type Service struct {
	manager             *Manager
	ownershipController *ownership.Controller
	db                  *sql.DB
	ownershipDB         ownership.OwnershipStorage
	communityManager    *community.Manager
	walletFeed          *event.Feed
	scheduler           *async.MultiClientScheduler
	group               *async.Group
	publisher           *pubsub.Publisher

	walletEventsWatcher *walletevent.Watcher

	closeCh chan struct{}

	// maxAssetSize is the largest asset, in bytes, we are willing to hand a
	// client a URL for. Set by the client, read on every response, so it is
	// atomic rather than guarded by the service's lifecycle.
	maxAssetSize atomic.Int64

	logger *zap.Logger
}

func NewService(
	db *sql.DB,
	walletFeed *event.Feed,
	communityManager *community.Manager,
	manager *Manager,
	ownershipController *ownership.Controller,
	publisher *pubsub.Publisher,
) *Service {
	ownershipDB := ownership.NewOwnershipDB(db)

	logger := logutils.ZapLogger().Named("Collectibles")

	s := &Service{
		manager:             manager,
		ownershipController: ownershipController,
		db:                  db,
		ownershipDB:         ownershipDB,
		communityManager:    communityManager,
		walletFeed:          walletFeed,
		scheduler:           async.NewMultiClientScheduler(),
		group:               async.NewGroup(context.Background()),
		publisher:           publisher,
		logger:              logger.Named("Service"),
	}
	return s
}

type ErrorCode = int

const (
	ErrorCodeSuccess ErrorCode = iota + 1
	ErrorCodeTaskCanceled
	ErrorCodeFailed
)

type OwnershipStatus struct {
	State     ownership.LoaderState `json:"state"`
	Timestamp int64                 `json:"timestamp"`
}

type OwnershipStatusPerChainID = map[walletCommon.ChainID]OwnershipStatus
type OwnershipStatusPerAddressAndChainID = map[common.Address]OwnershipStatusPerChainID

type GetOwnedCollectiblesResponse struct {
	Collectibles []Collectible `json:"collectibles"`
	Offset       int           `json:"offset"`
	// Used to indicate that there might be more collectibles that were not returned
	// based on a simple heuristic
	HasMore         bool                                `json:"hasMore"`
	OwnershipStatus OwnershipStatusPerAddressAndChainID `json:"ownershipStatus"`
	ErrorCode       ErrorCode                           `json:"errorCode"`
}

type GetCollectiblesByUniqueIDResponse struct {
	Collectibles []Collectible `json:"collectibles"`
	ErrorCode    ErrorCode     `json:"errorCode"`
}

type GetOwnedCollectiblesReturnType struct {
	collectibles    []Collectible
	hasMore         bool
	ownershipStatus OwnershipStatusPerAddressAndChainID
}

type GetCollectiblesByUniqueIDReturnType struct {
	collectibles []Collectible
}

func (s *Service) GetOwnedCollectibles(
	ctx context.Context,
	chainIDs []walletCommon.ChainID,
	addresses []common.Address,
	filter Filter,
	offset int,
	limit int,
	dataType CollectibleDataType,
	fetchCriteria FetchCriteria) (*GetOwnedCollectiblesReturnType, error) {
	err := s.fetchOwnedCollectiblesIfNeeded(ctx, chainIDs, addresses, fetchCriteria)
	if err != nil {
		return nil, err
	}

	ids, hasMore, err := s.FilterOwnedCollectibles(chainIDs, addresses, filter, offset, limit)
	if err != nil {
		return nil, err
	}

	collectibles, err := s.collectibleIDsToDataType(ctx, ids, dataType)
	if err != nil {
		return nil, err
	}

	ownershipStatus, err := s.GetOwnershipStatus(chainIDs, addresses)
	if err != nil {
		return nil, err
	}

	return &GetOwnedCollectiblesReturnType{
		collectibles:    collectibles,
		hasMore:         hasMore,
		ownershipStatus: ownershipStatus,
	}, err
}

func (s *Service) shouldTriggerLoad(chainID walletCommon.ChainID, address common.Address, fetchCriteria FetchCriteria) (bool, error) {
	switch fetchCriteria.FetchType {
	case FetchTypeNeverFetch:
		return false, nil
	case FetchTypeAlwaysFetch:
		return true, nil
	case FetchTypeFetchIfNotCached, FetchTypeFetchIfCacheOld:
		timestamp, err := s.ownershipDB.GetOwnershipUpdateTimestamp(address, chainID)
		if err != nil {
			return false, err
		}
		if timestamp != ownership.InvalidTimestamp &&
			!(fetchCriteria.FetchType == FetchTypeFetchIfCacheOld && timestamp+fetchCriteria.MaxCacheAgeSeconds < time.Now().Unix()) {
			return false, nil
		}
		// Skip duplicate load unless the loader is missing or errored
		if fetchCriteria.FetchType == FetchTypeFetchIfNotCached {
			state := s.ownershipController.GetLoaderState(chainID, address)
			if state != ownership.LoaderStateNotAvailable && state != ownership.LoaderStateError {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) fetchOwnedCollectiblesIfNeeded(ctx context.Context, chainIDs []walletCommon.ChainID, addresses []common.Address, fetchCriteria FetchCriteria) error {
	if fetchCriteria.FetchType == FetchTypeNeverFetch {
		return nil
	}

	wg := sync.WaitGroup{}
	wgErr := atomic.NewError(nil)
	for _, address := range addresses {
		for _, chainID := range chainIDs {
			if ok, err := s.shouldTriggerLoad(chainID, address, fetchCriteria); err != nil {
				return err
			} else if !ok {
				continue
			}

			wg.Add(1)
			go func() {
				defer gocommon.LogOnPanic()
				defer wg.Done()
				if err := s.ownershipController.TriggerLoad(ctx, chainID, address); err != nil {
					wgErr.Store(err)
				}
			}()
		}
	}

	wg.Wait()
	return wgErr.Load()
}

// GetOwnedCollectiblesAsync allows only one filter task to run at a time
// and it cancels the current one if a new one is started
// All calls will trigger an EventOwnedCollectiblesFilteringDone event with the result of the filtering
func (s *Service) GetOwnedCollectiblesAsync(
	requestID int32,
	chainIDs []walletCommon.ChainID,
	addresses []common.Address,
	filter Filter,
	offset int,
	limit int,
	dataType CollectibleDataType,
	fetchCriteria FetchCriteria) {
	s.scheduler.Enqueue(requestID, filterOwnedCollectiblesTask, func(ctx context.Context) (interface{}, error) {
		return s.GetOwnedCollectibles(ctx, chainIDs, addresses, filter, offset, limit, dataType, fetchCriteria)
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := GetOwnedCollectiblesResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			fnRet := result.(*GetOwnedCollectiblesReturnType)
			res.Collectibles = fnRet.collectibles
			res.Offset = offset
			res.HasMore = fnRet.hasMore
			res.OwnershipStatus = fnRet.ownershipStatus
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(&requestID, EventOwnedCollectiblesFilteringDone, res, err)
	})
}

func (s *Service) GetCollectiblesByUniqueID(
	ctx context.Context,
	uniqueIDs []thirdparty.CollectibleUniqueID,
	dataType CollectibleDataType) (*GetCollectiblesByUniqueIDReturnType, error) {
	collectibles, err := s.collectibleIDsToDataType(ctx, uniqueIDs, dataType)
	if err != nil {
		return nil, err
	}
	return &GetCollectiblesByUniqueIDReturnType{
		collectibles: collectibles,
	}, err
}

func (s *Service) GetCollectiblesByUniqueIDAsync(
	requestID int32,
	uniqueIDs []thirdparty.CollectibleUniqueID,
	dataType CollectibleDataType) {
	s.scheduler.Enqueue(requestID, getCollectiblesDataTask, func(ctx context.Context) (interface{}, error) {
		return s.GetCollectiblesByUniqueID(ctx, uniqueIDs, dataType)
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := GetCollectiblesByUniqueIDResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			fnRet := result.(*GetCollectiblesByUniqueIDReturnType)
			res.Collectibles = fnRet.collectibles
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(&requestID, EventGetCollectiblesDetailsDone, res, err)
	})
}

func (s *Service) RefetchOwnedCollectibles() {
	s.manager.ResetConnectionStatus()
	s.ownershipController.RefetchOwnedCollectibles()
}

// SetMaxAssetSize caps the size of a single collectible asset, in bytes, that
// the client is willing to be handed a URL for. Zero or less lifts the cap.
//
// The client owns the number because it is the one paying for the download and
// the one that can measure what it cost; status-go only knows the sizes the
// providers reported. Applied on read, so a new value takes effect on the next
// response without refetching anything.
func (s *Service) SetMaxAssetSize(size int64) {
	s.maxAssetSize.Store(size)
}

func (s *Service) Start(ctx context.Context) {
	if s.closeCh != nil {
		return
	}
	s.closeCh = make(chan struct{})

	s.ownershipController.Start()
	s.startOwnershipLoadWatcher()
}

func (s *Service) Stop() {
	if s.closeCh == nil {
		return
	}

	close(s.closeCh)
	s.closeCh = nil

	s.scheduler.Stop()
	s.ownershipController.Stop()
}

func (s *Service) sendResponseEvent(requestID *int32, eventType walletevent.EventType, payloadObj interface{}, resErr error) {
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		s.logger.Error("Error marshaling", zap.NamedError("response", err), zap.NamedError("result", resErr))
	} else {
		err = resErr
	}

	s.logger.Debug("RESPONSE",
		zap.Any("requestID", requestID),
		zap.String("eventType", string(eventType)),
		zap.Int("payload.len", len(payload)),
		zap.Error(err),
	)

	event := walletevent.Event{
		Type:    eventType,
		Message: string(payload),
	}

	if requestID != nil {
		event.RequestID = new(int)
		*event.RequestID = int(*requestID)
	}

	s.walletFeed.Send(event)
}

func (s *Service) FilterOwnedCollectibles(chainIDs []walletCommon.ChainID, owners []common.Address, filter Filter, offset int, limit int) ([]thirdparty.CollectibleUniqueID, bool, error) {
	ctx := context.Background()
	// Request one more than limit, to check if DB has more available
	ids, err := filterOwnedCollectibles(ctx, s.db, chainIDs, owners, filter, offset, limit+1)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(ids) > limit
	if hasMore {
		ids = ids[:limit]
	}

	return ids, hasMore, nil
}

func (s *Service) GetOwnedCollectible(chainID walletCommon.ChainID, owner common.Address, contractAddress common.Address, tokenID *big.Int) (*thirdparty.CollectibleUniqueID, error) {
	return s.ownershipDB.GetOwnedCollectible(chainID, owner, contractAddress, tokenID)
}

func (s *Service) GetOwnershipStatus(chainIDs []walletCommon.ChainID, owners []common.Address) (OwnershipStatusPerAddressAndChainID, error) {
	ret := make(OwnershipStatusPerAddressAndChainID)
	for _, address := range owners {
		ret[address] = make(OwnershipStatusPerChainID)
		for _, chainID := range chainIDs {
			timestamp, err := s.ownershipDB.GetOwnershipUpdateTimestamp(address, chainID)
			if err != nil {
				return nil, err
			}
			ret[address][chainID] = OwnershipStatus{
				State:     s.ownershipController.GetLoaderState(chainID, address),
				Timestamp: timestamp,
			}
		}
	}

	return ret, nil
}

func (s *Service) collectibleIDsToDataType(ctx context.Context, ids []thirdparty.CollectibleUniqueID, dataType CollectibleDataType) ([]Collectible, error) {
	switch dataType {
	case CollectibleDataTypeUniqueID:
		return idsToCollectibles(ids), nil
	case CollectibleDataTypeHeader, CollectibleDataTypeDetails, CollectibleDataTypeCommunityHeader:
		collectibles, err := s.manager.FetchAssetsByCollectibleUniqueID(ctx, ids, true)
		if err != nil {
			return nil, err
		}
		// The one place every shape of collectible response passes through, so
		// the cap is applied once rather than per shape.
		applyAssetSizeLimit(collectibles, s.maxAssetSize.Load())
		switch dataType {
		case CollectibleDataTypeHeader:
			return fullCollectiblesDataToHeaders(collectibles), nil
		case CollectibleDataTypeDetails:
			return fullCollectiblesDataToDetails(collectibles), nil
		case CollectibleDataTypeCommunityHeader:
			return fullCollectiblesDataToCommunityHeader(collectibles), nil
		}
	}
	return nil, errors.New("unknown data type")
}

func (s *Service) startOwnershipLoadWatcher() {
	if s.publisher == nil {
		return
	}

	loadStartCh, loadStartUnsub := pubsub.Subscribe[ownership.EventOwnedCollectiblesLoadStarted](s.publisher, 100)
	loadPartialCh, loadPartialUnsub := pubsub.Subscribe[ownership.EventOwnedCollectiblesLoadPartial](s.publisher, 100)
	loadFinishedCh, loadFinishedUnsub := pubsub.Subscribe[ownership.EventOwnedCollectiblesLoadFinished](s.publisher, 100)
	loadErrorCh, loadErrorUnsub := pubsub.Subscribe[ownership.EventOwnedCollectiblesLoadError](s.publisher, 100)

	go func() {
		defer gocommon.LogOnPanic()
		defer loadStartUnsub()
		defer loadPartialUnsub()
		defer loadFinishedUnsub()
		defer loadErrorUnsub()
		for {
			select {
			case event := <-loadStartCh:
				// Send WalletEvent to the client
				s.triggerWalletEvent(EventCollectiblesOwnershipUpdateStarted, event.ChainID, event.Account, "")
			case event := <-loadPartialCh:
				// Send WalletEvent to the client
				updateMessage := OwnershipUpdateMessage{
					Added: event.Added,
				}
				encodedMessage, err := json.Marshal(updateMessage)
				if err != nil {
					s.logger.Error("Error marshaling", zap.NamedError("response", err))
				}
				s.triggerWalletEvent(EventCollectiblesOwnershipUpdatePartial, event.ChainID, event.Account, string(encodedMessage))
			case event := <-loadFinishedCh:
				// Send WalletEvent to the client
				updateMessage := OwnershipUpdateMessage{
					Added:   event.Added,
					Updated: event.Updated,
					Removed: event.Removed,
				}
				encodedMessage, err := json.Marshal(updateMessage)
				if err != nil {
					s.logger.Error("Error marshaling", zap.NamedError("response", err))
				}
				s.triggerWalletEvent(EventCollectiblesOwnershipUpdateFinished, event.ChainID, event.Account, string(encodedMessage))
			case event := <-loadErrorCh:
				// Send WalletEvent to the client
				s.triggerWalletEvent(EventCollectiblesOwnershipUpdateFinishedWithError, event.ChainID, event.Account, event.Error.Error())
			case <-s.closeCh:
				return
			}
		}
	}()
}

func (s *Service) triggerWalletEvent(eventType walletevent.EventType, chainID walletCommon.ChainID, account common.Address, message string) {
	s.walletFeed.Send(walletevent.Event{
		Type:    eventType,
		ChainID: uint64(chainID),
		Accounts: []common.Address{
			account,
		},
		Message: message,
	})
}
