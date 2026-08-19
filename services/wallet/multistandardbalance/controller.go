package multistandardbalance

//go:generate go tool mockgen -package=mock_multistandardbalance -source=controller.go -destination=mock/controller.go

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/walletevent"

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

type FetchConfig map[BalancesKey][]multistandardfetcher.ResultType

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
	walletFeed              *event.Feed
	walletEventsWatcher     *walletevent.Watcher

	publisher *pubsub.Publisher

	stopCh                     chan struct{}
	fetchDebounceFn            func(f func())
	firstFetchPending          atomic.Bool
	tokenListsColdFetchPending atomic.Bool
	pendingFullFetch           bool
	pendingFetchConfig         FetchConfig

	chainFetchMu      sync.Mutex
	chainFetchCancels map[uint64]context.CancelFunc

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
	walletFeed *event.Feed,
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
		pendingFetchConfig:      make(FetchConfig),
		chainFetchCancels:       make(map[uint64]context.CancelFunc),
		walletFeed:              walletFeed,
		logger:                  logger,
	}
}

func (c *Controller) Start() {
	if c.stopCh != nil {
		return
	}

	c.stopCh = make(chan struct{})
	// Arm the leading edges only for a COLD start (some pair has never been fetched)
	coldStart := c.hasNeverFetchedBalances()
	c.firstFetchPending.Store(coldStart)
	c.tokenListsColdFetchPending.Store(coldStart)

	c.startAccountsWatcher()
	c.startNetworksWatcher()
	c.startWalletEventsWatcher()
	c.startFetcher()
}

func (c *Controller) Stop() {
	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
	}

	c.cancelAllChainFetches()
	c.stopWalletEventsWatcher()

	// After the watcher is stopped (Stop waits for the callback to return), so a
	// token-lists event racing teardown can't leave a leading edge armed.
	c.firstFetchPending.Store(false)
	c.tokenListsColdFetchPending.Store(false)
}

// hasNeverFetchedBalances reports whether any (account, chain) pair has a
// never-fetched native or ERC20 balance.
//
// Collectible types are deliberately excluded: an account owning no
// collectibles on a chain never produces an ERC721/ERC1155 fetch job
// (buildMultiStandardFetcherFetchConfigs only adds a map entry for a non-empty
// list, and the SDK builds one job per entry), so their state stays
// NeverFetched forever and would make every start look cold.
func (c *Controller) hasNeverFetchedBalances() bool {
	accounts, err := c.getAllAccounts()
	if err != nil {
		return true
	}
	networks, err := c.getAllNetworks()
	if err != nil {
		return true
	}
	for _, account := range accounts {
		for _, network := range networks {
			key := BalancesKey{Account: account, ChainID: network}
			if _, state, err := c.storage.GetNativeBalance(context.Background(), key); err != nil || state.FetchedAt == NeverFetched {
				return true
			}
			if _, state, err := c.storage.GetERC20Balances(context.Background(), key); err != nil || state.FetchedAt == NeverFetched {
				return true
			}
		}
	}
	return false
}

