package ownership

//go:generate go tool mockgen -package=mock_ownership -source=controller.go -destination=mock/controller.go

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	activityRefetchMarginSeconds = 30 * 60 // Trigger a fetch if activity is detected this many seconds before the last fetch
)

type loaderPerChainID = map[walletCommon.ChainID]*PeriodicalLoader
type loaderPerAddressAndChainID = map[common.Address]loaderPerChainID

type TransferCb func(common.Address, walletCommon.ChainID, []transfer.Transfer)

type AccountsProvider interface {
	GetWalletAddresses() ([]types.Address, error)
}

type NetworksProvider interface {
	GetActiveNetworks() ([]*params.Network, error)
	GetPublisher() *pubsub.Publisher
}

type Controller struct {
	fetcher           CollectibleOwnershipFetcher
	storage           CollectibleOwnershipStorage
	walletFeed        *event.Feed
	accountsProvider  AccountsProvider
	accountsPublisher *pubsub.Publisher

	networksProvider NetworksProvider

	walletEventsWatcher *walletevent.Watcher

	collectiblesPublisher *pubsub.Publisher

	periodicalLoaders loaderPerAddressAndChainID
	loadersLock       sync.RWMutex

	stopCh chan struct{}

	loaderParams *PeriodicalLoaderParams

	logger *zap.Logger
}

func NewController(
	storage CollectibleOwnershipStorage,
	walletFeed *event.Feed,
	accountsProvider AccountsProvider,
	accountsPublisher *pubsub.Publisher,
	networksProvider NetworksProvider,
	fetcher CollectibleOwnershipFetcher,
	collectiblesPublisher *pubsub.Publisher,
	logger *zap.Logger,
) *Controller {
	return &Controller{
		fetcher:               fetcher,
		storage:               storage,
		walletFeed:            walletFeed,
		accountsProvider:      accountsProvider,
		accountsPublisher:     accountsPublisher,
		networksProvider:      networksProvider,
		periodicalLoaders:     make(loaderPerAddressAndChainID),
		collectiblesPublisher: collectiblesPublisher,
		logger:                logger.Named("OwnershipController"),
	}
}

func (c *Controller) StartWithLoaderParams(params PeriodicalLoaderParams) {
	c.loaderParams = &params
	c.Start()
}

func (c *Controller) Start() {
	if c.stopCh != nil {
		return
	}

	c.stopCh = make(chan struct{})

	// Setup periodical collectibles refresh
	c.checkPeriodicalLoaders()

	// Setup collectibles fetch when a new account gets added
	c.startAccountsWatcher()

	// Setup collectibles fetch when relevant activity is detected
	c.startWalletEventsWatcher()

	// Setup collectibles fetch when active networks change
	c.startNetworkEventsWatcher()
}

func (c *Controller) Stop() {
	if c.stopCh == nil {
		return
	}

	close(c.stopCh)
	c.stopCh = nil

	c.stopWalletEventsWatcher()

	c.stopPeriodicalLoaders()
}

func (c *Controller) RefetchOwnedCollectibles() {
	c.stopPeriodicalLoaders()
	c.checkPeriodicalLoaders()
}

func (c *Controller) GetLoaderState(chainID walletCommon.ChainID, address common.Address) LoaderState {
	loader := c.getLoader(chainID, address)
	if loader == nil {
		return LoaderStateNotAvailable
	}

	return loader.GetState()
}

func (c *Controller) stopPeriodicalLoaders() {
	c.loadersLock.Lock()
	defer c.loadersLock.Unlock()

	for _, addressLoaders := range c.periodicalLoaders {
		for _, loader := range addressLoaders {
			loader.Stop()
		}
	}

	c.periodicalLoaders = make(loaderPerAddressAndChainID)
}

// Starts (or restarts) periodical fetching for the given account address for all chains
func (c *Controller) startPeriodicalLoaderForAccountAndChainID(address common.Address, chainID walletCommon.ChainID, delayed bool) {
	c.stopPeriodicalLoaderForAccountAndChainID(address, chainID)

	c.logger.Debug("Start periodical fetching",
		zap.Stringer("address", address),
		zap.Stringer("chainID", chainID),
		zap.Bool("delayed", delayed),
	)

	if _, ok := c.periodicalLoaders[address]; !ok {
		c.periodicalLoaders[address] = make(loaderPerChainID)
	}

	loaderParams := DefaultPeriodicalLoaderParams()
	if c.loaderParams != nil {
		loaderParams = *c.loaderParams
	}

	loader := NewPeriodicalLoader(
		chainID,
		address,
		c.fetcher,
		c.storage,
		c.collectiblesPublisher,
		loaderParams,
		c.logger,
	)

	c.periodicalLoaders[address][chainID] = loader
	const waitForInterval = false
	loader.Start(delayed, waitForInterval)
}

