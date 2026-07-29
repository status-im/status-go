package wallet

//go:generate go tool mockgen -package=mock_reader -source=reader.go -destination=mock/reader/reader.go

import (
	"context"
	"math"
	"math/big"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/bep/debounce"

	"golang.org/x/exp/maps"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc20"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/pubsub"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/services/wallet/token"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
	"github.com/status-im/status-go/services/wallet/transferdetector"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// EventWalletTickReload tells the client to re-read wallet balances/history: it
// is emitted (leading edge for first-ever data, otherwise debounced) after
// balance refreshes complete.
const EventWalletTickReload walletevent.EventType = "wallet-tick-reload"
const reloadDebounceTime = 5 * time.Second

type AccountAddress = tokenbalances.AccountAddress
type ContractAddress = tokenbalances.ContractAddress

type ReaderInterface interface {
	Start() error
	Stop()
	GetCachedBalances(chainIDs []uint64, addresses []common.Address) (map[common.Address][]tokentypes.StorageToken, error)
	GetLastTokenUpdateTimestamps() map[common.Address]int64
}

func NewReader(
	tokenManager token.ManagerInterface,
	marketManager *market.Manager,
	walletFeed *event.Feed,
	multistandardBalancePublisher *pubsub.Publisher,
	tokenBalancesStorage tokenbalances.Storage,
	transferDetectorPublisher *pubsub.Publisher) *Reader {
	return &Reader{
		tokenManager:                  tokenManager,
		marketManager:                 marketManager,
		multistandardBalancePublisher: multistandardBalancePublisher,
		tokenBalancesStorage:          tokenBalancesStorage,
		transferDetectorPublisher:     transferDetectorPublisher,
		walletFeed:                    walletFeed,
		reloadDebounceFn:              debounce.New(reloadDebounceTime),
	}
}

type Reader struct {
	tokenManager                   token.ManagerInterface
	marketManager                  *market.Manager
	multistandardBalancePublisher  *pubsub.Publisher
	transferDetectorPublisher      *pubsub.Publisher
	tokenBalancesStorage           tokenbalances.Storage
	walletFeed                     *event.Feed
	lastWalletTokenUpdateTimestamp sync.Map
	reloadDebounceFn               func(f func())
	// Armed by the balance-change watcher when a completion covers a
	// never-fetched pair — the first data a cold UI can show — so that refresh
	// announces the reload immediately instead of sitting out the debounce.
	// Warm refreshes (e.g. after a mobile resume) stay debounced.
	firstReloadPending atomic.Bool

	stopCh chan struct{}
}

func (r *Reader) Start() error {
	if r.stopCh != nil {
		return nil
	}

	r.stopCh = make(chan struct{})

	// Start balance change watcher
	r.startBalanceChangeWatcher()

	// Start transfer detection watcher
	r.startTransferDetectionWatcher()

	return nil
}

func (r *Reader) Stop() {
	if r.stopCh == nil {
		return
	}

	close(r.stopCh)
	r.stopCh = nil

	r.lastWalletTokenUpdateTimestamp = sync.Map{}
}

func (r *Reader) IsRunning() bool {
	return r.stopCh != nil
}

func (r *Reader) triggerWalletReload() {
	r.walletFeed.Send(walletevent.Event{
		Type: EventWalletTickReload,
	})
}

func (r *Reader) startBalanceChangeWatcher() {
	if r.multistandardBalancePublisher == nil {
		return
	}

	ch, unsub := pubsub.Subscribe[multistandardbalance.EventBalanceFetchFinished](r.multistandardBalancePublisher, 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsub()
		for {
			select {
			case <-r.stopCh:
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				switch event.ResultType {
				case multistandardfetcher.ResultTypeNative, multistandardfetcher.ResultTypeERC20:
					if !event.BalanceChanged && event.OldState.FetchedAt != multistandardbalance.NeverFetched {
						continue
					}

					if event.OldState.FetchedAt == multistandardbalance.NeverFetched {
						// First-ever data for this (chain, account) pair: a cold UI
						// is waiting on it, so the reload must not sit out the
						// debounce.
						r.firstReloadPending.Store(true)
					}
					r.refreshBalanceCache(context.TODO(), []uint64{event.Key.ChainID}, []common.Address{event.Key.Account})
				}
			}
		}
	}()
}

func (r *Reader) startTransferDetectionWatcher() {
	if r.transferDetectorPublisher == nil {
		return
	}

	ch, unsub := pubsub.Subscribe[transferdetector.EventTransferDetectionFinished](r.transferDetectorPublisher, 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsub()
		for {
			select {
			case <-r.stopCh:
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				for _, event := range msg.Events {
					switch event.EventKey {
					case eventlog.ERC20Transfer:
						unpackedEvent, ok := event.Unpacked.(erc20.Erc20Transfer)
						if !ok {
							logutils.ZapLogger().Error("failed to unpack ERC20Transfer event")
							continue
						}

						err := r.processERC20TransferEvent(msg.ChainID, unpackedEvent)
						if err != nil {
							logutils.ZapLogger().Error("failed to process ERC20Transfer event", zap.Error(err))
						}
					}
				}
			}
		}
	}()
}

func (r *Reader) processERC20TransferEvent(chainID uint64, event erc20.Erc20Transfer) error {
	// Find token in db or if this is a community token, find its metadata
	token, err := r.tokenManager.FindOrCreateTokenByAddress(context.TODO(), chainID, event.Raw.Address)
	if err != nil {
		return err
	}
	if token.CommunityData != nil {
		// Only add community tokens to the previously owned tokens list,
		// not any other spam token the account might've received.
		_, err = r.tokenManager.MarkAsPreviouslyOwnedToken(token, event.To)
		if err != nil {
			return err
		}
	}

	return nil
}

func tokensToBalancesPerChain(cachedTokens map[common.Address][]tokentypes.StorageToken) (map[uint64]map[common.Address]map[common.Address]*big.Int, error) {
	cachedBalancesPerChain := map[uint64]map[common.Address]map[common.Address]*big.Int{}
	for address, tokens := range cachedTokens {
		for _, token := range tokens {
			if _, ok := cachedBalancesPerChain[token.TokenChainID]; !ok {
				cachedBalancesPerChain[token.TokenChainID] = map[common.Address]map[common.Address]*big.Int{}
			}
			if _, ok := cachedBalancesPerChain[token.TokenChainID][address]; !ok {
				cachedBalancesPerChain[token.TokenChainID][address] = map[common.Address]*big.Int{}
			}

			bigBalance, ok := new(big.Int).SetString(token.RawBalance, 10)
			if !ok {
				return nil, gocommon.ErrBigIntSetFromString(token.RawBalance)
			}
			cachedBalancesPerChain[token.TokenChainID][address][token.TokenAddress] = bigBalance
		}
	}

	return cachedBalancesPerChain, nil
}

func (r *Reader) balancesToTokensByAddress(addresses []common.Address, allTokens []*tokentypes.Token,
	balances map[uint64]map[common.Address]map[common.Address]*big.Int, cachedTokens map[common.Address][]tokentypes.StorageToken,
) map[common.Address][]tokentypes.StorageToken {

	result := make(map[common.Address][]tokentypes.StorageToken)

	for _, address := range addresses {
		for _, token := range allTokens {
			isMandatoryToken := slices.Contains(walletcommon.MandatoryTokens(), token.Key())

			_, ok := balances[token.ChainID][address][token.Address]
			hasError := !ok

			hexBalance := &big.Int{}
			if tokenBalance := balances[token.ChainID][address][token.Address]; tokenBalance != nil && len(tokenBalance.Bytes()) <= 32 {
				// Balances should be represented by a uint256 (32 bytes). Some spam tokens return
				// fake values larger than that, so we ignore them.
				hexBalance = tokenBalance
			}

			balance := big.NewFloat(0.0)
			if hexBalance != nil {
				balance = new(big.Float).Quo(
					new(big.Float).SetInt(hexBalance),
					big.NewFloat(math.Pow(10, float64(token.Decimals))),
				)
			}

			isVisible := balance.Cmp(big.NewFloat(0.0)) > 0 || isCachedToken(cachedTokens, address, token)
			if !isVisible && !isMandatoryToken {
				continue
			}

			walletToken := tokentypes.StorageToken{
				TokenAddress: token.Address,
				TokenChainID: token.ChainID,
				RawBalance:   hexBalance.String(),
				Balance:      balance,
				HasError:     hasError,
			}

			result[address] = append(result[address], walletToken)
		}
	}

	return result
}

// GetLastTokenUpdateTimestamps returns last timestamps of successful token updates
func (r *Reader) GetLastTokenUpdateTimestamps() map[common.Address]int64 {
	result := make(map[common.Address]int64)

	r.lastWalletTokenUpdateTimestamp.Range(func(key, value interface{}) bool {
		addr, ok1 := key.(common.Address)
		timestamp, ok2 := value.(int64)
		if ok1 && ok2 {
			result[addr] = timestamp
		}
		return true
	})

	return result
}

func isCachedToken(cachedTokens map[common.Address][]tokentypes.StorageToken, address common.Address, token *tokentypes.Token) bool {
	if tokens, ok := cachedTokens[address]; ok {
		for _, t := range tokens {
			if types.TokenKey(t.TokenChainID, t.TokenAddress) != token.Key() {
				continue
			}
			return true
		}
	}
	return false
}

// getCachedWalletTokensWithoutMarketData returns the latest fetched balances, minus
// price information
func (r *Reader) getCachedWalletTokensWithoutMarketData() (map[common.Address][]tokentypes.StorageToken, error) {
	return r.tokenManager.GetCachedBalances()
}

func (r *Reader) updateTokenUpdateTimestamp(addresses []common.Address) {
	for _, address := range addresses {
		r.lastWalletTokenUpdateTimestamp.Store(address, time.Now().Unix())
	}
}

func (r *Reader) refreshBalanceCache(ctx context.Context, chainIDs []uint64, addresses []common.Address) {
	cachedTokens, err := r.getCachedWalletTokensWithoutMarketData()
	if err != nil {
		logutils.ZapLogger().Error("failed to get cached tokens", zap.Error(err))
		return
	}

	allTokens, err := r.tokenManager.GetTokensByChains(chainIDs)
	if err != nil {
		logutils.ZapLogger().Error("failed to get tokens list", zap.Error(err))
		return
	}

	balances, err := r.tokenBalancesStorage.GetBalances(ctx, allTokens, addresses)
	if err != nil {
		logutils.ZapLogger().Error("failed to update balances", zap.Error(err))
		return
	}

	tokens := r.balancesToTokensByAddress(addresses, allTokens, balances, cachedTokens)

	err = r.tokenManager.CacheBalances(tokens)
	if err != nil {
		logutils.ZapLogger().Error("failed to save tokens", zap.Error(err)) // Do not return error, as it is not critical
	}

	r.updateTokenUpdateTimestamp(addresses)

	if r.firstReloadPending.CompareAndSwap(true, false) {
		r.triggerWalletReload()
	} else {
		r.reloadDebounceFn(r.triggerWalletReload)
	}

	return
}

func (r *Reader) GetCachedBalances(chainIDs []uint64, addresses []common.Address) (map[common.Address][]tokentypes.StorageToken, error) {
	cachedTokens, err := r.getCachedWalletTokensWithoutMarketData()
	if err != nil {
		return nil, err
	}

	balances, err := tokensToBalancesPerChain(cachedTokens)
	if err != nil {
		return nil, err
	}

	tokensOfInterest := make(map[string]struct{}, 0)
	for _, cachedToken := range cachedTokens {
		for _, token := range cachedToken {
			if !slices.Contains(chainIDs, token.TokenChainID) {
				continue
			}
			tokensOfInterest[types.TokenKey(token.TokenChainID, token.TokenAddress)] = struct{}{}
		}
	}

	// add mandatory tokens for chainIDs to tokensOfInterest if not already present
	for _, chainID := range chainIDs {
		mandatoryTokens := walletcommon.MandatoryTokensByChainID(chainID)
		for _, tokenKey := range mandatoryTokens {
			if _, ok := tokensOfInterest[tokenKey]; ok {
				continue
			}
			tokensOfInterest[tokenKey] = struct{}{}
		}
	}

	allTokens, err := r.tokenManager.GetTokensByKeys(maps.Keys(tokensOfInterest))
	if err != nil {
		return nil, err
	}

	if storageBalances, storageErr := r.tokenBalancesStorage.GetBalances(context.Background(), allTokens, addresses); storageErr != nil {
		logutils.ZapLogger().Error("failed to get live storage balances", zap.Error(storageErr))
	} else {
		balances = mergeFetchedBalances(balances, storageBalances)
	}

	return r.balancesToTokensByAddress(addresses, allTokens, balances, cachedTokens), nil
}

func mergeFetchedBalances(cached, fetched map[uint64]map[common.Address]map[common.Address]*big.Int) map[uint64]map[common.Address]map[common.Address]*big.Int {
	if fetched == nil {
		return cached
	}
	if cached == nil {
		return fetched
	}
	for chainID, accounts := range fetched {
		if cached[chainID] == nil {
			cached[chainID] = make(map[common.Address]map[common.Address]*big.Int)
		}
		for account, tokens := range accounts {
			if cached[chainID][account] == nil {
				cached[chainID][account] = make(map[common.Address]*big.Int)
			}
			for tokenAddress, balance := range tokens {
				cached[chainID][account][tokenAddress] = balance
			}
		}
	}
	return cached
}
