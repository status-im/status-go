package activityfetcher

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/rpc"
	network2 "github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	activityFetchInterval = 10 * time.Minute
)

const (
	EventActivityFetchComplete walletevent.EventType = "wallet-activity-fetch-complete"
)

// EventActivityDetected is published when new activity entries are detected for an account/chainID
type EventActivityDetected struct {
	ChainID uint64
	Account gethcommon.Address
}

type fetcherID struct {
	account gethcommon.Address
	chainID uint64
}

type Service struct {
	activityFetcherManager ManagerIface
	networksGetter         network2.GetterInterface
	accountsGetter         accounts.AccountsStorage
	ethClientGetter        rpc.EthClientGetter

	networksPublisher         *pubsub.Publisher
	accountsPublisher         *pubsub.Publisher
	activityDetectedPublisher *pubsub.Publisher
	eventFeed                 *event.Feed
	checkRefetchCh            chan bool

	cancelFnMap      map[fetcherID]context.CancelFunc
	cancelFnMapMutex sync.RWMutex

	logger        *zap.Logger
	stopCh        chan struct{}
	subscriptions event.Subscription
	ch            chan walletevent.Event
}

func NewService(
	activityFetcherManager ManagerIface,
	networksGetter network2.GetterInterface,
	accountsGetter accounts.AccountsStorage,
	accountsPublisher *pubsub.Publisher,
	ethClientGetter rpc.EthClientGetter,
	eventFeed *event.Feed,
) *Service {
	logger := logutils.ZapLogger().Named("ActivityFetcher")

	service := &Service{
		activityFetcherManager:    activityFetcherManager,
		networksGetter:            networksGetter,
		accountsGetter:            accountsGetter,
		ethClientGetter:           ethClientGetter,
		networksPublisher:         networksGetter.GetPublisher(),
		accountsPublisher:         accountsPublisher,
		activityDetectedPublisher: pubsub.NewPublisher(),
		eventFeed:                 eventFeed,
		checkRefetchCh:            make(chan bool),
		cancelFnMap:               make(map[fetcherID]context.CancelFunc),
		logger:                    logger,
		ch:                        make(chan walletevent.Event, 100),
	}

	return service
}

// GetPublisher returns the publisher for activity detected events
func (s *Service) GetPublisher() *pubsub.Publisher {
	return s.activityDetectedPublisher
}

// GetActivityTokens returns unique tokens from activity transfers for a given account and chain
func (s *Service) GetActivityTokens(chainID uint64, address gethcommon.Address) ([]ActivityToken, error) {
	return s.activityFetcherManager.GetActivityTokens(chainID, address)
}

func (s *Service) startNetworkWatcher() {
	if s.networksPublisher == nil {
		return
	}

	chNetworkChange, unsubFnNetworkChange := pubsub.Subscribe[network2.EventActiveNetworksChanged](s.networksPublisher, 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsubFnNetworkChange()
		for {
			select {
			case <-s.stopCh:
				return
			case _, ok := <-chNetworkChange:
				if !ok {
					return
				}
				s.triggerRefetch(false)
			}
		}
	}()
}

func (s *Service) startTransactionWatcher() {
	if s.eventFeed == nil {
		return
	}

	s.subscriptions = s.eventFeed.Subscribe(s.ch)
	go func() {
		defer gocommon.LogOnPanic()
		for {
			select {
			case <-s.stopCh:
				return
			case event, ok := <-s.ch:
				if !ok {
					return
				}
				s.handleWalletEvent(event)
			}
		}
	}()
}

func (s *Service) handleWalletEvent(event walletevent.Event) {
	switch event.Type {
	case pendingtxtracker.EventPendingTransactionStatusChanged:
		var payload pendingtxtracker.StatusChangedPayload
		if err := json.Unmarshal([]byte(event.Message), &payload); err != nil {
			s.logger.Error("Failed to extract transaction status payload", zap.Error(err))
			return
		}

		// Trigger immediate fetch when transaction succeeds - bypass interval check
		if payload.Status == ac.Success {
			s.triggerRefetch(true)
		}
	}
}

func (s *Service) startAccountWatcher() {
	if s.accountsPublisher == nil {
		return
	}

	chAdded, unsubFnAdded := pubsub.Subscribe[accountsevent.AccountsAddedEvent](s.accountsPublisher, 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsubFnAdded()
		for {
			select {
			case <-s.stopCh:
				return
			case _, ok := <-chAdded:
				if !ok {
					return
				}
				s.triggerRefetch(false)
			}
		}
	}()

	chRemoved, unsubFnRemoved := pubsub.Subscribe[accountsevent.AccountsRemovedEvent](s.accountsPublisher, 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsubFnRemoved()
		for {
			select {
			case <-s.stopCh:
				return
			case _, ok := <-chRemoved:
				if !ok {
					return
				}
				s.triggerRefetch(false)
			}
		}
	}()
}