func (c *Controller) stopPeriodicalLoaderForAccountAndChainID(address common.Address, chainID walletCommon.ChainID) {
	if _, ok := c.periodicalLoaders[address]; !ok {
		return
	}
	if _, ok := c.periodicalLoaders[address][chainID]; !ok {
		return
	}

	c.logger.Debug("Stop periodical fetching",
		zap.Stringer("address", address),
		zap.Stringer("chainID", chainID),
	)

	c.periodicalLoaders[address][chainID].Stop()
	delete(c.periodicalLoaders[address], chainID)
	// If it was the last chain, delete the address as well
	if len(c.periodicalLoaders[address]) == 0 {
		delete(c.periodicalLoaders, address)
	}
}

func (c *Controller) startAccountsWatcher() {
	if c.accountsPublisher == nil {
		return
	}

	chAdded, unsubFnAdded := pubsub.Subscribe[accountsevent.AccountsAddedEvent](c.accountsPublisher, 10)
	chRemoved, unsubFnRemoved := pubsub.Subscribe[accountsevent.AccountsRemovedEvent](c.accountsPublisher, 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsubFnAdded()
		defer unsubFnRemoved()
		for {
			select {
			case <-c.stopCh:
				return
			case _, ok := <-chAdded:
				if !ok {
					return
				}
				c.checkPeriodicalLoaders()
			case _, ok := <-chRemoved:
				if !ok {
					return
				}
				c.checkPeriodicalLoaders()
			}
		}
	}()
}

func (c *Controller) startWalletEventsWatcher() {
	if c.walletEventsWatcher != nil {
		return
	}

	if c.walletFeed == nil {
		return
	}

	walletEventCb := func(event walletevent.Event) {
		if event.Type != transfer.EventInternalERC721TransferDetected &&
			event.Type != transfer.EventInternalERC1155TransferDetected {
			return
		}

		chainID := walletCommon.ChainID(event.ChainID)
		for _, account := range event.Accounts {
			c.refetchOwnershipIfRecentTransfer(account, chainID, event.At)
		}
	}

	c.walletEventsWatcher = walletevent.NewWatcher(c.walletFeed, walletEventCb)

	c.walletEventsWatcher.Start()
}

func (c *Controller) stopWalletEventsWatcher() {
	if c.walletEventsWatcher != nil {
		c.walletEventsWatcher.Stop()
		c.walletEventsWatcher = nil
	}
}

func (c *Controller) startNetworkEventsWatcher() {
	if c.networksProvider == nil {
		return
	}

	ch, unsub := pubsub.Subscribe[network.EventActiveNetworksChanged](c.networksProvider.GetPublisher(), 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsub()
		for {
			select {
			case <-c.stopCh:
				return
			case _, ok := <-ch:
				if !ok {
					return
				}
				c.checkPeriodicalLoaders()
			}
		}
	}()
}

func (c *Controller) refetchOwnershipIfRecentTransfer(account common.Address, chainID walletCommon.ChainID, latestTxTimestamp int64) {
	// Check last ownership update timestamp
	timestamp, err := c.storage.GetOwnershipUpdateTimestamp(account, chainID)

	if err != nil {
		c.logger.Error("Error getting ownership update timestamp",
			zap.Error(err))
		return
	}
	if timestamp == InvalidTimestamp {
		// Ownership was never fetched for this account
		return
	}

	timeCheck := max(timestamp-activityRefetchMarginSeconds, 0)

	if latestTxTimestamp < timeCheck {
		c.logger.Debug("Detected transfer is too old, skipping refetch",
			zap.Stringer("chainID", chainID),
			zap.String("address", gocommon.TruncateWithDot(account.String())),
			zap.Int64("tx timestamp", latestTxTimestamp),
			zap.Int64("last fetch timestamp", timestamp),
		)
		return
	}

	// Restart fetching for account + chainID
	c.loadersLock.Lock()
	defer c.loadersLock.Unlock()
	c.startPeriodicalLoaderForAccountAndChainID(account, chainID, true)
}

