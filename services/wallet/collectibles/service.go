package collectibles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"

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
	EventCollectiblesOwnershipUpdatePartial           walletevent.EventType = "wallet-collectibles-ownership-update-partial"
	EventCollectiblesOwnershipUpdateFinished          walletevent.EventType = "wallet-collectibles-ownership-update-finished"
	EventCollectiblesOwnershipUpdateFinishedWithError walletevent.EventType = "wallet-collectibles-ownership-update-finished-with-error"

	EventOwnedCollectiblesFilteringDone walletevent.EventType = "wallet-owned-collectibles-filtering-done"
	EventGetCollectiblesDetailsDone     walletevent.EventType = "wallet-get-collectibles-details-done"
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

type commandPerChainID = map[walletCommon.ChainID]*periodicRefreshOwnedCollectiblesCommand
type commandPerAddressAndChainID = map[common.Address]commandPerChainID

type Service struct {
	manager      *Manager
	ownershipDB  *OwnershipDB
	walletFeed   *event.Feed
	accountsDB   *accounts.Database
	accountsFeed *event.Feed

	networkManager *network.Manager
	cancelFn       context.CancelFunc

	commands        commandPerAddressAndChainID
	group           *async.Group
	scheduler       *async.MultiClientScheduler
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
		commands:       make(commandPerAddressAndChainID),
		scheduler:      async.NewMultiClientScheduler(),
	}
}

type ErrorCode = int

const (
	ErrorCodeSuccess ErrorCode = iota + 1
	ErrorCodeTaskCanceled
	ErrorCodeFailed
)

type OwnershipState = int

const (
	OwnershipStateIdle OwnershipState = iota + 1
	OwnershipStateUpdating
	OwnershipStateError
)

type OwnershipStatus struct {
	State     OwnershipState `json:"state"`
	Timestamp int64          `json:"timestamp"`
}

type OwnershipStatusPerChainID = map[walletCommon.ChainID]OwnershipStatus
type OwnershipStatusPerAddressAndChainID = map[common.Address]OwnershipStatusPerChainID

type FilterOwnedCollectiblesResponse struct {
	Collectibles []CollectibleHeader `json:"collectibles"`
	Offset       int                 `json:"offset"`
	// Used to indicate that there might be more collectibles that were not returned
	// based on a simple heuristic
	HasMore         bool                                `json:"hasMore"`
	OwnershipStatus OwnershipStatusPerAddressAndChainID `json:"ownershipStatus"`
	ErrorCode       ErrorCode                           `json:"errorCode"`
}

type GetCollectiblesDetailsResponse struct {
	Collectibles []CollectibleDetails `json:"collectibles"`
	ErrorCode    ErrorCode            `json:"errorCode"`
}

type filterOwnedCollectiblesTaskReturnType struct {
	collectibles    []CollectibleHeader
	hasMore         bool
	ownershipStatus OwnershipStatusPerAddressAndChainID
}

// FilterOwnedCollectiblesResponse allows only one filter task to run at a time
// and it cancels the current one if a new one is started
// All calls will trigger an EventOwnedCollectiblesFilteringDone event with the result of the filtering
func (s *Service) FilterOwnedCollectiblesAsync(requestID int32, chainIDs []walletCommon.ChainID, addresses []common.Address, offset int, limit int) {
	s.scheduler.Enqueue(requestID, filterOwnedCollectiblesTask, func(ctx context.Context) (interface{}, error) {
		collectibles, hasMore, err := s.GetOwnedCollectibles(chainIDs, addresses, offset, limit)
		if err != nil {
			return nil, err
		}
		data, err := s.manager.FetchAssetsByCollectibleUniqueID(collectibles)
		if err != nil {
			return nil, err
		}
		ownershipStatus, err := s.GetOwnershipStatus(chainIDs, addresses)
		if err != nil {
			return nil, err
		}

		return filterOwnedCollectiblesTaskReturnType{
			collectibles:    fullCollectiblesDataToHeaders(data),
			hasMore:         hasMore,
			ownershipStatus: ownershipStatus,
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
			res.OwnershipStatus = fnRet.ownershipStatus
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(&requestID, EventOwnedCollectiblesFilteringDone, res, err)
	})
}

func (s *Service) GetCollectiblesDetailsAsync(requestID int32, uniqueIDs []thirdparty.CollectibleUniqueID) {
	s.scheduler.Enqueue(requestID, getCollectiblesDataTask, func(ctx context.Context) (interface{}, error) {
		collectibles, err := s.manager.FetchAssetsByCollectibleUniqueID(uniqueIDs)
		return collectibles, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := GetCollectiblesDetailsResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			collectibles := result.([]thirdparty.FullCollectibleData)
			res.Collectibles = fullCollectiblesDataToDetails(collectibles)
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(&requestID, EventGetCollectiblesDetailsDone, res, err)
	})
}

// Starts periodical fetching for the all wallet addresses and all chains
func (s *Service) startPeriodicalOwnershipFetch() error {
	if s.group != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel

	s.group = async.NewGroup(ctx)

	addresses, err := s.accountsDB.GetWalletAddresses()
	if err != nil {
		return err
	}

	for _, addr := range addresses {
		err := s.startPeriodicalOwnershipFetchForAccount(common.Address(addr))
		if err != nil {
			log.Error("Error starting periodical collectibles fetch for accpunt", "address", addr, "error", err)
			return err
		}
	}

	return nil
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
		s.commands = make(commandPerAddressAndChainID)
	}
}