func (s *Service) Start(ctx context.Context) {
	s.logger.Info("Starting activity fetcher")
	s.stopCh = make(chan struct{})

	s.startNetworkWatcher()
	s.startAccountWatcher()
	s.startTransactionWatcher()

	go func() {
		defer gocommon.LogOnPanic()

		ticker := time.NewTicker(activityFetchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.stop()
				return
			case <-ticker.C:
				s.fetchActivityForAllAccountsAndChains(ctx, false)
			case bypassIntervalCheck := <-s.checkRefetchCh:
				s.fetchActivityForAllAccountsAndChains(ctx, !bypassIntervalCheck)
			}
		}
	}()

	// Initial fetch on service start
	s.triggerRefetch(false)
}

func (s *Service) triggerRefetch(bypassIntervalCheck bool) {
	s.checkRefetchCh <- bypassIntervalCheck
}

func (s *Service) stop() {
	s.logger.Info("Stopping activity fetcher")

	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}

	if s.subscriptions != nil {
		s.subscriptions.Unsubscribe()
	}

	s.removeAllCancelFns()
}

func listDiff[T comparable](a, b []T) (onlyA, onlyB, both []T) {
	onlyA = make([]T, 0, len(a))
	onlyB = make([]T, 0, len(b))
	both = make([]T, 0, len(a))

	aMap := make(map[T]struct{}, len(a))
	bMap := make(map[T]struct{}, len(b))

	for _, a := range a {
		aMap[a] = struct{}{}
	}
	for _, b := range b {
		bMap[b] = struct{}{}
	}

	for a := range aMap {
		if _, ok := bMap[a]; ok {
			both = append(both, a)
		} else {
			onlyA = append(onlyA, a)
		}
	}

	for b := range bMap {
		if _, ok := aMap[b]; !ok {
			onlyB = append(onlyB, b)
		}
	}

	return onlyA, onlyB, both
}

func (s *Service) cancelFetcherIDs(fetcherIDs []fetcherID) {
	s.cancelFnMapMutex.Lock()
	defer s.cancelFnMapMutex.Unlock()

	for _, fetcherID := range fetcherIDs {
		if cancelFn, ok := s.cancelFnMap[fetcherID]; ok {
			cancelFn()
			delete(s.cancelFnMap, fetcherID)
		}
	}
}

func (s *Service) getCurrentChainIDs() ([]uint64, error) {
	networks, err := s.networksGetter.GetActiveNetworks()
	if err != nil {
		return nil, err
	}
	chainIDs := make([]uint64, 0, len(networks))
	for _, network := range networks {
		chainIDs = append(chainIDs, network.ChainID)
	}
	return chainIDs, nil
}

func (s *Service) getCurrentAccountsAddresses() ([]gethcommon.Address, error) {
	accounts, err := s.accountsGetter.GetWalletAddresses()
	if err != nil {
		return nil, err
	}
	addresses := make([]gethcommon.Address, 0, len(accounts))
	for _, account := range accounts {
		addresses = append(addresses, gethcommon.Address(account))
	}
	return addresses, nil
}

func (s *Service) getDesiredFetcherIDs() ([]fetcherID, error) {
	chainIDs, err := s.getCurrentChainIDs()
	if err != nil {
		return nil, err
	}
	accounts, err := s.getCurrentAccountsAddresses()
	if err != nil {
		return nil, err
	}
	desiredFetcherIDs := make([]fetcherID, 0, len(chainIDs)*len(accounts))
	for _, chainID := range chainIDs {
		for _, account := range accounts {
			desiredFetcherIDs = append(desiredFetcherIDs, fetcherID{account, chainID})
		}
	}
	return desiredFetcherIDs, nil
}

func (s *Service) getRunningFetcherIDs() ([]fetcherID, error) {
	s.cancelFnMapMutex.RLock()
	defer s.cancelFnMapMutex.RUnlock()

	fetcherIDs := make([]fetcherID, 0, len(s.cancelFnMap))
	for fetcherID := range s.cancelFnMap {
		fetcherIDs = append(fetcherIDs, fetcherID)
	}
	return fetcherIDs, nil
}

