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
	"github.com/status-im/status-go/internal/traffic"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"
	"github.com/status-im/status-go/pkg/services/networks"
	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/pkg/services/wallet/responses"
	"github.com/status-im/status-go/pkg/services/wallet/router/sendtype"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/balance/refresh"

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
	// IdleFetchPeriod is the full fetch period while the client says nobody
	// is looking at balances (SetActive(false)). A full fetch reads every
	// token of every list on every chain for every account, so it is most of
	// the RPC traffic. Zero keeps FetchPeriod.
	IdleFetchPeriod time.Duration
}

func DefaultControllerConfig() ControllerConfig {
	return ControllerConfig{
		FetchDebounceTime: 10 * time.Second,
		FetchPeriod:       2 * time.Minute,
		IdleFetchPeriod:   15 * time.Minute,
	}
}

func policyConfig(config ControllerConfig) refresh.Config {
	hiddenPeriod := config.IdleFetchPeriod
	if hiddenPeriod == 0 {
		hiddenPeriod = config.FetchPeriod
	}
	policy := refresh.DefaultConfig()
	policy.Schedule = refresh.FullEveryTick
	policy.OnScreenPeriod = config.FetchPeriod
	policy.HiddenPeriod = hiddenPeriod
	policy.Debounce = config.FetchDebounceTime
	return policy
}

type FetchConfig map[BalancesKey][]multistandardfetcher.ResultType

type Controller struct {
	config  ControllerConfig
	storage Storage
	fetcher BalanceFetcher

	accountsProvider        AccountsProvider
	accountsPublisher       *pubsub.Publisher
	networksProvider        NetworksProvider
	tokenListProvider       TokenListProvider
	collectibleListProvider CollectiblesListProvider
	lastBlockManager        LastBlockManager
	walletFeed              *event.Feed
	walletEventsWatcher     *walletevent.Watcher

	publisher *pubsub.Publisher

	stopCh chan struct{}

	// policyMu guards the policy, which is not safe for concurrent use, and
	// the timer that calls it back.
	policyMu sync.Mutex
	policy   *refresh.Policy
	running  bool
	timer    *time.Timer

	hidden atomic.Bool

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
		policy:                  refresh.NewPolicy(policyConfig(config)),
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
	c.startAccountsWatcher()
	c.startNetworksWatcher()
	c.startWalletEventsWatcher()
	c.handle(refresh.Started{})
}

func (c *Controller) Stop() {
	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
		c.handle(refresh.Stopped{})
	}

	c.cancelAllChainFetches()
	c.stopWalletEventsWatcher()
}

// handle passes ev to the policy, makes the reads it asks for and sets the
// timer to its next tick. While stopped, the policy hears events but nothing
// is read.
func (c *Controller) handle(ev refresh.Event) {
	c.policyMu.Lock()
	switch ev.(type) {
	case refresh.Started:
		c.running = true
	case refresh.Stopped:
		c.running = false
	case refresh.Tick:
		if !c.running {
			c.policyMu.Unlock()
			return
		}
	}
	decision := c.policy.Handle(time.Now(), ev)
	running := c.running
	c.setTimer(decision.WakeAt)
	c.policyMu.Unlock()

	if running {
		c.read(decision.Reads)
	}
}

func (c *Controller) setTimer(wakeAt time.Time) {
	if !c.running || wakeAt.IsZero() {
		if c.timer != nil {
			c.timer.Stop()
		}
		return
	}
	if c.timer == nil {
		c.timer = time.AfterFunc(time.Until(wakeAt), func() {
			defer panics.LogOnPanic()
			c.handle(refresh.Tick{})
		})
		return
	}
	c.timer.Reset(time.Until(wakeAt))
}

