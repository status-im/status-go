package activityfetcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/rpc/network/networksevent"
	"github.com/status-im/status-go/services/accounts/accountsevent"
)

const (
	activityFetchInterval = 10 * time.Minute
)

type fetcherID struct {
	account gethcommon.Address
	chainID uint64
}

type Service struct {
	activityFetcherManager ManagerIface
	networksGetter         network.GetterInterface
	accountsGetter         accounts.AccountsStorage
	rpcClient              rpc.ClientInterface

	networkEventWatcher *networksevent.Watcher
	accountEventWatcher *accountsevent.Watcher
	checkRefetchCh      chan struct{}

	cancelFnMap      map[fetcherID]context.CancelFunc
	cancelFnMapMutex sync.RWMutex

	logger *zap.Logger
}

func NewService(
	activityFetcherManager ManagerIface,
	networksGetter network.GetterInterface,
	networksFeed *event.Feed,
	accountsGetter accounts.AccountsStorage,
	accountsFeed *event.Feed,
	rpcClient rpc.ClientInterface,
) *Service {
	logger := logutils.ZapLogger().Named("ActivityFetcher")

	service := &Service{
		activityFetcherManager: activityFetcherManager,
		networksGetter:         networksGetter,
		accountsGetter:         accountsGetter,
		rpcClient:              rpcClient,
		checkRefetchCh:         make(chan struct{}),
		cancelFnMap:            make(map[fetcherID]context.CancelFunc),
		logger:                 logger,
	}

	networkEventCallbacks := networksevent.EventCallbacks{
		ActiveNetworksChangeCb: func() {
			service.triggerRefetch()
		},
	}

	accountsChangeCb := func(changedAddresses []gethcommon.Address, eventType accountsevent.EventType, currentAddresses []gethcommon.Address) {
		service.triggerRefetch()
	}

	service.networkEventWatcher = networksevent.NewWatcher(networksFeed, networkEventCallbacks)
	service.accountEventWatcher = accountsevent.NewWatcher(accountsGetter, accountsFeed, accountsChangeCb)

	return service
}

func (s *Service) Start(ctx context.Context) {
	s.logger.Info("Starting activity fetcher")

	s.networkEventWatcher.Start()
	s.accountEventWatcher.Start()

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
			case <-s.checkRefetchCh:
				s.fetchActivityForAllAccountsAndChains(ctx, true)
			}
		}
	}()

	// Initial fetch on service start
	s.triggerRefetch()
}

func (s *Service) triggerRefetch() {
	s.checkRefetchCh <- struct{}{}
}

func (s *Service) stop() {
	s.logger.Info("Stopping activity fetcher")

	s.removeAllCancelFns()
	s.networkEventWatcher.Stop()
	s.accountEventWatcher.Stop()
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

	fmt.Printf("cancelFetcherIDs fetcherIDs: %+v\n", fetcherIDs)
	fmt.Printf("cancelFetcherIDs old cancelFnMap: %+v\n", s.cancelFnMap)

	for _, fetcherID := range fetcherIDs {
		if cancelFn, ok := s.cancelFnMap[fetcherID]; ok {
			cancelFn()
			delete(s.cancelFnMap, fetcherID)
		}
	}
	fmt.Printf("cancelFetcherIDs new cancelFnMap: %+v\n", s.cancelFnMap)
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
	fmt.Println("fetchActivityForAllAccountsAndChains")

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

	fmt.Println("runAndRemoveCancelFn", fetcherID)

	fmt.Printf("cancelFnMap before: %+v\n", s.cancelFnMap)

	var cancelFn context.CancelFunc
	var ok bool
	if cancelFn, ok = s.cancelFnMap[fetcherID]; !ok {
		return
	}

	cancelFn()
	delete(s.cancelFnMap, fetcherID)

	fmt.Printf("cancelFnMap after: %+v\n", s.cancelFnMap)
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
	rpcClient, err := s.rpcClient.EthClient(chainID)
	if err != nil {
		s.logger.Error("Failed to get rpc client", zap.Error(err))
		return
	}
	currentBlock, err := rpcClient.BlockNumber(ctx)
	if err != nil {
		s.logger.Error("Failed to get current block", zap.Error(err))
		return
	}

	_, err = s.activityFetcherManager.FetchActivity(ctx, chainID, account, currentBlock)
	if err != nil {
		s.logger.Error("Failed to fetch activity", zap.Error(err))
		return
	}
}