func (s *Service) fetchActivityForAllAccountsAndChains(ctx context.Context, checkLastTimestamp bool) {
	desiredFetcherIDs, err := s.getDesiredFetcherIDs()
	if err != nil {
		s.logger.Error("Failed to get desired fetcher IDs", zap.Error(err))
		return
	}

	runningFetcherIDs, err := s.getRunningFetcherIDs()
	if err != nil {
		s.logger.Error("Failed to get running fetcher IDs", zap.Error(err))
		return
	}

	idsToAdd, idsToRemove, _ := listDiff(desiredFetcherIDs, runningFetcherIDs)

	// Cancel any fetcherIDs that are no longer in the list
	s.cancelFetcherIDs(idsToRemove)

	// Start fetching activity for all fetcherIDs
	for _, fetcherID := range idsToAdd {
		if !s.activityFetcherManager.IsChainSupported(fetcherID.chainID) {
			s.logger.Debug("Activity fetcher does not support chain", zap.Uint64("chainID", fetcherID.chainID))
			continue
		}

		if checkLastTimestamp {
			_, lastFetchedTimestamp, err := s.activityFetcherManager.GetLastFetchedBlockAndTimestamp(ctx, fetcherID.chainID, fetcherID.account)
			if err != nil {
				s.logger.Error("Failed to get last fetched block and timestamp", zap.Error(err))
				continue
			}

			if lastFetchedTimestamp != nil && time.Since(*lastFetchedTimestamp) < activityFetchInterval {
				s.logger.Debug("Skipping fetch for fetcherID", zap.Uint64("chainID", fetcherID.chainID), zap.String("account", fetcherID.account.Hex()))
				continue
			}
		}

		s.startFetchActivity(ctx, fetcherID)
	}
}

// Upserts the cancel function for a given account and chainID
// If the task is running, it will be cancelled before creating a new one
func (s *Service) upsertCancelFn(ctx context.Context, fetcherID fetcherID) context.Context {
	// If the task is running, cancel it before creating a new one
	s.runAndRemoveCancelFn(fetcherID)

	s.cancelFnMapMutex.Lock()
	defer s.cancelFnMapMutex.Unlock()

	tmpCtx, cancelFn := context.WithCancel(ctx)
	s.cancelFnMap[fetcherID] = cancelFn
	return tmpCtx
}

// Removes the cancel function for a given account and chainID, running it if it exists
// Also ensures proper initialization of the map
func (s *Service) runAndRemoveCancelFn(fetcherID fetcherID) {
	s.cancelFnMapMutex.Lock()
	defer s.cancelFnMapMutex.Unlock()

	var cancelFn context.CancelFunc
	var ok bool
	if cancelFn, ok = s.cancelFnMap[fetcherID]; !ok {
		return
	}

	cancelFn()
	delete(s.cancelFnMap, fetcherID)

	// Check if all fetchers have completed
	if len(s.cancelFnMap) == 0 {
		s.emitFetchComplete()
	}
}

func (s *Service) emitFetchComplete() {
	// Emit event to notify that activity fetch is complete
	// This happens both on initial fetch and periodic fetches
	if s.eventFeed != nil {
		s.eventFeed.Send(walletevent.Event{
			Type:    EventActivityFetchComplete,
			Message: "{}",
		})
	}
}

func (s *Service) removeAllCancelFns() {
	s.cancelFnMapMutex.Lock()
	defer s.cancelFnMapMutex.Unlock()

	for fetcherID := range s.cancelFnMap {
		s.cancelFnMap[fetcherID]()
		delete(s.cancelFnMap, fetcherID)
	}
}

// Starts or restarts fetching activity for a given account and chainID
func (s *Service) startFetchActivity(ctx context.Context, fetcherID fetcherID) {
	tmpCtx := s.upsertCancelFn(ctx, fetcherID)

	go func() {
		defer gocommon.LogOnPanic()
		defer s.runAndRemoveCancelFn(fetcherID)
		s.fetchActivity(tmpCtx, fetcherID.chainID, fetcherID.account)
	}()
}

func (s *Service) fetchActivity(ctx context.Context, chainID uint64, account gethcommon.Address) {

	// Get current block
	ethClient, err := s.ethClientGetter.EthClient(chainID)
	if err != nil {
		s.logger.Error("Failed to get rpc client", zap.Error(err))
		return
	}
	currentBlock, err := ethClient.BlockNumber(ctx)
	if err != nil {
		s.logger.Error("Failed to get current block", zap.Error(err))
		return
	}

	_, err = s.activityFetcherManager.FetchActivity(ctx, chainID, account, currentBlock)
	if err != nil {
		s.logger.Error("Failed to fetch activity", zap.Error(err))
		return
	}

	// Publish event that new activity entries were detected
	if s.activityDetectedPublisher != nil {
		pubsub.Publish(s.activityDetectedPublisher, EventActivityDetected{
			ChainID: chainID,
			Account: account,
		})
	}
}

func (s *Service) RefetchTxHistory() error {
	s.logger.Info("Refetching tx history")

	err := s.activityFetcherManager.ClearAll(context.Background())
	if err != nil {
		return err
	}

	// Trigger a refetch
	s.triggerRefetch(true)

	return nil
}