func (c *Controller) startChainFetch(chainID uint64) (context.Context, context.CancelFunc) {
	c.chainFetchMu.Lock()
	defer c.chainFetchMu.Unlock()
	if cancel, ok := c.chainFetchCancels[chainID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.chainFetchCancels[chainID] = cancel
	return ctx, cancel
}

func (c *Controller) cancelAllChainFetches() {
	c.chainFetchMu.Lock()
	defer c.chainFetchMu.Unlock()
	for chainID, cancel := range c.chainFetchCancels {
		cancel()
		delete(c.chainFetchCancels, chainID)
	}
}

func (c *Controller) GetPublisher() *pubsub.Publisher {
	return c.publisher
}

func (c *Controller) startAccountsWatcher() {
	if c.accountsPublisher == nil {
		return
	}

	addedCh, addedUnsubFn := pubsub.Subscribe[accountsevent.AccountsAddedEvent](c.accountsPublisher, 10)
	go func() {
		defer panics.LogOnPanic()
		defer addedUnsubFn()
		for {
			select {
			case <-c.stopCh:
				return
			case _, ok := <-addedCh:
				if !ok {
					return
				}
				c.triggerFetch()
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
				c.triggerFetch()
			}
		}
	}()
}

func (c *Controller) startWalletEventsWatcher() {
	if c.walletEventsWatcher != nil {
		return
	}

	// Respond to any sent transaction update
	walletEventCb := func(event walletevent.Event) {
		switch event.Type {
		case pendingtxtracker.EventPendingTransactionUpdate:
			var p pendingtxtracker.PendingTxUpdatePayload
			err := json.Unmarshal([]byte(event.Message), &p)
			if err != nil {
				return
			}
			if p.Deleted {
				// Some pending transaction moved to its
				// final state, trigger a fetch
				fetchConfig := make(FetchConfig)
				accounts := make([]common.Address, 0)
				networks := make([]uint64, 0)
				for _, sentTransaction := range p.TxDetails.SentTransactions {
					accounts = append(accounts, common.BytesToAddress(sentTransaction.FromAddress.Bytes()))
					accounts = append(accounts, common.BytesToAddress(sentTransaction.ToAddress.Bytes()))
					networks = append(networks, sentTransaction.FromChain)
					networks = append(networks, sentTransaction.ToChain)
				}
				for _, account := range accounts {
					for _, network := range networks {
						c.logger.Debug("triggering fetch due to pending transaction update", zap.Uint64("chainID", network), zap.String("account", logutils.TruncateWithDot(account.String())))
						// TODO: Add only relevant result types to the transaction type
						// For now we simply fetch all result types. It shouldn't normally cause additional calls
						// due to use of multicall
						fetchConfig[BalancesKey{Account: account, ChainID: network}] = []multistandardfetcher.ResultType{
							multistandardfetcher.ResultTypeNative,
							multistandardfetcher.ResultTypeERC20,
							multistandardfetcher.ResultTypeERC721,
							multistandardfetcher.ResultTypeERC1155,
						}
					}
				}
				c.TriggerFetchWithConfig(fetchConfig)
			}
		case pendingtxtracker.EventPendingTransactionStatusChanged:
			var p pendingtxtracker.StatusChangedPayload
			err := json.Unmarshal([]byte(event.Message), &p)
			if err != nil {
				return
			}
			if p.Status != ac.Success {
				return
			}

			fetchConfig, err := c.buildFetchConfigFromTransactionStatus(&p)
			if err != nil {
				c.logger.Error("failed to build fetch config from transaction status", zap.Error(err))
				return
			}
			if len(fetchConfig) > 0 {
				c.fetchImmediatelyWithConfig(fetchConfig)
			}
		case walletevent.EventTokenListsUpdated:
			if c.tokenListsColdFetchPending.CompareAndSwap(true, false) {
				c.firstFetchPending.Store(true)
			}
			c.TriggerFullFetch()
		default:
			// Unrelated event, do not trigger a fetch
			return
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
				c.TriggerFullFetch()
			}
		}
	}()
	// Trigger initial fetch
	c.TriggerFullFetch()
}

// Triggers delayed fetch for some accounts/networks/types, according to fetchConfig
func (c *Controller) TriggerFetchWithConfig(fetchConfig FetchConfig) {
	c.upsertFetchConfig(fetchConfig)
	c.triggerFetch()
}

// Triggers delayed fetch for all accounts/networks/types
func (c *Controller) TriggerFullFetch() {
	c.mu.Lock()
	c.pendingFullFetch = true
	c.mu.Unlock()
	c.triggerFetch()
}

func (c *Controller) triggerFetch() {
	if c.firstFetchPending.CompareAndSwap(true, false) {
		go func() {
			defer panics.LogOnPanic()
			c.fetchNow()
		}()
		return
	}
	c.fetchDebounceFn(c.fetchNow)
}

