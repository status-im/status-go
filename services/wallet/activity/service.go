package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/collectibles"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"
)

const (
	// EventActivityFilteringDone contains a FilterResponse payload
	EventActivityFilteringDone          walletevent.EventType = "wallet-activity-filtering-done"
	EventActivityFilteringUpdate        walletevent.EventType = "wallet-activity-filtering-entries-updated"
	EventActivityGetRecipientsDone      walletevent.EventType = "wallet-activity-get-recipients-result"
	EventActivityGetOldestTimestampDone walletevent.EventType = "wallet-activity-get-oldest-timestamp-result"
	EventActivityGetCollectibles        walletevent.EventType = "wallet-activity-get-collectibles"

	// EventActivitySessionUpdated contains a SessionUpdate payload
	EventActivitySessionUpdated walletevent.EventType = "wallet-activity-session-updated"
)

var (
	filterTask = async.TaskType{
		ID:     1,
		Policy: async.ReplacementPolicyCancelOld,
	}
	getRecipientsTask = async.TaskType{
		ID:     2,
		Policy: async.ReplacementPolicyIgnoreNew,
	}
	getOldestTimestampTask = async.TaskType{
		ID:     3,
		Policy: async.ReplacementPolicyCancelOld,
	}
	getCollectiblesTask = async.TaskType{
		ID:     4,
		Policy: async.ReplacementPolicyCancelOld,
	}
)

// Service provides an async interface, ensuring only one filter request, of each type, is running at a time. It also provides lazy load of NFT info and token mapping
type Service struct {
	db           *sql.DB
	accountsDB   *accounts.Database
	tokenManager token.ManagerInterface
	collectibles collectibles.ManagerInterface
	eventFeed    *event.Feed

	scheduler *async.MultiClientScheduler

	sessions              map[SessionID]*Session
	lastSessionID         atomic.Int32
	subscriptions         event.Subscription
	subscriptionsCancelFn context.CancelFunc
	ch                    chan walletevent.Event
	// sessionsRWMutex is used to protect all sessions related members
	sessionsRWMutex  sync.RWMutex
	debounceDuration time.Duration

	pendingTracker *transactions.PendingTxTracker
}

func (s *Service) nextSessionID() SessionID {
	return SessionID(s.lastSessionID.Add(1))
}

func NewService(db *sql.DB, accountsDB *accounts.Database, tokenManager token.ManagerInterface, collectibles collectibles.ManagerInterface, eventFeed *event.Feed, pendingTracker *transactions.PendingTxTracker) *Service {
	return &Service{
		db:           db,
		accountsDB:   accountsDB,
		tokenManager: tokenManager,
		collectibles: collectibles,
		eventFeed:    eventFeed,
		scheduler:    async.NewMultiClientScheduler(),

		sessions: make(map[SessionID]*Session),
		// here to be overwritten by tests
		debounceDuration: 1 * time.Second,

		pendingTracker: pendingTracker,
	}
}

type ErrorCode = int

const (
	ErrorCodeSuccess ErrorCode = iota + 1
	ErrorCodeTaskCanceled
	ErrorCodeFailed
)

type FilterResponse struct {
	Activities []Entry `json:"activities"`
	Offset     int     `json:"offset"`
	// Used to indicate that there might be more entries that were not returned
	// based on a simple heuristic
	HasMore   bool      `json:"hasMore"`
	ErrorCode ErrorCode `json:"errorCode"`
}

// FilterActivityAsync allows only one filter task to run at a time
// it cancels the current one if a new one is started
// and should not expect other owners to have data in one of the queried tables
//
// All calls will trigger an EventActivityFilteringDone event with the result of the filtering
// TODO #12120: replace with session based APIs
func (s *Service) FilterActivityAsync(requestID int32, addresses []common.Address, chainIDs []w_common.ChainID, filter Filter, offset int, limit int) {
	s.scheduler.Enqueue(requestID, filterTask, func(ctx context.Context) (interface{}, error) {
		allAddresses := s.areAllAddresses(addresses)
		activities, err := getActivityEntries(ctx, s.getDeps(), addresses, allAddresses, chainIDs, filter, offset, limit)
		return activities, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := FilterResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			activities := result.([]Entry)
			res.Activities = activities
			res.Offset = offset
			res.HasMore = len(activities) == limit
			res.ErrorCode = ErrorCodeSuccess
		}

		sendResponseEvent(s.eventFeed, &requestID, EventActivityFilteringDone, res, err)

		s.getActivityDetailsAsync(requestID, res.Activities)
	})
}

type CollectibleHeader struct {
	ID       thirdparty.CollectibleUniqueID `json:"id"`
	Name     string                         `json:"name"`
	ImageURL string                         `json:"image_url"`
}

