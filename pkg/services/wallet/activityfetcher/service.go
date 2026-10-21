package activityfetcher

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"
	"github.com/status-im/status-go/pkg/services/networks"
	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	"github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

const (
	activityFetchInterval = 10 * time.Minute
)

const (
	EventActivityFetchComplete        walletevent.EventType = "wallet-activity-fetch-complete"
	EventActivityNewTransfersDetected walletevent.EventType = "wallet-activity-new-transfers-detected"
)

type fetcherID struct {
	account gethcommon.Address
	chainID uint64
}

// FetchKey names the history of one account on one chain.
type FetchKey struct {
	Address gethcommon.Address
	ChainID uint64
}

// fetchRun is one fetch of one fetcherID. done is closed when it ends, err is set before that.
type fetchRun struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

type Service struct {
	activityFetcherManager ManagerIface
	networksGetter         networks.GetterInterface
	accountsGetter         accounts.AccountsStorage
	ethClientGetter        rpc.EthClientGetter

	networksPublisher             *pubsub.Publisher
	accountsPublisher             *pubsub.Publisher
	multistandardBalancePublisher *pubsub.Publisher
	eventFeed                     *event.Feed
	checkRefetchCh                chan bool

	cancelFnMap      map[fetcherID]*fetchRun
	cancelFnMapMutex sync.RWMutex
	ctx              context.Context

	logger          *zap.Logger
	stopCh          chan struct{}
	subscriptions   event.Subscription
	ch              chan walletevent.Event
	targetedFetchCh chan []fetcherID
}

func NewService(
	activityFetcherManager ManagerIface,
	networksGetter networks.GetterInterface,
	accountsGetter accounts.AccountsStorage,
	accountsPublisher *pubsub.Publisher,
	ethClientGetter rpc.EthClientGetter,
	eventFeed *event.Feed,
	multistandardBalancePublisher *pubsub.Publisher,
) *Service {
	logger := logutils.ZapLogger().Named("ActivityFetcher")

	service := &Service{
		activityFetcherManager:        activityFetcherManager,
		networksGetter:                networksGetter,
		accountsGetter:                accountsGetter,
		ethClientGetter:               ethClientGetter,
		networksPublisher:             networksGetter.GetPublisher(),
		accountsPublisher:             accountsPublisher,
		multistandardBalancePublisher: multistandardBalancePublisher,
		eventFeed:                     eventFeed,
		checkRefetchCh:                make(chan bool),
		cancelFnMap:                   make(map[fetcherID]*fetchRun),
		targetedFetchCh:               make(chan []fetcherID),
		logger:                        logger,
		ch:                            make(chan walletevent.Event, 100),
	}

	return service
}

func (s *Service) startNetworkWatcher() {
	if s.networksPublisher == nil {
		return
	}

	chNetworkChange, unsubFnNetworkChange := pubsub.Subscribe[networks.EventActiveNetworksChanged](s.networksPublisher, 10)
	go func() {
		defer panics.LogOnPanic()
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
		defer panics.LogOnPanic()
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
			// Instead of triggering a refetch for all accounts and chains, we trigger a fetch for the specific fetcher IDs
			// This approach saves a lot of rpc calls and speeds up the fetch process
			ids, err := s.buildFetcherIDsFromTransactionStatus(&payload)
			if err != nil {
				s.logger.Error("Failed to build fetcher IDs from transaction status", zap.Error(err))
				return
			}
			if len(ids) > 0 {
				s.targetedFetchCh <- ids
				return
			}
		}
	}
}

func (s *Service) startAccountWatcher() {
	if s.accountsPublisher == nil {
		return
	}

	chAdded, unsubFnAdded := pubsub.Subscribe[accountsevent.AccountsAddedEvent](s.accountsPublisher, 10)
	go func() {
		defer panics.LogOnPanic()
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
		defer panics.LogOnPanic()
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

// startBalanceChangeWatcher subscribes to multistandardbalance fetch-finished events and triggers activity fetch
func (s *Service) startBalanceChangeWatcher() {
	if s.multistandardBalancePublisher == nil {
		return
	}

	ch, unsub := pubsub.Subscribe[multistandardbalance.EventBalanceFetchFinished](s.multistandardBalancePublisher, 10)
	go func() {
		defer panics.LogOnPanic()
		defer unsub()
		for {
			select {
			case <-s.stopCh:
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				if !event.BalanceChanged {
					continue
				}
				if !s.activityFetcherManager.IsChainSupported(event.Key.ChainID) {
					continue
				}
				ids := []fetcherID{{account: event.Key.Account, chainID: event.Key.ChainID}}
				select {
				case s.targetedFetchCh <- ids:
				case <-s.stopCh:
					return
				}
			}
		}
	}()
}

func (s *Service) Start(ctx context.Context) {
	s.logger.Info("Starting activity fetcher")
	s.stopCh = make(chan struct{})
	s.cancelFnMapMutex.Lock()
	s.ctx = ctx
	s.cancelFnMapMutex.Unlock()

	s.startNetworkWatcher()
	s.startAccountWatcher()
	s.startTransactionWatcher()
	s.startBalanceChangeWatcher()

	go func() {
		defer panics.LogOnPanic()

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
			case ids := <-s.targetedFetchCh:
				s.fetchActivityForSpecificFetcherIDs(ctx, ids)
			}
		}
	}()

	// Initial fetch on service start
	s.triggerRefetch(false)
}

