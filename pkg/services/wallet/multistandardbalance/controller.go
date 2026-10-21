package multistandardbalance

//go:generate go tool mockgen -package=mock_multistandardbalance -source=controller.go -destination=mock/controller.go

import (
	"context"
	"encoding/json"
	"maps"
	"math/big"
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
	tokentypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

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

// HeldTokensProvider gives the tokens read with the held set whatever their
// balance: custom tokens and the tokens of pending transactions.
type HeldTokensProvider interface {
	GetHeldTokens(chainID uint64, account common.Address) ([]common.Address, error)
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
	// FetchPeriod is how often the held set is read while balances are on
	// screen, IdleFetchPeriod while they are not (SetActive(false)). Zero
	// IdleFetchPeriod keeps FetchPeriod.
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
	policy.Schedule = refresh.HeldEveryTick
	policy.OnScreenPeriod = config.FetchPeriod
	policy.HiddenPeriod = hiddenPeriod
	policy.Debounce = config.FetchDebounceTime
	return policy
}

type Controller struct {
	config  ControllerConfig
	storage Storage
	fetcher BalanceFetcher

	accountsProvider        AccountsProvider
	accountsPublisher       *pubsub.Publisher
	networksProvider        NetworksProvider
	tokenListProvider       TokenListProvider
	collectibleListProvider CollectiblesListProvider
	heldTokensProvider      HeldTokensProvider
	changeSource            refresh.ChangeSource
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

	stopWatching func()

	// detected are the tokens transfer detection found, kept in the held set
	// until read as zero.
	detectedMu sync.Mutex
	detected   map[BalancesKey]map[common.Address]struct{}

	// chainFetches are the fetches in flight by chain. A chain reads one fetch
	// at a time; reads for it wait for the fetch in flight to end.
	chainFetchMu sync.Mutex
	chainFetches map[uint64]*chainFetchState

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
	heldTokensProvider HeldTokensProvider,
	changeSource refresh.ChangeSource,
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
		heldTokensProvider:      heldTokensProvider,
		changeSource:            changeSource,
		lastBlockManager:        lastBlockManager,
		publisher:               pubsub.NewPublisher(),
		policy:                  refresh.NewPolicy(policyConfig(config)),
		chainFetches:            make(map[uint64]*chainFetchState),
		detected:                make(map[BalancesKey]map[common.Address]struct{}),
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
	c.stopWatching = c.changeSource.Watch(c.handleChange)
	c.handle(refresh.Started{})
}

func (c *Controller) Stop() {
	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
		c.stopWatching()
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
		if scansForChanges(ev, decision.Reads) {
			c.changeSource.Scan()
		}
	}
}

// scansForChanges reports whether the change source looks for changes after
// ev: with every periodic refresh, and at once when the user asks for one.
func scansForChanges(ev refresh.Event, reads []refresh.Read) bool {
	if _, requested := ev.(refresh.RefreshRequested); requested {
		return true
	}
	return slices.ContainsFunc(reads, func(r refresh.Read) bool {
		return r.Scope == (refresh.Scope{}) && (r.Tokens == refresh.Held || r.Tokens == refresh.Full)
	})
}

// handleChange takes what the change source finds.
func (c *Controller) handleChange(ev refresh.Event) {
	if transfer, ok := ev.(refresh.TransferDetected); ok && len(transfer.Contracts) > 0 {
		key := BalancesKey{Account: transfer.Pair.Account, ChainID: transfer.Pair.Chain}
		c.detectedMu.Lock()
		if c.detected[key] == nil {
			c.detected[key] = make(map[common.Address]struct{})
		}
		for _, token := range transfer.Contracts {
			c.detected[key][token] = struct{}{}
		}
		c.detectedMu.Unlock()
	}
	c.handle(ev)
}

func (c *Controller) detectedTokens(key BalancesKey) []common.Address {
	c.detectedMu.Lock()
	defer c.detectedMu.Unlock()
	return slices.Collect(maps.Keys(c.detected[key]))
}