type GetollectiblesResponse struct {
	Collectibles []CollectibleHeader `json:"collectibles"`
	Offset       int                 `json:"offset"`
	// Used to indicate that there might be more collectibles that were not returned
	// based on a simple heuristic
	HasMore   bool      `json:"hasMore"`
	ErrorCode ErrorCode `json:"errorCode"`
}

func (s *Service) GetActivityCollectiblesAsync(requestID int32, chainIDs []w_common.ChainID, addresses []common.Address, offset int, limit int) {
	s.scheduler.Enqueue(requestID, getCollectiblesTask, func(ctx context.Context) (interface{}, error) {
		collectibles, err := GetActivityCollectibles(ctx, s.db, chainIDs, addresses, offset, limit)

		if err != nil {
			return nil, err
		}

		data, err := s.collectibles.FetchAssetsByCollectibleUniqueID(ctx, collectibles, true)
		if err != nil {
			return nil, err
		}

		res := make([]CollectibleHeader, 0, len(data))

		for _, c := range data {
			res = append(res, CollectibleHeader{
				ID:       c.CollectibleData.ID,
				Name:     c.CollectibleData.Name,
				ImageURL: c.CollectibleData.ImageURL,
			})
		}

		return res, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := GetollectiblesResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			collectibles := result.([]CollectibleHeader)
			res.Collectibles = collectibles
			res.Offset = offset
			res.HasMore = len(collectibles) == limit
			res.ErrorCode = ErrorCodeSuccess
		}

		sendResponseEvent(s.eventFeed, &requestID, EventActivityGetCollectibles, res, err)
	})
}

func (s *Service) GetMultiTxDetails(ctx context.Context, multiTxID int) (*EntryDetails, error) {
	return getMultiTxDetails(ctx, s.db, multiTxID)
}

func (s *Service) GetTxDetails(ctx context.Context, id string) (*EntryDetails, error) {
	return getTxDetails(ctx, s.db, id)
}

// getActivityDetails check if any of the entries have details that are not loaded then fetch and emit result
func (s *Service) getActivityDetails(ctx context.Context, entries []Entry) ([]*EntryData, error) {
	res := make([]*EntryData, 0)
	var err error
	ids := make([]thirdparty.CollectibleUniqueID, 0)
	entriesForIds := make(map[string][]*Entry)

	idExists := func(ids []thirdparty.CollectibleUniqueID, id *thirdparty.CollectibleUniqueID) bool {
		for _, existingID := range ids {
			if existingID.Same(id) {
				return true
			}
		}
		return false
	}

	for i := range entries {
		if !entries[i].isNFT() {
			continue
		}

		id := entries[i].anyIdentity()
		if id == nil {
			continue
		}

		entriesForIds[id.HashKey()] = append(entriesForIds[id.HashKey()], &entries[i])
		if !idExists(ids, id) {
			ids = append(ids, *id)
		}
	}

	if len(ids) == 0 {
		return nil, nil
	}

	logutils.ZapLogger().Debug("wallet.activity.Service lazyLoadDetails",
		zap.Int("entries.len", len(entries)),
		zap.Int("ids.len", len(ids)),
	)

	colData, err := s.collectibles.FetchAssetsByCollectibleUniqueID(ctx, ids, true)
	if err != nil {
		logutils.ZapLogger().Error("Error fetching collectible details", zap.Error(err))
		return nil, err
	}

	for _, col := range colData {
		nftName := w_common.NewAndSet(col.CollectibleData.Name)
		nftURL := w_common.NewAndSet(col.CollectibleData.ImageURL)
		for i := range ids {
			if !col.CollectibleData.ID.Same(&ids[i]) {
				continue
			}

			entryList, ok := entriesForIds[ids[i].HashKey()]
			if !ok {
				continue
			}
			for _, e := range entryList {
				data := &EntryData{
					Key:     e.Key(),
					NftName: nftName,
					NftURL:  nftURL,
				}
				if e.payloadType == MultiTransactionPT {
					data.ID = w_common.NewAndSet(e.id)
				} else {
					data.Transaction = e.transaction
				}

				data.PayloadType = e.payloadType
				res = append(res, data)
			}
		}
	}
	return res, nil
}

type GetRecipientsResponse struct {
	Addresses []common.Address `json:"addresses"`
	Offset    int              `json:"offset"`
	// Used to indicate that there might be more entries that were not returned
	// based on a simple heuristic
	HasMore   bool      `json:"hasMore"`
	ErrorCode ErrorCode `json:"errorCode"`
}

