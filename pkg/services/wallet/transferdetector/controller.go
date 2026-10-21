package transferdetector

//go:generate go tool mockgen -package=mock_transferdetector -source=controller.go -destination=mock/controller.go

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/bep/debounce"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/eventfilter"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/traffic"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"
	"github.com/status-im/status-go/pkg/services/networks"

	"go.uber.org/zap"
)

type AccountsProvider interface {
	GetWalletAddresses() ([]types.Address, error)
}

type NetworksProvider interface {
	GetActiveNetworks() ([]*params.Network, error)
	GetPublisher() *pubsub.Publisher
}

type LastBlockProvider interface {
	// GetReadBlocks are the first and the last block balances of chainID were
	// read at.
	GetReadBlocks(chainID uint64) (first uint64, last uint64, ok bool)
}

type FilterProvider interface {
	FilterTransfers(ctx context.Context, chainID uint64, config eventfilter.TransferQueryConfig) ([]eventlog.Event, error)
}

type ControllerConfig struct {
	FetchDebounceTime time.Duration
}

func DefaultControllerConfig() ControllerConfig {
	return ControllerConfig{
		FetchDebounceTime: 10 * time.Second,
	}
}

type Controller struct {
	config  ControllerConfig
	fetcher FilterProvider

	mu                     sync.RWMutex
	lastFetchedBlockNumber map[uint64]uint64 // map[chainID]blockNumber

	accountsProvider  AccountsProvider
	accountsPublisher *pubsub.Publisher
	networksProvider  NetworksProvider
	lastBlockProvider LastBlockProvider

	publisher *pubsub.Publisher

	stopCh          chan struct{}
	triggerFetchCh  chan struct{}
	fetchDebounceFn func(f func())

	logger *zap.Logger
}

func NewController(
	config ControllerConfig,
	fetcher FilterProvider,
	accountsProvider AccountsProvider,
	accountsPublisher *pubsub.Publisher,
	networksProvider NetworksProvider,
	lastBlockProvider LastBlockProvider,
	logger *zap.Logger,
) *Controller {
	return &Controller{
		config:                 config,
		fetcher:                fetcher,
		accountsProvider:       accountsProvider,
		accountsPublisher:      accountsPublisher,
		networksProvider:       networksProvider,
		lastBlockProvider:      lastBlockProvider,
		publisher:              pubsub.NewPublisher(),
		fetchDebounceFn:        debounce.New(config.FetchDebounceTime),
		triggerFetchCh:         make(chan struct{}, 1),
		logger:                 logger,
		lastFetchedBlockNumber: make(map[uint64]uint64),
	}
}

func (c *Controller) Start() {
	if c.stopCh != nil {
		return
	}
	c.stopCh = make(chan struct{})

	c.startAccountsWatcher(c.stopCh)
	c.startNetworksWatcher(c.stopCh)
	c.startFetcher(c.stopCh)
}

func (c *Controller) Stop() {
	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
	}
}

func (c *Controller) GetPublisher() *pubsub.Publisher {
	return c.publisher
}

func (c *Controller) startAccountsWatcher(stopCh <-chan struct{}) {
	if c.accountsPublisher == nil {
		return
	}

	removedCh, removedUnsubFn := pubsub.Subscribe[accountsevent.AccountsRemovedEvent](c.accountsPublisher, 10)
	addedCh, addedUnsubFn := pubsub.Subscribe[accountsevent.AccountsAddedEvent](c.accountsPublisher, 10)
	go func() {
		defer panics.LogOnPanic()
		defer removedUnsubFn()
		defer addedUnsubFn()
		for {
			select {
			case <-stopCh:
				return
			case _, ok := <-removedCh:
				if !ok {
					return
				}
				c.run(stopCh)
			case _, ok := <-addedCh:
				if !ok {
					return
				}
				c.run(stopCh)
			}
		}
	}()
}

func (c *Controller) startNetworksWatcher(stopCh <-chan struct{}) {
	if c.networksProvider == nil {
		return
	}

	ch, unsubFn := pubsub.Subscribe[networks.EventActiveNetworksChanged](c.networksProvider.GetPublisher(), 10)

	go func() {
		defer panics.LogOnPanic()
		defer unsubFn()
		for {
			select {
			case <-stopCh:
				return
			case _, ok := <-ch:
				if !ok {
					return
				}
				c.run(stopCh)
			}
		}
	}()
}

// startFetcher runs detection on TriggerFetch; the balance refresh triggers it
// with every periodic read, so it runs as often as balances are read.
func (c *Controller) startFetcher(stopCh <-chan struct{}) {
	go func() {
		defer panics.LogOnPanic()
		for {
			select {
			case <-stopCh:
				return
			case <-c.triggerFetchCh:
				c.run(stopCh)
			}
		}
	}()
	// Trigger initial fetch
	c.run(stopCh)
}

func (c *Controller) TriggerFetch() {
	select {
	case c.triggerFetchCh <- struct{}{}:
	default:
	}
}