// Triggers a load for the given account and chainID in a blocking way
func (c *Controller) TriggerLoad(ctx context.Context, chainID walletCommon.ChainID, account common.Address) error {
	as := account.String()
	c.logger.Debug("Trigger load on-demand",
		zap.String("address", gocommon.TruncateWithDot(as)),
		zap.Stringer("chainID", chainID),
	)

	found, err := c.loadWithPeriodicalLoaderIfFound(ctx, chainID, account)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	// Periodical loader not found for this account + chainID, start a singleshot Loader
	parameters := DefaultLoaderParams()
	parameters.LoadDelay = 0
	parameters.FetchLimit = thirdparty.FetchNoLimit

	loader := NewLoader(
		chainID,
		account,
		c.fetcher,
		c.storage,
		c.collectiblesPublisher,
		parameters,
		c.logger,
	)

	_, err = loader.Load(ctx)
	return err
}

func (c *Controller) getLoader(chainID walletCommon.ChainID, account common.Address) *PeriodicalLoader {
	c.loadersLock.RLock()
	defer c.loadersLock.RUnlock()

	if _, ok := c.periodicalLoaders[account]; !ok {
		return nil
	}

	if _, ok := c.periodicalLoaders[account][chainID]; !ok {
		return nil
	}

	return c.periodicalLoaders[account][chainID]
}

func (c *Controller) loadWithPeriodicalLoaderIfFound(ctx context.Context, chainID walletCommon.ChainID, account common.Address) (found bool, err error) {
	loader := c.getLoader(chainID, account)
	if loader == nil {
		return
	}

	found = true

	finishedCh, finishedUnsubFn := pubsub.Subscribe[EventOwnedCollectiblesLoadFinished](c.collectiblesPublisher, 10)
	defer finishedUnsubFn()

	errCh, errUnsubFn := pubsub.Subscribe[EventOwnedCollectiblesLoadError](c.collectiblesPublisher, 10)
	defer errUnsubFn()

	go func() {
		defer gocommon.LogOnPanic()
		// Trigger load in case it's not already in progress
		_ = loader.Load(ctx)
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case finishedEvent, ok := <-finishedCh:
		if !ok {
			return
		}
		if finishedEvent.ChainID == chainID && finishedEvent.Account == account {
			return
		}
	case errEvent, ok := <-errCh:
		if !ok {
			return
		}
		if errEvent.ChainID == chainID && errEvent.Account == account {
			err = errEvent.Error
			return
		}
	}

	return
}

func (c *Controller) checkPeriodicalLoaders() {
	c.loadersLock.Lock()
	defer c.loadersLock.Unlock()

	type loaderKey struct {
		address common.Address
		chainID walletCommon.ChainID
	}

	// Get current list of addresses and active networks
	newAddresses, err := c.accountsProvider.GetWalletAddresses()
	if err != nil {
		c.logger.Error("Error getting wallet addresses", zap.Error(err))
		return
	}
	newNetworks, err := c.networksProvider.GetActiveNetworks()
	if err != nil {
		c.logger.Error("Error getting active networks", zap.Error(err))
		return
	}

	newLoaderKeys := make(map[loaderKey]struct{})
	for _, address := range newAddresses {
		for _, network := range newNetworks {
			newLoaderKeys[loaderKey{address: common.Address(address), chainID: walletCommon.ChainID(network.ChainID)}] = struct{}{}
		}
	}

	currentLoaderKeys := make(map[loaderKey]struct{})
	for address, chainIDs := range c.periodicalLoaders {
		for chainID := range chainIDs {
			currentLoaderKeys[loaderKey{address: address, chainID: chainID}] = struct{}{}
		}
	}

	loaderKeysToAdd := make(map[loaderKey]struct{})
	loaderKeysToRemove := make(map[loaderKey]struct{})

	for loaderKey := range newLoaderKeys {
		if _, ok := currentLoaderKeys[loaderKey]; !ok {
			// Key in new list, but not in current list, add it
			loaderKeysToAdd[loaderKey] = struct{}{}
		}
	}

	for loaderKey := range currentLoaderKeys {
		if _, ok := newLoaderKeys[loaderKey]; !ok {
			// Key in current list, but not in new list, remove it
			loaderKeysToRemove[loaderKey] = struct{}{}
		}
	}

	for loaderKey := range loaderKeysToAdd {
		c.startPeriodicalLoaderForAccountAndChainID(loaderKey.address, loaderKey.chainID, false)
	}

	for loaderKey := range loaderKeysToRemove {
		c.stopPeriodicalLoaderForAccountAndChainID(loaderKey.address, loaderKey.chainID)
	}
}
