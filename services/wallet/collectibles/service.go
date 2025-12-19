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
	EventCommunityCollectiblesReceived                walletevent.EventType = "wallet-collectibles-community-collectibles-received"
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

func (s *Service) needsToFetch(chainID walletCommon.ChainID, address common.Address, fetchCriteria FetchCriteria) (bool, error) {
	mustFetch := false
	switch fetchCriteria.FetchType {
	case FetchTypeAlwaysFetch:
		mustFetch = true
	case FetchTypeNeverFetch:
		mustFetch = false
	case FetchTypeFetchIfNotCached, FetchTypeFetchIfCacheOld:
		timestamp, err := s.ownershipDB.GetOwnershipUpdateTimestamp(address, chainID)
		if err != nil {
			return false, err
		}
		if timestamp == ownership.InvalidTimestamp ||
			(fetchCriteria.FetchType == FetchTypeFetchIfCacheOld && timestamp+fetchCriteria.MaxCacheAgeSeconds < time.Now().Unix()) {
			mustFetch = true
		}
	}
	return mustFetch, nil
}

func (s *Service) fetchOwnedCollectiblesIfNeeded(ctx context.Context, chainIDs []walletCommon.ChainID, addresses []common.Address, fetchCriteria FetchCriteria) error {
	if fetchCriteria.FetchType == FetchTypeNeverFetch {
		return nil
	}

	wg := sync.WaitGroup{}
	wgErr := atomic.NewError(nil)
	for _, address := range addresses {
		for _, chainID := range chainIDs {
			mustFetch, err := s.needsToFetch(chainID, address, fetchCriteria)
			if err != nil {
				return err
			}
			if mustFetch {
				wg.Add(1)
				go func() {
					defer gocommon.LogOnPanic()
					defer wg.Done()
					err := s.ownershipController.TriggerLoad(ctx, chainID, address)
					if err != nil {
						wgErr.Store(err)
					}
				}()
			}
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

func (s *Service) notifyCommunityCollectiblesReceived(account common.Address, chainID walletCommon.ChainID, collectibles []thirdparty.CollectibleUniqueID, hashMap map[thirdparty.CollectibleUniqueID]TxHashData) {
	ctx := context.Background()

	firstCollectibles, err := s.ownershipDB.GetIsFirstOfCollection(account, collectibles)
	if err != nil {
		return
	}

	collectiblesData, err := s.manager.FetchAssetsByCollectibleUniqueID(ctx, collectibles, false)
	if err != nil {
		s.logger.Error("Error fetching collectibles data", zap.Error(err))
		return
	}

	communityCollectibles := fullCollectiblesDataToCommunityHeader(collectiblesData)

	if len(communityCollectibles) == 0 {
		return
	}

	type CollectibleGroup struct {
		contractID thirdparty.ContractID
		txHash     string
	}

	groups := make(map[CollectibleGroup]Collectible)
	for _, localCollectible := range communityCollectibles {
		// to satisfy gosec: C601 checks
		collectible := localCollectible
		txHash := ""
		for key, value := range hashMap {
			if key.Same(&collectible.ID) {
				collectible.LatestTxHash = value.TxID.Hex()
				txHash = value.Hash.Hex()
				break
			}
		}

		for id, value := range firstCollectibles {
			if value && id.Same(&collectible.ID) {
				collectible.IsFirst = true
				break
			}
		}

		group := CollectibleGroup{
			contractID: collectible.ID.ContractID,
			txHash:     txHash,
		}
		_, ok := groups[group]
		if !ok {
			collectible.ReceivedAmount = float64(0)
		}
		collectible.ReceivedAmount = collectible.ReceivedAmount + 1
		groups[group] = collectible
	}

	groupedCommunityCollectibles := make([]Collectible, 0, len(groups))
	for _, collectible := range groups {
		groupedCommunityCollectibles = append(groupedCommunityCollectibles, collectible)
	}

	encodedMessage, err := json.Marshal(groupedCommunityCollectibles)
	if err != nil {
		return
	}

	s.walletFeed.Send(walletevent.Event{
		Type:    EventCommunityCollectiblesReceived,
		ChainID: uint64(chainID),
		Accounts: []common.Address{
			account,
		},
		Message: string(encodedMessage),
	})
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
