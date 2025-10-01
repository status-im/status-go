package multistandardbalance

//go:generate go tool mockgen -package=mock_multistandardbalance -source=controller.go -destination=mock/controller.go

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/accounts/accountsevent"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"github.com/bep/debounce"

	"go.uber.org/zap"
)

type AccountsProvider interface {
	GetWalletAddresses() ([]types.Address, error)
}

type NetworksProvider interface {
	GetActiveNetworks() ([]*params.Network, error)
	GetPublisher() *pubsub.Publisher
}

type TokenListProvider interface {
	GetTokenContractAddresses(chainID uint64) ([]common.Address, error)
}

type CollectiblesListProvider interface {
	GetCollectiblesList(chainID uint64, account common.Address) (erc721 []CollectibleID, erc1155 []CollectibleID, err error)
}

type BalanceFetcher interface {
	FetchBalances(ctx context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error)
}

type LastBlockManager interface {
	SetLatestBlockNumber(chainID uint64, blockNumber uint64)
}

type ControllerConfig struct {
	FetchDebounceTime time.Duration
	FetchPeriod       time.Duration
}

func DefaultControllerConfig() ControllerConfig {
	return ControllerConfig{
		FetchDebounceTime: 10 * time.Second,
		FetchPeriod:       2 * time.Minute,
	}
}

type Controller struct {
	config  ControllerConfig
	storage Storage
	fetcher BalanceFetcher

	mu sync.RWMutex

	accountsProvider        AccountsProvider
	accountsPublisher       *pubsub.Publisher
	networksProvider        NetworksProvider
	tokenListProvider       TokenListProvider
	collectibleListProvider CollectiblesListProvider
	lastBlockManager        LastBlockManager

	publisher *pubsub.Publisher

	stopCh          chan struct{}
	triggerFetchCh  chan bool
	fetchDebounceFn func(f func())

	logger *zap.Logger
}

func NewController(
	config ControllerConfig,
	storage Storage,
	fetcher BalanceFetcher,
	accountsProvider AccountsProvider,
	accountsPublisher *pubsub.Publisher,
	networksProvider NetworksProvider,
	tokenListProvider TokenListProvider,
	collectibleListProvider CollectiblesListProvider,
	lastBlockManager LastBlockManager,
	logger *zap.Logger,
) *Controller {
	return &Controller{
		config:                  config,
		storage:                 storage,
		fetcher:                 fetcher,
		accountsProvider:        accountsProvider,
		accountsPublisher:       accountsPublisher,
		networksProvider:        networksProvider,
		tokenListProvider:       tokenListProvider,
		collectibleListProvider: collectibleListProvider,
		lastBlockManager:        lastBlockManager,
		publisher:               pubsub.NewPublisher(),
		fetchDebounceFn:         debounce.New(config.FetchDebounceTime),
		triggerFetchCh:          make(chan bool),
		logger:                  logger,
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
		defer gocommon.LogOnPanic()
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
				c.triggerFetch(false)
				accounts, err := c.getAllAccounts()
				if err != nil {
					continue
				}
				c.storage.ClearMissingAccounts(context.Background(), accounts)
			case _, ok := <-addedCh:
				if !ok {
					return
				}
				c.triggerFetch(false)
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
		defer gocommon.LogOnPanic()
		defer unsubFn()
		for {
			select {
			case <-c.stopCh:
				return
			case _, ok := <-ch:
				if !ok {
					return
				}
				c.triggerFetch(false)
				networks, err := c.getAllNetworks()
				if err != nil {
					continue
				}
				c.storage.ClearMissingChains(context.Background(), networks)
			}
		}
	}()
}

func (c *Controller) startFetcher() {
	ticker := time.NewTicker(c.config.FetchPeriod)
	go func() {
		defer gocommon.LogOnPanic()
		defer ticker.Stop()
		for {
			select {
			case <-c.stopCh:
				return
			case <-ticker.C:
				// Force fetch regardless of last fetch time
				c.triggerFetch(true)
			case forced := <-c.triggerFetchCh:
				ticker.Reset(c.config.FetchPeriod)
				c.triggerFetch(forced)
			}
		}
	}()
	// Trigger initial fetch
	c.triggerFetch(true)
}