// GetRecipientsAsync returns true if a task is already running or scheduled due to a previous call; meaning that
// this call won't receive an answer but client should rely on the answer from the previous call.
// If no task is already scheduled false will be returned
func (s *Service) GetRecipientsAsync(requestID int32, chainIDs []w_common.ChainID, addresses []common.Address, offset int, limit int) bool {
	return s.scheduler.Enqueue(requestID, getRecipientsTask, func(ctx context.Context) (interface{}, error) {
		var err error
		result := &GetRecipientsResponse{
			Offset:    offset,
			ErrorCode: ErrorCodeSuccess,
		}
		result.Addresses, result.HasMore, err = GetRecipients(ctx, s.db, chainIDs, addresses, offset, limit)
		return result, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := result.(*GetRecipientsResponse)
		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err != nil {
			res.ErrorCode = ErrorCodeFailed
		}

		sendResponseEvent(s.eventFeed, &requestID, EventActivityGetRecipientsDone, result, err)
	})
}

type GetOldestTimestampResponse struct {
	Timestamp int64     `json:"timestamp"`
	ErrorCode ErrorCode `json:"errorCode"`
}

func (s *Service) GetOldestTimestampAsync(requestID int32, addresses []common.Address) {
	s.scheduler.Enqueue(requestID, getOldestTimestampTask, func(ctx context.Context) (interface{}, error) {
		timestamp, err := GetOldestTimestamp(ctx, s.db, addresses)
		return timestamp, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := GetOldestTimestampResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			res.Timestamp = int64(result.(uint64))
			res.ErrorCode = ErrorCodeSuccess
		}

		sendResponseEvent(s.eventFeed, &requestID, EventActivityGetOldestTimestampDone, res, err)
	})
}

func (s *Service) CancelFilterTask(requestID int32) {
	s.scheduler.Enqueue(requestID, filterTask, func(ctx context.Context) (interface{}, error) {
		// No-op
		return nil, nil
	}, func(result interface{}, taskType async.TaskType, err error) {
		// Ignore result
	})
}

func (s *Service) Stop() {
	s.scheduler.Stop()
}

func (s *Service) getDeps() FilterDependencies {
	return FilterDependencies{
		db: s.db,
		tokenSymbol: func(t Token) string {
			info := s.tokenManager.LookupTokenIdentity(uint64(t.ChainID), t.Address, t.TokenType == Native)
			if info == nil {
				return ""
			}
			return info.Symbol
		},
		tokenFromSymbol: func(chainID *w_common.ChainID, symbol string) *Token {
			var cID *uint64
			if chainID != nil {
				cID = new(uint64)
				*cID = uint64(*chainID)
			}
			t, detectedNative := s.tokenManager.LookupToken(cID, symbol)
			if t == nil {
				return nil
			}
			tokenType := Native
			if !detectedNative {
				tokenType = Erc20
			}
			return &Token{
				TokenType: tokenType,
				ChainID:   w_common.ChainID(t.ChainID),
				Address:   t.Address,
			}
		},
		currentTimestamp: func() int64 {
			return time.Now().Unix()
		},
	}
}

func sendResponseEvent(eventFeed *event.Feed, requestID *int32, eventType walletevent.EventType, payloadObj interface{}, resErr error) {
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		logutils.ZapLogger().Error("Error marshaling", zap.NamedError("response", err), zap.NamedError("result", resErr))
	} else {
		err = resErr
	}

	requestIDStr := nilStr
	if requestID != nil {
		requestIDStr = strconv.Itoa(int(*requestID))
	}
	logutils.ZapLogger().Debug("wallet.api.activity.Service RESPONSE",
		zap.String("requestID", requestIDStr),
		zap.String("eventType", string(eventType)),
		zap.Error(err),
		zap.Int("payload.len", len(payload)),
	)

	event := walletevent.Event{
		Type:    eventType,
		Message: string(payload),
	}

	if requestID != nil {
		event.RequestID = new(int)
		*event.RequestID = int(*requestID)
	}

	eventFeed.Send(event)
}

func (s *Service) getWalletAddreses() ([]common.Address, error) {
	ethAddresses, err := s.accountsDB.GetWalletAddresses()
	if err != nil {
		return nil, err
	}

	addresses := make([]common.Address, 0, len(ethAddresses))
	for _, ethAddress := range ethAddresses {
		addresses = append(addresses, common.Address(ethAddress))
	}

	return addresses, nil
}

func (s *Service) areAllAddresses(addresses []common.Address) bool {
	// Compare with addresses in accountsDB
	walletAddresses, err := s.getWalletAddreses()
	if err != nil {
		logutils.ZapLogger().Error("Error getting wallet addresses", zap.Error(err))
		return false
	}

	// Check if passed addresses are the same as in the accountsDB ignoring the order
	return areSlicesEqual(walletAddresses, addresses)
}

// Comparison function to check if slices are the same ignoring the order
func areSlicesEqual(a, b []common.Address) bool {
	if len(a) != len(b) {
		return false
	}

	// Create a map of addresses
	aMap := make(map[common.Address]struct{}, len(a))
	for _, address := range a {
		aMap[address] = struct{}{}
	}

	// Check if all passed addresses are in the map
	for _, address := range b {
		if _, ok := aMap[address]; !ok {
			return false
		}
	}

	return true
}
