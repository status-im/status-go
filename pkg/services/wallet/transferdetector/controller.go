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
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"

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
	GetEstimatedLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error)
}

type FilterProvider interface {
	FilterTransfers(ctx context.Context, chainID uint64, config eventfilter.TransferQueryConfig) ([]eventlog.Event, error)
}

type ControllerConfig struct {
	FetchDebounceTime time.Duration
	FetchPeriod       time.Duration
}

func DefaultControllerConfig() ControllerConfig {
	return ControllerConfig{
		FetchDebounceTime: 10 * time.Second,
		FetchPeriod:       10 * time.Minute,
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
		triggerFetchCh:         make(chan struct{}),
		logger:                 logger,
		lastFetchedBlockNumber: make(map[uint64]uint64),
	}
}

func (c *Controller) Start() {
	if c.stopCh != nil {
		return
	}
	c.stopCh = make(chan struct{})

	c.startAccountsWatcher()
	c.startNetworksWatcher()
	c.startFetcher()
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

func (c *Controller) startAccountsWatcher() {
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
			case <-c.stopCh:
				return
			case _, ok := <-removedCh:
				if !ok {
					return
				}
				c.run()
			case _, ok := <-addedCh:
				if !ok {
					return
				}
				c.run()
			}
		}
	}()
}

func (c *Controller) startNetworksWatcher() {
	if c.networksProvider == nil {
		return
	}

	ch, unsubFn := pubsub.Subscribe[network.EventActiveNetworksChanged](c.networksProvider.GetPublisher(), 10)

	go func() {
		defer panics.LogOnPanic()
		defer unsubFn()
		for {
			select {
			case <-c.stopCh:
				return
			case _, ok := <-ch:
				if !ok {
					return
				}
				c.run()
			}
		}
	}()
}

func (c *Controller) startFetcher() {
	ticker := time.NewTicker(c.config.FetchPeriod)
	go func() {
		defer panics.LogOnPanic()
		defer ticker.Stop()
		for {
			select {
			case <-c.stopCh:
				return
			case <-ticker.C:
				c.run()
			case <-c.triggerFetchCh:
				ticker.Reset(c.config.FetchPeriod)
				c.run()
			}
		}
	}()
	// Trigger initial fetch
	c.run()
}

func (c *Controller) TriggerFetch() {
	select {
	case c.triggerFetchCh <- struct{}{}:
	default:
	}
}

func (c *Controller) run() {
	c.fetchDebounceFn(func() {
		go func() {
			defer panics.LogOnPanic()
			c.runNow()
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

func (c *Controller) runNow() {
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

	ctx, cancel := context.WithCancel(context.Background())
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
		case <-c.stopCh:
			cancel()
			return
		case <-ctx.Done():
			return
		}
	}()
	wg.Wait()
}

func (c *Controller) fetchTransfers(ctx context.Context, chainID uint64, accounts []common.Address) {
	latestBlockNumber, err := c.lastBlockProvider.GetEstimatedLatestBlockNumber(ctx, chainID)
	if err != nil {
		c.logger.Error("failed to get estimated latest block number", zap.Uint64("chainID", chainID), zap.Error(err))
		return
	}

	lastFetchedBlockNumber := c.getLastFetchedBlockNumber(chainID)
	if lastFetchedBlockNumber == 0 {
		c.logger.Debug("setting initial block for the chain", zap.Uint64("chainID", chainID), zap.Uint64("latestBlockNumber", latestBlockNumber))
		c.updateLastFetchedBlockNumber(chainID, latestBlockNumber)
		return
	}

	if lastFetchedBlockNumber >= latestBlockNumber {
		c.logger.Debug("latest block number is already fetched", zap.Uint64("chainID", chainID), zap.Uint64("latestBlockNumber", latestBlockNumber), zap.Uint64("lastFetchedBlockNumber", lastFetchedBlockNumber))
		return
	}

	fromBlock := lastFetchedBlockNumber + 1
	toBlock := latestBlockNumber

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
	if err != nil {
		c.logger.Error("failed to filter transfers", zap.Uint64("chainID", chainID), zap.Error(err))
		c.sendEventTransferDetectionError(chainID, accounts, fromBlock, toBlock, err)
		return
	}

	c.logger.Debug("filtered transfers", zap.Uint64("chainID", chainID), zap.Int64("fromBlock", int64(fromBlock)), zap.Int64("toBlock", int64(toBlock)), zap.Int("events", len(events)))
	c.sendEventTransferDetectionFinished(chainID, accounts, fromBlock, toBlock, events)
	c.updateLastFetchedBlockNumber(chainID, toBlock)
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