// Starts (or restarts) periodical fetching for the given account address for all chains
func (s *Service) startPeriodicalOwnershipFetchForAccount(address common.Address) error {
	if s.group == nil {
		return errors.New("periodical fetch group not initialized")
	}

	networks, err := s.networkManager.Get(false)
	if err != nil {
		return err
	}

	areTestNetworksEnabled, err := s.accountsDB.GetTestNetworksEnabled()
	if err != nil {
		return err
	}

	if _, ok := s.commands[address]; ok {
		for chainID, command := range s.commands[address] {
			command.Stop()
			delete(s.commands[address], chainID)
		}
	}

	s.commands[address] = make(commandPerChainID)

	for _, network := range networks {
		if network.IsTest != areTestNetworksEnabled {
			continue
		}
		chainID := walletCommon.ChainID(network.ChainID)

		command := newPeriodicRefreshOwnedCollectiblesCommand(
			s.manager,
			s.ownershipDB,
			s.walletFeed,
			chainID,
			address,
		)

		s.commands[address][chainID] = command
		s.group.Add(command.Command())
	}

	return nil
}

// Stop periodical fetching for the given account address for all chains
func (s *Service) stopPeriodicalOwnershipFetchForAccount(address common.Address) error {
	if s.group == nil {
		return errors.New("periodical fetch group not initialized")
	}

	if _, ok := s.commands[address]; ok {
		for _, command := range s.commands[address] {
			command.Stop()
		}
		delete(s.commands, address)
	}

	return nil
}

func (s *Service) startAccountsWatcher() {
	if s.accountsWatcher != nil {
		return
	}

	accountChangeCb := func(changedAddresses []common.Address, eventType accountsevent.EventType, currentAddresses []common.Address) {
		// Whenever an account gets added, start fetching
		if eventType == accountsevent.EventTypeAdded {
			for _, address := range changedAddresses {
				err := s.startPeriodicalOwnershipFetchForAccount(address)
				if err != nil {
					log.Error("Error starting periodical collectibles fetch", "address", address, "error", err)
				}
			}
		} else if eventType == accountsevent.EventTypeRemoved {
			for _, address := range changedAddresses {
				err := s.stopPeriodicalOwnershipFetchForAccount(address)
				if err != nil {
					log.Error("Error starting periodical collectibles fetch", "address", address, "error", err)
				}
			}
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
	_ = s.startPeriodicalOwnershipFetch()

	// Setup collectibles fetch when a new account gets added
	s.startAccountsWatcher()
}

func (s *Service) Stop() {
	s.stopAccountsWatcher()

	s.stopPeriodicalOwnershipFetch()

	s.scheduler.Stop()
}

func (s *Service) sendResponseEvent(requestID *int32, eventType walletevent.EventType, payloadObj interface{}, resErr error) {
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		log.Error("Error marshaling response: %v; result error: %w", err, resErr)
	} else {
		err = resErr
	}

	log.Debug("wallet.api.collectibles.Service RESPONSE", "requestID", requestID, "eventType", eventType, "error", err, "payload.len", len(payload))

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
			state := OwnershipStateIdle
			if s.commands[address] != nil && s.commands[address][chainID] != nil {
				state = s.commands[address][chainID].GetState()
			}
			ret[address][chainID] = OwnershipStatus{
				State:     state,
				Timestamp: timestamp,
			}
		}
	}

	return ret, nil
}