func (c *Controller) TriggerFetch(forced bool) {
	select {
	case c.triggerFetchCh <- forced:
	default:
	}
}

func (c *Controller) triggerFetch(forced bool) {
	c.fetchDebounceFn(func() {
		balancesToFetch := c.computeBalancesToFetch(forced)
		c.triggerFetchNow(balancesToFetch)
	})
}

// Needs to fetch if the balance has never been fetched or if the last fetch was more than fetchPeriod ago
func (c *Controller) needsToFetch(state State) bool {
	return state.FetchedAt == NeverFetched || state.FetchedAt+int64(c.config.FetchPeriod.Seconds()) < time.Now().Unix()
}

func (c *Controller) getAllAccounts() ([]common.Address, error) {
	accounts, err := c.accountsProvider.GetWalletAddresses()
	if err != nil {
		return nil, nil
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
		return nil, nil
	}
	chains := make([]uint64, len(networks))
	for i, network := range networks {
		chains[i] = network.ChainID
	}
	return chains, nil
}

// For all accounts and networks, check which balance types needs to be fetched
func (c *Controller) computeBalancesToFetch(forced bool) map[BalancesKey][]multistandardfetcher.ResultType {
	balancesToFetch := make(map[BalancesKey][]multistandardfetcher.ResultType)

	accounts, err := c.getAllAccounts()
	if err != nil {
		return nil
	}

	networks, err := c.getAllNetworks()
	if err != nil {
		return nil
	}
	c.storage.ClearMissingChains(context.Background(), networks)

	for _, account := range accounts {
		for _, network := range networks {
			key := BalancesKey{Account: account, ChainID: network}

			resultTypes := make([]multistandardfetcher.ResultType, 0, 4)
			if _, state, err := c.storage.GetNativeBalance(context.Background(), key); err == nil && c.needsToFetch(state) || forced {
				resultTypes = append(resultTypes, multistandardfetcher.ResultTypeNative)
			}
			if _, state, err := c.storage.GetERC20Balances(context.Background(), key); err == nil && c.needsToFetch(state) || forced {
				resultTypes = append(resultTypes, multistandardfetcher.ResultTypeERC20)
			}
			if _, state, err := c.storage.GetERC721Balances(context.Background(), key); err == nil && c.needsToFetch(state) || forced {
				resultTypes = append(resultTypes, multistandardfetcher.ResultTypeERC721)
			}
			if _, state, err := c.storage.GetERC1155Balances(context.Background(), key); err == nil && c.needsToFetch(state) || forced {
				resultTypes = append(resultTypes, multistandardfetcher.ResultTypeERC1155)
			}

			balancesToFetch[key] = resultTypes
		}
	}

	return balancesToFetch
}

func (c *Controller) triggerFetchNow(balancesToFetch map[BalancesKey][]multistandardfetcher.ResultType) {
	c.logger.Debug("starting fetch")
	fetchConfigs := c.buildFetchConfigs(balancesToFetch)

	if len(fetchConfigs) == 0 {
		return
	}

	for chainID, fetchConfig := range fetchConfigs {
		ctx, cancel := context.WithCancel(context.Background())

		resultsCh, err := c.fetcher.FetchBalances(ctx, chainID, fetchConfig)
		if err != nil {
			c.logger.Error("failed to fetch balances", zap.Uint64("chainID", chainID), zap.Error(err))
			c.sendEventBalanceFetchFailedToStart(chainID, fetchConfig, err)
			cancel()
			continue
		}

		c.logger.Debug("fetch started", zap.Uint64("chainID", chainID))
		c.sendEventBalanceFetchStarted(chainID, fetchConfig)

		go func() {
			defer gocommon.LogOnPanic()
			defer cancel()

			for {
				select {
				case <-c.stopCh:
					return
				case result, ok := <-resultsCh:
					if !ok {
						return
					}
					c.handleFetchResult(ctx, chainID, result)
				}
			}
		}()
	}
}