func (c *Controller) startChainFetch(chainID uint64) (context.Context, context.CancelFunc) {
	c.chainFetchMu.Lock()
	defer c.chainFetchMu.Unlock()
	if cancel, ok := c.chainFetchCancels[chainID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(traffic.WithSource(context.Background(), traffic.Balances))
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
	stopCh := c.stopCh
	go func() {
		defer panics.LogOnPanic()
		defer addedUnsubFn()
		for {
			select {
			case <-stopCh:
				return
			case _, ok := <-addedCh:
				if !ok {
					return
				}
				c.handle(refresh.AccountsOrChainsChanged{})
			}
		}
	}()
}

func (c *Controller) startNetworksWatcher() {
	if c.networksProvider == nil {
		return
	}

	ch, unsubFn := pubsub.Subscribe[networks.EventActiveNetworksChanged](c.networksProvider.GetPublisher(), 10)

	stopCh := c.stopCh
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
				c.handle(refresh.AccountsOrChainsChanged{})
			}
		}
	}()
}

func (c *Controller) startWalletEventsWatcher() {
	if c.walletEventsWatcher != nil {
		return
	}

	walletEventCb := func(event walletevent.Event) {
		switch event.Type {
		case pendingtxtracker.EventPendingTransactionUpdate:
			var p pendingtxtracker.PendingTxUpdatePayload
			if err := json.Unmarshal([]byte(event.Message), &p); err != nil || !p.Deleted {
				return
			}
			if pairs := sentTransactionsPairs(p.SentTransactions); len(pairs) > 0 {
				c.handle(refresh.TransactionEnded{Pairs: pairs})
			}
		case pendingtxtracker.EventPendingTransactionStatusChanged:
			var p pendingtxtracker.StatusChangedPayload
			if err := json.Unmarshal([]byte(event.Message), &p); err != nil || p.Status != ac.Success {
				return
			}
			c.handle(confirmedTransaction(&p))
		case walletevent.EventTokenListsUpdated:
			c.handle(refresh.TokenListsUpdated{})
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

// sentTransactionsPairs are the senders and recipients of the transactions on
// their source and destination chains.
func sentTransactionsPairs(transactions []*responses.RouterSentTransaction) []refresh.Pair {
	var accounts []common.Address
	var chains []uint64
	for _, tx := range transactions {
		for _, account := range []common.Address{common.Address(tx.FromAddress), common.Address(tx.ToAddress)} {
			if account != walletcommon.ZeroAddress() && !slices.Contains(accounts, account) {
				accounts = append(accounts, account)
			}
		}
		for _, chain := range []uint64{tx.FromChain, tx.ToChain} {
			if chain != 0 && !slices.Contains(chains, chain) {
				chains = append(chains, chain)
			}
		}
	}
	pairs := make([]refresh.Pair, 0, len(accounts)*len(chains))
	for _, account := range accounts {
		for _, chain := range chains {
			pairs = append(pairs, refresh.Pair{Account: account, Chain: chain})
		}
	}
	return pairs
}

// confirmedTransaction reads the sender on the source chain and the recipient
// on the destination chain, or every balance when the details are unknown.
func confirmedTransaction(p *pendingtxtracker.StatusChangedPayload) refresh.TransactionEnded {
	if p.SendDetails == nil {
		return refresh.TransactionEnded{Pairs: []refresh.Pair{{}}, Confirmed: true}
	}
	kinds := refresh.Native | refresh.ERC20
	if sendtype.SendType(p.SendDetails.SendType).IsCollectiblesTransfer() {
		kinds |= refresh.ERC721 | refresh.ERC1155
	}
	return refresh.TransactionEnded{
		Pairs: []refresh.Pair{
			{Account: common.Address(p.SendDetails.FromAddress), Chain: p.SendDetails.FromChain},
			{Account: common.Address(p.SendDetails.ToAddress), Chain: p.SendDetails.ToChain},
		},
		Kinds:     kinds,
		Confirmed: true,
	}
}

// SetActive tells whether someone looks at balances: while not, full fetches
// run at IdleFetchPeriod instead of FetchPeriod. Becoming active fetches at
// once when the last periodic fetch is older than FetchPeriod.
func (c *Controller) SetActive(active bool) {
	if c.hidden.Swap(!active) != !active {
		c.logger.Info("balances refresh period changed", zap.Bool("active", active))
	}
	c.handle(refresh.OnScreenChanged{OnScreen: active})
}

// TriggerFullFetch reads every balance after the debounce.
func (c *Controller) TriggerFullFetch() {
	c.handle(refresh.RefreshRequested{})
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

// read makes the reads of one decision with one fetch per chain, so that they
// do not cancel each other.
func (c *Controller) read(reads []refresh.Read) {
	if len(reads) == 0 {
		return
	}

	accounts, err := c.getAllAccounts()
	if err != nil {
		c.logger.Error("failed to get accounts to read balances of", zap.Error(err))
		return
	}
	networks, err := c.getAllNetworks()
	if err != nil {
		c.logger.Error("failed to get networks to read balances on", zap.Error(err))
		return
	}

	// Clear up dangling data from storage
	c.storage.ClearMissingAccounts(context.Background(), accounts)
	c.storage.ClearMissingChains(context.Background(), networks)

	balancesToFetch := make(FetchConfig)
	for _, r := range reads {
		c.addRead(balancesToFetch, r, accounts, networks)
	}

	fetchConfigs := c.buildMultiStandardFetcherFetchConfigs(balancesToFetch)
	if len(fetchConfigs) == 0 {
		return
	}
	c.executeFetchConfigs(fetchConfigs)
}

func (c *Controller) addRead(balancesToFetch FetchConfig, r refresh.Read, accounts []common.Address, chains []uint64) {
	if r.Tokens != refresh.Full || r.Scope.Unread {
		c.logger.Warn("unsupported balance read", zap.Any("read", r))
		return
	}
	if r.Scope.Pair.Account != (common.Address{}) {
		accounts = []common.Address{r.Scope.Pair.Account}
	}
	if r.Scope.Pair.Chain != 0 {
		chains = []uint64{r.Scope.Pair.Chain}
	}
	for _, account := range accounts {
		for _, chain := range chains {
			key := BalancesKey{Account: account, ChainID: chain}
			for _, resultType := range resultTypes(r.Kinds) {
				if r.Scope.OlderThan > 0 && !c.readBefore(key, resultType, r.Scope.OlderThan) {
					continue
				}
				if !slices.Contains(balancesToFetch[key], resultType) {
					balancesToFetch[key] = append(balancesToFetch[key], resultType)
				}
			}
		}
	}
}

func resultTypes(kinds refresh.Kinds) []multistandardfetcher.ResultType {
	all := []struct {
		kind       refresh.Kinds
		resultType multistandardfetcher.ResultType
	}{
		{refresh.Native, multistandardfetcher.ResultTypeNative},
		{refresh.ERC20, multistandardfetcher.ResultTypeERC20},
		{refresh.ERC721, multistandardfetcher.ResultTypeERC721},
		{refresh.ERC1155, multistandardfetcher.ResultTypeERC1155},
	}
	selected := make([]multistandardfetcher.ResultType, 0, len(all))
	for _, k := range all {
		if kinds == 0 || kinds&k.kind != 0 {
			selected = append(selected, k.resultType)
		}
	}
	return selected
}

// readBefore reports whether the balances of key of resultType were never
// read or last read longer than d ago.
func (c *Controller) readBefore(key BalancesKey, resultType multistandardfetcher.ResultType, d time.Duration) bool {
	ctx := context.Background()
	var state State
	var err error
	switch resultType {
	case multistandardfetcher.ResultTypeNative:
		_, state, err = c.storage.GetNativeBalance(ctx, key)
	case multistandardfetcher.ResultTypeERC20:
		_, state, err = c.storage.GetERC20Balances(ctx, key)
	case multistandardfetcher.ResultTypeERC721:
		_, state, err = c.storage.GetERC721Balances(ctx, key)
	case multistandardfetcher.ResultTypeERC1155:
		_, state, err = c.storage.GetERC1155Balances(ctx, key)
	}
	if err != nil {
		return false
	}
	return state.FetchedAt == NeverFetched || state.FetchedAt+int64(d.Seconds()) < time.Now().Unix()
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
				case <-ctx.Done():
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
