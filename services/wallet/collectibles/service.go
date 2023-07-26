package collectibles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc/network"

	"github.com/status-im/status-go/services/accounts/accountsevent"
	walletaccounts "github.com/status-im/status-go/services/wallet/accounts"
	"github.com/status-im/status-go/services/wallet/async"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// These events are used to notify the UI of state changes
const (
	EventCollectiblesOwnershipUpdateStarted           walletevent.EventType = "wallet-collectibles-ownership-update-started"
	EventCollectiblesOwnershipUpdateFinished          walletevent.EventType = "wallet-collectibles-ownership-update-finished"
	EventCollectiblesOwnershipUpdateFinishedWithError walletevent.EventType = "wallet-collectibles-ownership-update-finished-with-error"

	EventOwnedCollectiblesFilteringDone walletevent.EventType = "wallet-owned-collectibles-filtering-done"
	EventGetCollectiblesDataDone        walletevent.EventType = "wallet-get-collectibles-data-done"
)

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
	manager      *Manager
	ownershipDB  *OwnershipDB
	walletFeed   *event.Feed
	accountsDB   *accounts.Database
	accountsFeed *event.Feed

	networkManager *network.Manager
	cancelFn       context.CancelFunc

	group           *async.Group
	scheduler       *async.Scheduler
	accountsWatcher *walletaccounts.Watcher
}

func NewService(db *sql.DB, walletFeed *event.Feed, accountsDB *accounts.Database, accountsFeed *event.Feed, networkManager *network.Manager, manager *Manager) *Service {
	return &Service{
		manager:        manager,
		ownershipDB:    NewOwnershipDB(db),
		walletFeed:     walletFeed,
		accountsDB:     accountsDB,
		accountsFeed:   accountsFeed,
		networkManager: networkManager,
		scheduler:      async.NewScheduler(),
	}
}

type ErrorCode = int

const (
	ErrorCodeSuccess ErrorCode = iota + 1
	ErrorCodeTaskCanceled
	ErrorCodeFailed
)

type FilterOwnedCollectiblesResponse struct {
	Collectibles []thirdparty.CollectibleHeader `json:"collectibles"`
	Offset       int                            `json:"offset"`
	// Used to indicate that there might be more collectibles that were not returned
	// based on a simple heuristic
	HasMore   bool      `json:"hasMore"`
	ErrorCode ErrorCode `json:"errorCode"`
}

type GetCollectiblesDataResponse struct {
	Collectibles []thirdparty.CollectibleData `json:"collectibles"`
	ErrorCode    ErrorCode                    `json:"errorCode"`
}

type filterOwnedCollectiblesTaskReturnType struct {
	collectibles []thirdparty.CollectibleHeader
	hasMore      bool
}

// FilterOwnedCollectiblesResponse allows only one filter task to run at a time
// and it cancels the current one if a new one is started
// All calls will trigger an EventOwnedCollectiblesFilteringDone event with the result of the filtering
func (s *Service) FilterOwnedCollectiblesAsync(ctx context.Context, chainIDs []walletCommon.ChainID, addresses []common.Address, offset int, limit int) {
	s.scheduler.Enqueue(filterOwnedCollectiblesTask, func(ctx context.Context) (interface{}, error) {
		collectibles, hasMore, err := s.GetOwnedCollectibles(chainIDs, addresses, offset, limit)
		if err != nil {
			return nil, err
		}
		data, err := s.manager.FetchAssetsByCollectibleUniqueID(collectibles)
		if err != nil {
			return nil, err
		}

		return filterOwnedCollectiblesTaskReturnType{
			collectibles: thirdparty.CollectiblesToHeaders(data),
			hasMore:      hasMore,
		}, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := FilterOwnedCollectiblesResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			fnRet := result.(filterOwnedCollectiblesTaskReturnType)
			res.Collectibles = fnRet.collectibles
			res.Offset = offset
			res.HasMore = fnRet.hasMore
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(EventOwnedCollectiblesFilteringDone, res, err)
	})
}

func (s *Service) GetCollectiblesDataAsync(ctx context.Context, uniqueIDs []thirdparty.CollectibleUniqueID) {
	s.scheduler.Enqueue(getCollectiblesDataTask, func(ctx context.Context) (interface{}, error) {
		collectibles, err := s.manager.FetchAssetsByCollectibleUniqueID(uniqueIDs)
		return collectibles, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := GetCollectiblesDataResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			collectibles := result.([]thirdparty.CollectibleData)
			res.Collectibles = collectibles
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(EventGetCollectiblesDataDone, res, err)
	})
}

func (s *Service) startPeriodicalOwnershipFetch() {
	if s.group != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel

	s.group = async.NewGroup(ctx)

	command := newRefreshOwnedCollectiblesCommand(
		s.manager,
		s.ownershipDB,
		s.accountsDB,
		s.walletFeed,
		s.networkManager,
	)

	s.group.Add(command.Command())
}

func (s *Service) stopPeriodicalOwnershipFetch() {
	if s.cancelFn != nil {
		s.cancelFn()
		s.cancelFn = nil
	}
	if s.group != nil {
		s.group.Stop()
		s.group.Wait()
		s.group = nil
	}
}

func (s *Service) startAccountsWatcher() {
	if s.accountsWatcher != nil {
		return
	}

	accountChangeCb := func(changedAddresses []common.Address, eventType accountsevent.EventType, currentAddresses []common.Address) {
		// Whenever an account gets added, restart fetch
		// TODO: Fetch only added accounts
		if eventType == accountsevent.EventTypeAdded {
			s.stopPeriodicalOwnershipFetch()
			s.startPeriodicalOwnershipFetch()
		}
	}

	s.accountsWatcher = walletaccounts.NewWatcher(s.accountsDB, s.accountsFeed, accountChangeCb)

	s.accountsWatcher.Start()
}

func (s *Service) stopAccountsWatcher() {
	if s.accountsWatcher != nil {
		s.accountsWatcher.Stop()
		s.accountsWatcher = nil
	}
}

func (s *Service) Start() {
	// Setup periodical collectibles refresh
	s.startPeriodicalOwnershipFetch()

	// Setup collectibles fetch when a new account gets added
	s.startAccountsWatcher()
}

func (s *Service) Stop() {
	s.stopAccountsWatcher()

	s.stopPeriodicalOwnershipFetch()

	s.scheduler.Stop()
}

func (s *Service) sendResponseEvent(eventType walletevent.EventType, payloadObj interface{}, resErr error) {
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		log.Error("Error marshaling response: %v; result error: %w", err, resErr)
	} else {
		err = resErr
	}

	log.Debug("wallet.api.collectibles.Service RESPONSE", "eventType", eventType, "error", err, "payload.len", len(payload))

	s.walletFeed.Send(walletevent.Event{
		Type:    eventType,
		Message: string(payload),
	})
}

func (s *Service) GetOwnedCollectibles(chainIDs []walletCommon.ChainID, owners []common.Address, offset int, limit int) ([]thirdparty.CollectibleUniqueID, bool, error) {
	// Request one more than limit, to check if DB has more available
	ids, err := s.ownershipDB.GetOwnedCollectibles(chainIDs, owners, offset, limit+1)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(ids) > limit
	if hasMore {
		ids = ids[:limit]
	}

	return ids, hasMore, nil
}