// Needs to fetch if the balance has never been fetched or if the last fetch was more than fetchPeriod ago
func (c *Controller) needsToFetch(state State) bool {
	return state.FetchedAt == NeverFetched || state.FetchedAt+int64(c.config.FetchPeriod.Seconds()) < time.Now().Unix()
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

// If fullFetch is true, fetch all balance types for all accounts and networks.
// If fullFetch is false, fetch only the balance types for the accounts and networks that need to be fetched based on the last fetch time and
// the pendingFetchConfig.
func (c *Controller) computeBalancesToFetch(fullFetch bool, pendingFetchConfig FetchConfig) FetchConfig {
	balancesToFetch := make(map[BalancesKey][]multistandardfetcher.ResultType)

	accounts, err := c.getAllAccounts()
	if err != nil {
		return nil
	}

	networks, err := c.getAllNetworks()
	if err != nil {
		return nil
	}

	// Clear up dangling data from storage
	c.storage.ClearMissingAccounts(context.Background(), accounts)
	c.storage.ClearMissingChains(context.Background(), networks)

	for _, account := range accounts {
		for _, network := range networks {
			key := BalancesKey{Account: account, ChainID: network}

			resultTypes := make([]multistandardfetcher.ResultType, 0, 4)
			if fullFetch {
				resultTypes = []multistandardfetcher.ResultType{
					multistandardfetcher.ResultTypeNative,
					multistandardfetcher.ResultTypeERC20,
					multistandardfetcher.ResultTypeERC721,
					multistandardfetcher.ResultTypeERC1155,
				}
			} else {
				// Insert values from pendingFetchConfig
				resultTypes = append(resultTypes, pendingFetchConfig[key]...)

				// Add remaining values that need to be fetched
				stateMap := make(map[multistandardfetcher.ResultType]State)
				if _, state, err := c.storage.GetNativeBalance(context.Background(), key); err == nil {
					stateMap[multistandardfetcher.ResultTypeNative] = state
				}
				if _, state, err := c.storage.GetERC20Balances(context.Background(), key); err == nil {
					stateMap[multistandardfetcher.ResultTypeERC20] = state
				}
				if _, state, err := c.storage.GetERC721Balances(context.Background(), key); err == nil {
					stateMap[multistandardfetcher.ResultTypeERC721] = state
				}
				if _, state, err := c.storage.GetERC1155Balances(context.Background(), key); err == nil {
					stateMap[multistandardfetcher.ResultTypeERC1155] = state
				}

				for resultType, state := range stateMap {
					if slices.Contains(resultTypes, resultType) {
						continue
					}
					if c.needsToFetch(state) {
						resultTypes = append(resultTypes, resultType)
					}
				}
			}

			balancesToFetch[key] = resultTypes
		}
	}

	return balancesToFetch
}

func (c *Controller) upsertFetchConfig(fetchConfig FetchConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, resultTypes := range fetchConfig {
		pendingFetchConfig := c.pendingFetchConfig[key]
		for _, resultType := range resultTypes {
			if !slices.Contains(pendingFetchConfig, resultType) {
				pendingFetchConfig = append(pendingFetchConfig, resultType)
			}
		}
		c.pendingFetchConfig[key] = pendingFetchConfig
	}
}

func (c *Controller) popPendingFetchValues() (bool, FetchConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fullFetch := c.pendingFullFetch
	c.pendingFullFetch = false
	fetchConfig := c.pendingFetchConfig
	c.pendingFetchConfig = make(FetchConfig)
	return fullFetch, fetchConfig
}

func (c *Controller) executeFetchConfigs(fetchConfigs map[uint64]multistandardfetcher.FetchConfig) {
	for chainID, chainFetchConfig := range fetchConfigs {
		for _, address := range chainFetchConfig.Native {
			c.logger.Debug(
				"fetching Native balance",
				zap.Uint64("chainID", chainID),
				zap.String("address", logutils.TruncateWithDot(address.String())),
			)
		}
		for address, tokenList := range chainFetchConfig.ERC20 {
			c.logger.Debug(
				"fetching ERC20 balance",
				zap.Uint64("chainID", chainID),
				zap.String("address", logutils.TruncateWithDot(address.String())),
				zap.Int("tokenList", len(tokenList)),
			)
		}
		for address, tokenList := range chainFetchConfig.ERC721 {
			c.logger.Debug(
				"fetching ERC721 balance",
				zap.Uint64("chainID", chainID),
				zap.String("address", logutils.TruncateWithDot(address.String())),
				zap.Int("tokenList", len(tokenList)),
			)
		}
		for address, tokenList := range chainFetchConfig.ERC1155 {
			c.logger.Debug(
				"fetching ERC1155 balance",
				zap.Uint64("chainID", chainID),
				zap.String("address", logutils.TruncateWithDot(address.String())),
				zap.Int("tokenList", len(tokenList)),
			)
		}

		ctx, cancel := c.startChainFetch(chainID)

		resultsCh, err := c.fetcher.FetchBalances(ctx, chainID, chainFetchConfig)
		if err != nil {
			cancel()
			c.logger.Error("failed to fetch balances", zap.Uint64("chainID", chainID), zap.Error(err))
			c.sendEventBalanceFetchFailedToStart(chainID, chainFetchConfig, err)
			continue
		}

		c.logger.Debug("fetch started", zap.Uint64("chainID", chainID))
		c.sendEventBalanceFetchStarted(chainID, chainFetchConfig)

		go func(fetchChainID uint64, ctx context.Context, cancel context.CancelFunc) {
			defer panics.LogOnPanic()
			defer cancel()
			for {
				select {
				case <-c.stopCh:
					return
				case result, ok := <-resultsCh:
					if !ok {
						return
					}
					c.handleFetchResult(ctx, fetchChainID, result)
				}
			}
		}(chainID, ctx, cancel)
	}
}

func (c *Controller) fetchNow() {
	c.logger.Debug("starting fetch")
	// Get the fullFetch flag + fetchConfig and reset them
	fullFetch, fetchConfig := c.popPendingFetchValues()

	balancesToFetch := c.computeBalancesToFetch(fullFetch, fetchConfig)
	fetchConfigs := c.buildMultiStandardFetcherFetchConfigs(balancesToFetch)

	if len(fetchConfigs) == 0 {
		return
	}

	c.executeFetchConfigs(fetchConfigs)
}

// fetchImmediatelyWithConfig triggers a balance fetch for specific accounts/chains immediately,
// bypassing the debounce. Used when a transaction has just confirmed and we need fresh balances now.
func (c *Controller) fetchImmediatelyWithConfig(fetchConfig FetchConfig) {
	go func() {
		defer panics.LogOnPanic()
		fetchConfigs := c.buildMultiStandardFetcherFetchConfigs(fetchConfig)
		if len(fetchConfigs) == 0 {
			return
		}
		c.executeFetchConfigs(fetchConfigs)
	}()
}

// buildFetchConfigFromTransactionStatus builds a FetchConfig for the from/to addresses and chains
// from a successful transaction's SendDetails.
func (c *Controller) buildFetchConfigFromTransactionStatus(p *pendingtxtracker.StatusChangedPayload) (FetchConfig, error) {
	fetchConfig := make(FetchConfig)

	addBalanceKeyToFetchConfig := func(addr common.Address, chainID uint64, includeCollectibles bool) {
		if chainID == 0 || addr == walletcommon.ZeroAddress() {
			return
		}
		key := BalancesKey{Account: addr, ChainID: chainID}
		fetchConfig[key] = []multistandardfetcher.ResultType{
			multistandardfetcher.ResultTypeNative,
			multistandardfetcher.ResultTypeERC20,
		}
		if includeCollectibles {
			fetchConfig[key] = append(fetchConfig[key], multistandardfetcher.ResultTypeERC721, multistandardfetcher.ResultTypeERC1155)
		}
	}

	addBalanceKeyOrKeysToFetchConfigIfParamsAreEmpty := func(addr common.Address, chainID uint64, includeCollectibles bool) error {
		var (
			addresses []common.Address
			chainIDs  []uint64
			err       error
		)
		if addr == walletcommon.ZeroAddress() {
			addresses, err = c.getAllAccounts()
			if err != nil {
				return err
			}
		} else {
			addresses = []common.Address{addr}
		}

		if chainID == 0 {
			chainIDs, err = c.getAllNetworks()
			if err != nil {
				return err
			}
		} else {
			chainIDs = []uint64{chainID}
		}

		for _, address := range addresses {
			for _, chainID := range chainIDs {
				addBalanceKeyToFetchConfig(address, chainID, includeCollectibles)
			}
		}
		return nil
	}

	if p.SendDetails != nil {
		includeCollectibles := sendtype.SendType(p.SendDetails.SendType).IsCollectiblesTransfer()
		err := addBalanceKeyOrKeysToFetchConfigIfParamsAreEmpty(common.Address(p.SendDetails.FromAddress), p.SendDetails.FromChain, includeCollectibles)
		if err != nil {
			return nil, err
		}
		err = addBalanceKeyOrKeysToFetchConfigIfParamsAreEmpty(common.Address(p.SendDetails.ToAddress), p.SendDetails.ToChain, includeCollectibles)
		if err != nil {
			return nil, err
		}
	} else {
		err := addBalanceKeyOrKeysToFetchConfigIfParamsAreEmpty(walletcommon.ZeroAddress(), 0, true)
		if err != nil {
			return nil, err
		}
	}

	return fetchConfig, nil
}