func (c *Controller) run(stopCh <-chan struct{}) {
	c.fetchDebounceFn(func() {
		go func() {
			defer panics.LogOnPanic()
			c.runNow(stopCh)
		}()
	})
}

func (c *Controller) getAllAccounts() ([]common.Address, error) {
	accounts, err := c.accountsProvider.GetWalletAddresses()
	if err != nil {
		return nil, err
	}
	gethAccounts := make([]common.Address, len(accounts))
	for i, account := range accounts {
		gethAccounts[i] = common.BytesToAddress(account.Bytes())
	}
	return gethAccounts, nil
}

func (c *Controller) getAllNetworks() ([]uint64, error) {
	networks, err := c.networksProvider.GetActiveNetworks()
	if err != nil {
		return nil, err
	}
	chains := make([]uint64, len(networks))
	for i, network := range networks {
		chains[i] = network.ChainID
	}
	return chains, nil
}

func (c *Controller) runNow(stopCh <-chan struct{}) {
	accounts, err := c.getAllAccounts()
	if err != nil {
		c.logger.Error("failed to get all accounts", zap.Error(err))
		return
	}
	networks, err := c.getAllNetworks()
	if err != nil {
		c.logger.Error("failed to get all networks", zap.Error(err))
		return
	}

	ctx, cancel := context.WithCancel(traffic.WithSource(context.Background(), traffic.TransferDetection))
	defer cancel()

	wg := sync.WaitGroup{}
	for _, network := range networks {
		wg.Add(1)
		go func(chainID uint64) {
			defer panics.LogOnPanic()
			defer wg.Done()

			c.fetchTransfers(ctx, chainID, accounts)
		}(network)
	}

	go func() {
		defer panics.LogOnPanic()
		select {
		case <-stopCh:
			cancel()
			return
		case <-ctx.Done():
			return
		}
	}()
	wg.Wait()
}

// fetchTransfers looks for transfers in the blocks after the last one it
// looked at, up to the last block balances were read at: an estimate of the
// latest block can run ahead of the node and skip blocks for good. With no
// block looked at yet, it starts at the first block balances were read at,
// which a full refresh covered. A range it fails to look at is a detection
// gap, closed by a full refresh of the chain, so it is never looked at again.
func (c *Controller) fetchTransfers(ctx context.Context, chainID uint64, accounts []common.Address) {
	firstReadBlock, lastReadBlock, ok := c.lastBlockProvider.GetReadBlocks(chainID)
	if !ok {
		c.logger.Debug("no balances read yet", zap.Uint64("chainID", chainID))
		return
	}

	lastFetchedBlockNumber := c.getLastFetchedBlockNumber(chainID)
	if lastFetchedBlockNumber == 0 {
		c.logger.Debug("setting initial block for the chain", zap.Uint64("chainID", chainID), zap.Uint64("firstReadBlock", firstReadBlock))
		lastFetchedBlockNumber = firstReadBlock
		c.updateLastFetchedBlockNumber(chainID, lastFetchedBlockNumber)
	}

	if lastFetchedBlockNumber >= lastReadBlock {
		c.logger.Debug("latest block number is already fetched", zap.Uint64("chainID", chainID), zap.Uint64("lastReadBlock", lastReadBlock), zap.Uint64("lastFetchedBlockNumber", lastFetchedBlockNumber))
		return
	}

	fromBlock := lastFetchedBlockNumber + 1
	toBlock := lastReadBlock

	c.sendEventTransferDetectionStarted(chainID, accounts, fromBlock, toBlock)
	events, err := c.fetcher.FilterTransfers(ctx, chainID, eventfilter.TransferQueryConfig{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Accounts:  accounts,
		TransferTypes: []eventfilter.TransferType{
			eventfilter.TransferTypeERC20,
			eventfilter.TransferTypeERC721,
			eventfilter.TransferTypeERC1155,
		},
		Direction: eventfilter.Both,
	})
	if ctx.Err() != nil {
		return
	}
	c.updateLastFetchedBlockNumber(chainID, toBlock)
	if err != nil {
		c.logger.Error("failed to filter transfers", zap.Uint64("chainID", chainID), zap.Error(err))
		c.sendEventTransferDetectionError(chainID, accounts, fromBlock, toBlock, err)
		return
	}

	c.logger.Debug("filtered transfers", zap.Uint64("chainID", chainID), zap.Int64("fromBlock", int64(fromBlock)), zap.Int64("toBlock", int64(toBlock)), zap.Int("events", len(events)))
	c.sendEventTransferDetectionFinished(chainID, accounts, fromBlock, toBlock, events)
}

func (c *Controller) getLastFetchedBlockNumber(chainID uint64) uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastFetchedBlockNumber[chainID]
}

func (c *Controller) updateLastFetchedBlockNumber(chainID uint64, blockNumber uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFetchedBlockNumber[chainID] = blockNumber
}