// forgetDetected drops the detected tokens read as zero.
func (c *Controller) forgetDetected(key BalancesKey, balances map[ContractAddress]*big.Int) {
	c.detectedMu.Lock()
	defer c.detectedMu.Unlock()
	for token, balance := range balances {
		if balance != nil && balance.Sign() == 0 {
			delete(c.detected[key], token)
		}
	}
	if len(c.detected[key]) == 0 {
		delete(c.detected, key)
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

type chainFetchState struct {
	cancel context.CancelFunc
	// waiting are the reads asked for while the fetch was in flight.
	waiting []refresh.Read
}

// startChainFetch starts a fetch of chainID, or keeps reads for later when a
// fetch of chainID is in flight.
func (c *Controller) startChainFetch(chainID uint64, reads []refresh.Read) (context.Context, *chainFetchState, bool) {
	c.chainFetchMu.Lock()
	defer c.chainFetchMu.Unlock()
	if state, ok := c.chainFetches[chainID]; ok {
		state.waiting = append(state.waiting, reads...)
		return nil, nil, false
	}
	ctx, cancel := context.WithCancel(traffic.WithSource(context.Background(), traffic.Balances))
	state := &chainFetchState{cancel: cancel}
	c.chainFetches[chainID] = state
	return ctx, state, true
}

// endChainFetch makes the reads that waited for the fetch of chainID.
func (c *Controller) endChainFetch(chainID uint64, state *chainFetchState) {
	c.chainFetchMu.Lock()
	state.cancel()
	if c.chainFetches[chainID] != state {
		c.chainFetchMu.Unlock()
		return
	}
	delete(c.chainFetches, chainID)
	waiting := state.waiting
	c.chainFetchMu.Unlock()

	if len(waiting) > 0 {
		c.readOn(waiting, &chainID)
	}
}

func (c *Controller) cancelAllChainFetches() {
	c.chainFetchMu.Lock()
	defer c.chainFetchMu.Unlock()
	for chainID, state := range c.chainFetches {
		state.cancel()
		delete(c.chainFetches, chainID)
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
			for _, ev := range transactionsEnded(sentTransactionsTransfers(p.SentTransactions), false) {
				c.handle(ev)
			}
		case pendingtxtracker.EventPendingTransactionStatusChanged:
			var p pendingtxtracker.StatusChangedPayload
			if err := json.Unmarshal([]byte(event.Message), &p); err != nil || p.Status != ac.Success {
				return
			}
			for _, ev := range confirmedTransaction(&p) {
				c.handle(ev)
			}
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

// transfer is one side of a transaction: the account, the chain and the token
// it moved there.
type transfer struct {
	account  common.Address
	chain    uint64
	tokenKey string
}

func sentTransactionsTransfers(transactions []*responses.RouterSentTransaction) []transfer {
	transfers := make([]transfer, 0, 2*len(transactions))
	for _, tx := range transactions {
		transfers = append(transfers,
			transfer{common.Address(tx.FromAddress), tx.FromChain, tx.FromTokenKey},
			transfer{common.Address(tx.ToAddress), tx.ToChain, tx.ToTokenKey},
		)
	}
	return transfers
}

// confirmedTransaction reads the sender on the source chain and the recipient
// on the destination chain, or every native coin when the details are unknown.
func confirmedTransaction(p *pendingtxtracker.StatusChangedPayload) []refresh.TransactionEnded {
	if p.SendDetails == nil {
		return []refresh.TransactionEnded{{Pairs: []refresh.Pair{{}}, Confirmed: true}}
	}
	fromToken, toToken := p.SendDetails.FromTokenKey, p.SendDetails.ToTokenKey
	if sendtype.SendType(p.SendDetails.SendType).IsCollectiblesTransfer() {
		fromToken, toToken = "", ""
	}
	return transactionsEnded([]transfer{
		{common.Address(p.SendDetails.FromAddress), p.SendDetails.FromChain, fromToken},
		{common.Address(p.SendDetails.ToAddress), p.SendDetails.ToChain, toToken},
	}, true)
}

// transactionsEnded makes one event per chain, so that each chain is read by
// one fetch.
func transactionsEnded(transfers []transfer, confirmed bool) []refresh.TransactionEnded {
	var events []refresh.TransactionEnded
	for _, t := range transfers {
		if t.account == walletcommon.ZeroAddress() || t.chain == 0 {
			continue
		}
		i := slices.IndexFunc(events, func(ev refresh.TransactionEnded) bool { return ev.Pairs[0].Chain == t.chain })
		if i < 0 {
			events = append(events, refresh.TransactionEnded{Confirmed: confirmed})
			i = len(events) - 1
		}
		ev := &events[i]
		pair := refresh.Pair{Account: t.account, Chain: t.chain}
		if !slices.Contains(ev.Pairs, pair) {
			ev.Pairs = append(ev.Pairs, pair)
		}
		if chain, contract, ok := tokentypes.ChainAndAddressFromTokenKey(t.tokenKey); ok && chain == t.chain &&
			contract != walletcommon.ZeroAddress() && !slices.Contains(ev.Contracts, contract) {
			ev.Contracts = append(ev.Contracts, contract)
		}
	}
	return events
}

// SetActive tells whether someone looks at balances: while not, the held set
// is read every IdleFetchPeriod and the hourly full refresh waits. Becoming
// active reads at once what is due.
func (c *Controller) SetActive(active bool) {
	if c.hidden.Swap(!active) != !active {
		c.logger.Info("balances refresh period changed", zap.Bool("active", active))
	}
	c.handle(refresh.OnScreenChanged{OnScreen: active})
}

// TriggerFullFetch reads the held set after the debounce, and every balance
// when the last full refresh is older than ten minutes.
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

// read makes the reads of one decision with one fetch per chain.
func (c *Controller) read(reads []refresh.Read) {
	c.readOn(reads, nil)
}

// readOn makes reads, only on chain when it is set. What a read covers is
// worked out now, from what storage holds.
func (c *Controller) readOn(reads []refresh.Read, chain *uint64) {
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

	plan := newReadPlan()
	for _, r := range reads {
		c.addRead(plan, r, accounts, networks)
	}
	if chain != nil {
		for key := range plan.pairs {
			if key.ChainID != *chain {
				delete(plan.pairs, key)
			}
		}
	}
	c.executeChainFetches(c.buildChainFetches(plan), reads)
}

func (c *Controller) executeChainFetches(fetches map[uint64]*chainFetch, reads []refresh.Read) {
	for chainID, fetch := range fetches {
		chainFetchConfig := fetch.config
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

		ctx, state, started := c.startChainFetch(chainID, reads)
		if !started {
			c.logger.Debug("balance read waits for the fetch in flight", zap.Uint64("chainID", chainID))
			continue
		}

		resultsCh, err := c.fetcher.FetchBalances(ctx, chainID, chainFetchConfig)
		if err != nil {
			c.endChainFetch(chainID, state)
			c.logger.Error("failed to fetch balances", zap.Uint64("chainID", chainID), zap.Error(err))
			c.sendEventBalanceFetchFailedToStart(chainID, chainFetchConfig, err)
			c.fullRefreshFailed(chainID, fetch)
			continue
		}

		c.logger.Debug("fetch started", zap.Uint64("chainID", chainID))
		c.sendEventBalanceFetchStarted(chainID, chainFetchConfig)

		go func(fetchChainID uint64, ctx context.Context, state *chainFetchState) {
			defer panics.LogOnPanic()
			defer c.endChainFetch(fetchChainID, state)
			failed := false
			for {
				select {
				case <-ctx.Done():
					c.fullRefreshFailed(fetchChainID, fetch)
					return
				case result, ok := <-resultsCh:
					if !ok {
						if failed {
							c.fullRefreshFailed(fetchChainID, fetch)
						}
						return
					}
					if !c.handleFetchResult(ctx, fetchChainID, result, fetch) {
						failed = true
					}
				}
			}
		}(chainID, ctx, state)
	}
}

// fullRefreshFailed tells the policy when fetch was a full refresh of chainID.
func (c *Controller) fullRefreshFailed(chainID uint64, fetch *chainFetch) {
	if len(fetch.listed) > 0 {
		c.handle(refresh.FullRefreshFailed{Chain: chainID})
	}
}