func (s *Service) triggerRefetch(bypassIntervalCheck bool) {
	s.checkRefetchCh <- bypassIntervalCheck
}

func (s *Service) buildFetcherIDsFromTransactionStatus(payload *pendingtxtracker.StatusChangedPayload) ([]fetcherID, error) {
	if payload.SendDetails == nil {
		return s.getDesiredFetcherIDs() // if no send details, fetch for all accounts and chains, this should never happen
	}

	alreadyAdded := make(map[fetcherID]struct{}) // avoid duplicates, in case the tx was sent to the same account and chain
	result := make([]fetcherID, 0)

	addFetcherIDIfNotAlreadyAdded := func(addr gethcommon.Address, chainID uint64) {
		if chainID == 0 || addr == common.ZeroAddress() {
			return
		}
		id := fetcherID{account: addr, chainID: chainID}
		if _, ok := alreadyAdded[id]; !ok {
			alreadyAdded[id] = struct{}{}
			result = append(result, id)
		}
	}

	addFetcherIDOrIDsIfParamsAreEmpty := func(addr gethcommon.Address, chainID uint64) error {
		var (
			addresses []gethcommon.Address
			chainIDs  []uint64
			err       error
		)
		if addr == common.ZeroAddress() {
			addresses, err = s.getCurrentAccountsAddresses()
			if err != nil {
				return err
			}
		} else {
			addresses = []gethcommon.Address{addr}
		}

		if chainID == 0 {
			chainIDs, err = s.getCurrentChainIDs()
			if err != nil {
				return err
			}
		} else {
			chainIDs = []uint64{chainID}
		}

		for _, address := range addresses {
			for _, chainID := range chainIDs {
				addFetcherIDIfNotAlreadyAdded(address, chainID)
			}
		}
		return nil
	}

	err := addFetcherIDOrIDsIfParamsAreEmpty(gethcommon.Address(payload.SendDetails.FromAddress), payload.SendDetails.FromChain)
	if err != nil {
		return nil, err
	}
	err = addFetcherIDOrIDsIfParamsAreEmpty(gethcommon.Address(payload.SendDetails.ToAddress), payload.SendDetails.ToChain)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) fetchActivityForSpecificFetcherIDs(ctx context.Context, ids []fetcherID) {
	for _, id := range ids {
		if !s.activityFetcherManager.IsChainSupported(id.chainID) {
			s.logger.Debug("Activity fetcher does not support chain", zap.Uint64("chainID", id.chainID))
			continue
		}
		s.startFetchActivity(ctx, id)
	}
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
		if run, ok := s.cancelFnMap[fetcherID]; ok {
			run.cancel()
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

// Upserts the run for a given account and chainID
// If the task is running, it will be cancelled before creating a new one
func (s *Service) upsertCancelFn(ctx context.Context, fetcherID fetcherID) (context.Context, *fetchRun) {
	s.cancelFnMapMutex.Lock()
	defer s.cancelFnMapMutex.Unlock()

	// If the task is running, cancel it before creating a new one
	if run, ok := s.cancelFnMap[fetcherID]; ok {
		run.cancel()
	}
	return s.newRunLocked(ctx, fetcherID)
}

func (s *Service) newRunLocked(ctx context.Context, fetcherID fetcherID) (context.Context, *fetchRun) {
	tmpCtx, cancelFn := context.WithCancel(ctx)
	run := &fetchRun{cancel: cancelFn, done: make(chan struct{})}
	s.cancelFnMap[fetcherID] = run
	return tmpCtx, run
}

// Cancels the run and removes it for a given account and chainID, unless a newer run replaced it
func (s *Service) runAndRemoveCancelFn(fetcherID fetcherID, run *fetchRun) {
	s.cancelFnMapMutex.Lock()
	defer s.cancelFnMapMutex.Unlock()

	run.cancel()
	if current, ok := s.cancelFnMap[fetcherID]; !ok || current != run {
		return
	}
	delete(s.cancelFnMap, fetcherID)

	// Check if all fetchers have completed
	if len(s.cancelFnMap) == 0 {
		s.emitFetchComplete(fetcherID)
	}
}

func (s *Service) emitFetchComplete(fetcherID fetcherID) {
	// Emit event to notify that activity fetch is complete
	// This happens both on initial fetch and periodic fetches
	if s.eventFeed != nil {
		s.eventFeed.Send(walletevent.Event{
			Type:     EventActivityFetchComplete,
			Message:  "{}",
			ChainID:  fetcherID.chainID,
			Accounts: []gethcommon.Address{fetcherID.account},
		})
	}
}

func (s *Service) removeAllCancelFns() {
	s.cancelFnMapMutex.Lock()
	defer s.cancelFnMapMutex.Unlock()

	for fetcherID, run := range s.cancelFnMap {
		run.cancel()
		delete(s.cancelFnMap, fetcherID)
	}
}

// Starts or restarts fetching activity for a given account and chainID
func (s *Service) startFetchActivity(ctx context.Context, fetcherID fetcherID) {
	tmpCtx, run := s.upsertCancelFn(ctx, fetcherID)
	s.runFetch(tmpCtx, fetcherID, run)
}

func (s *Service) runFetch(ctx context.Context, fetcherID fetcherID, run *fetchRun) {
	go func() {
		defer panics.LogOnPanic()
		defer s.runAndRemoveCancelFn(fetcherID, run)
		defer close(run.done)
		run.err = s.fetchActivity(ctx, fetcherID.chainID, fetcherID.account)
	}()
}

// runningOrNewFetch returns the running fetch of fetcherID, or starts one.
func (s *Service) runningOrNewFetch(fetcherID fetcherID) (*fetchRun, error) {
	s.cancelFnMapMutex.Lock()
	if run, ok := s.cancelFnMap[fetcherID]; ok {
		s.cancelFnMapMutex.Unlock()
		return run, nil
	}
	if s.ctx == nil {
		s.cancelFnMapMutex.Unlock()
		return nil, errors.New("activity fetcher is not started")
	}
	tmpCtx, run := s.newRunLocked(s.ctx, fetcherID)
	s.cancelFnMapMutex.Unlock()

	s.runFetch(tmpCtx, fetcherID, run)
	return run, nil
}

func (s *Service) currentRun(fetcherID fetcherID) *fetchRun {
	s.cancelFnMapMutex.RLock()
	defer s.cancelFnMapMutex.RUnlock()
	return s.cancelFnMap[fetcherID]
}

// IsChainSupported reports whether a history provider is available for the chain.
func (s *Service) IsChainSupported(chainID uint64) bool {
	return s.activityFetcherManager.IsChainSupported(chainID)
}

// FetchAndWait fetches new transfers for each key, or waits for the fetch already running, until ctx ends.
// It reports which keys finished a fetch without an error.
func (s *Service) FetchAndWait(ctx context.Context, keys []FetchKey) map[FetchKey]bool {
	complete := make(map[FetchKey]bool, len(keys))
	var mutex sync.Mutex
	var wg sync.WaitGroup
	for _, key := range keys {
		id := fetcherID{account: key.Address, chainID: key.ChainID}
		run, err := s.runningOrNewFetch(id)
		if err != nil {
			s.logger.Error("Failed to start activity fetch", zap.Error(err))
			complete[key] = false
			continue
		}
		wg.Add(1)
		go func() {
			defer panics.LogOnPanic()
			defer wg.Done()
			ok := s.waitForFetch(ctx, id, run)
			mutex.Lock()
			complete[key] = ok
			mutex.Unlock()
		}()
	}
	wg.Wait()
	return complete
}

// waitForFetch waits for run, and for the runs that replace it, until one succeeds.
func (s *Service) waitForFetch(ctx context.Context, id fetcherID, run *fetchRun) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case <-run.done:
		}
		if run.err == nil {
			return true
		}
		next := s.currentRun(id)
		if next == nil || next == run {
			return false
		}
		run = next
	}
}

func (s *Service) fetchActivity(ctx context.Context, chainID uint64, account gethcommon.Address) error {

	// Get current block
	ethClient, err := s.ethClientGetter.EthClient(chainID)
	if err != nil {
		s.logger.Error("Failed to get rpc client", zap.Error(err))
		return err
	}
	currentBlock, err := ethClient.BlockNumber(ctx)
	if err != nil {
		s.logger.Error("Failed to get current block", zap.Error(err))
		return err
	}

	container, err := s.activityFetcherManager.FetchActivity(ctx, chainID, account, currentBlock)
	if err != nil {
		s.logger.Error("Failed to fetch activity", zap.Error(err))
		return err
	}
	if len(container.Items) > 0 {
		// at least one new transfer for detected, so trigger a balance refresh
		s.emitNewTransfersDetected(chainID, account, len(container.Items))
	}
	return nil
}

func (s *Service) emitNewTransfersDetected(chainID uint64, account gethcommon.Address, count int) {
	if s.eventFeed == nil {
		return
	}
	s.eventFeed.Send(walletevent.Event{
		Type:     EventActivityNewTransfersDetected,
		ChainID:  chainID,
		Accounts: []gethcommon.Address{account},
	})
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
