package wallet

import (
	"context"
	"math"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// WalletTickReload emitted every 15mn to reload the wallet balance and history
const EventWalletTickReload walletevent.EventType = "wallet-tick-reload"
const EventWalletTickCheckConnected walletevent.EventType = "wallet-tick-check-connected"

const (
	walletTickReloadPeriod      = 10 * time.Minute
	activityReloadDelay         = 30 // Wait this many seconds after activity is detected before triggering a wallet reload
	activityReloadMarginSeconds = 30 // Trigger a wallet reload if activity is detected this many seconds before the last reload
)

func belongsToMandatoryTokens(symbol string) bool {
	var mandatoryTokens = []string{"ETH", "DAI", "SNT", "STT"}
	for _, t := range mandatoryTokens {
		if t == symbol {
			return true
		}
	}
	return false
}

func NewReader(tokenManager token.ManagerInterface, marketManager *market.Manager, persistence token.TokenBalancesStorage, walletFeed *event.Feed) *Reader {
	return &Reader{
		tokenManager:        tokenManager,
		marketManager:       marketManager,
		persistence:         persistence,
		walletFeed:          walletFeed,
		refreshBalanceCache: true,
	}
}

type Reader struct {
	tokenManager                   token.ManagerInterface
	marketManager                  *market.Manager
	persistence                    token.TokenBalancesStorage
	walletFeed                     *event.Feed
	cancel                         context.CancelFunc
	walletEventsWatcher            *walletevent.Watcher
	lastWalletTokenUpdateTimestamp sync.Map
	reloadDelayTimer               *time.Timer
	refreshBalanceCache            bool
	rw                             sync.RWMutex
}

func splitVerifiedTokens(tokens []*token.Token) ([]*token.Token, []*token.Token) {
	verified := make([]*token.Token, 0)
	unverified := make([]*token.Token, 0)

	for _, t := range tokens {
		if t.Verified {
			verified = append(verified, t)
		} else {
			unverified = append(unverified, t)
		}
	}

	return verified, unverified
}

func getTokenBySymbols(tokens []*token.Token) map[string][]*token.Token {
	res := make(map[string][]*token.Token)

	for _, t := range tokens {
		if _, ok := res[t.Symbol]; !ok {
			res[t.Symbol] = make([]*token.Token, 0)
		}

		res[t.Symbol] = append(res[t.Symbol], t)
	}

	return res
}

func getTokenAddresses(tokens []*token.Token) []common.Address {
	set := make(map[common.Address]bool)
	for _, token := range tokens {
		set[token.Address] = true
	}
	res := make([]common.Address, 0)
	for address := range set {
		res = append(res, address)
	}
	return res
}

func (r *Reader) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	r.startWalletEventsWatcher()

	go func() {
		defer gocommon.LogOnPanic()
		ticker := time.NewTicker(walletTickReloadPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.triggerWalletReload()
			}
		}
	}()
	return nil
}

func (r *Reader) Stop() {
	if r.cancel != nil {
		r.cancel()
	}

	r.stopWalletEventsWatcher()

	r.cancelDelayedWalletReload()

	r.lastWalletTokenUpdateTimestamp = sync.Map{}
}

func (r *Reader) Restart() error {
	r.Stop()
	return r.Start()
}

func (r *Reader) triggerWalletReload() {
	r.cancelDelayedWalletReload()

	r.walletFeed.Send(walletevent.Event{
		Type: EventWalletTickReload,
	})
}

func (r *Reader) triggerDelayedWalletReload() {
	r.cancelDelayedWalletReload()

	r.reloadDelayTimer = time.AfterFunc(time.Duration(activityReloadDelay)*time.Second, r.triggerWalletReload)
}

func (r *Reader) cancelDelayedWalletReload() {

	if r.reloadDelayTimer != nil {
		r.reloadDelayTimer.Stop()
		r.reloadDelayTimer = nil
	}
}

func (r *Reader) startWalletEventsWatcher() {
	if r.walletEventsWatcher != nil {
		return
	}

	// Respond to ETH/Token transfers
	walletEventCb := func(event walletevent.Event) {
		if event.Type != transfer.EventInternalETHTransferDetected &&
			event.Type != transfer.EventInternalERC20TransferDetected {
			return
		}

		for _, address := range event.Accounts {
			timestamp, ok := r.lastWalletTokenUpdateTimestamp.Load(address)
			timecheck := int64(0)
			if ok {
				timecheck = timestamp.(int64) - activityReloadMarginSeconds
			}

			if !ok || event.At > timecheck {
				r.triggerDelayedWalletReload()
				r.invalidateBalanceCache()
				break
			}
		}
	}

	r.walletEventsWatcher = walletevent.NewWatcher(r.walletFeed, walletEventCb)

	r.walletEventsWatcher.Start()
}

func (r *Reader) stopWalletEventsWatcher() {
	if r.walletEventsWatcher != nil {
		r.walletEventsWatcher.Stop()
		r.walletEventsWatcher = nil
	}
}

func (r *Reader) tokensCachedForAddresses(addresses []common.Address) bool {
	cachedTokens, err := r.getCachedWalletTokensWithoutMarketData()
	if err != nil {
		return false
	}

	for _, address := range addresses {
		_, ok := cachedTokens[address]
		if !ok {
			return false
		}
	}

	return true
}

func (r *Reader) isCacheTimestampValidForAddress(address common.Address) bool {
	_, ok := r.lastWalletTokenUpdateTimestamp.Load(address)
	return ok
}

func (r *Reader) areCacheTimestampsValid(addresses []common.Address) bool {
	for _, address := range addresses {
		if !r.isCacheTimestampValidForAddress(address) {
			return false
		}
	}

	return true
}

func (r *Reader) isBalanceCacheValid(addresses []common.Address) bool {
	r.rw.RLock()
	defer r.rw.RUnlock()

	return !r.refreshBalanceCache && r.tokensCachedForAddresses(addresses) && r.areCacheTimestampsValid(addresses)
}

func (r *Reader) balanceRefreshed() {
	r.rw.Lock()
	defer r.rw.Unlock()

	r.refreshBalanceCache = false
}

func (r *Reader) invalidateBalanceCache() {
	r.rw.Lock()
	defer r.rw.Unlock()

	r.refreshBalanceCache = true
}

func (r *Reader) FetchOrGetCachedWalletBalances(ctx context.Context, clients map[uint64]chain.ClientInterface, addresses []common.Address, forceRefresh bool) (map[common.Address][]token.StorageToken, error) {
	needFetch := forceRefresh || !r.isBalanceCacheValid(addresses) || r.isBalanceUpdateNeededAnyway(clients, addresses)

	if needFetch {
		_, err := r.FetchBalances(ctx, clients, addresses)
		if err != nil {
			logutils.ZapLogger().Error("FetchOrGetCachedWalletBalances error", zap.Error(err))
		}
	}

	return r.GetCachedBalances(clients, addresses)
}

func (r *Reader) isBalanceUpdateNeededAnyway(clients map[uint64]chain.ClientInterface, addresses []common.Address) bool {
	cachedTokens, err := r.getCachedWalletTokensWithoutMarketData()
	if err != nil {
		return true
	}

	chainIDs := maps.Keys(clients)
	updateAnyway := false
	for _, address := range addresses {
		if res, ok := cachedTokens[address]; !ok || len(res) == 0 {
			updateAnyway = true
			break
		}

		networkFound := map[uint64]bool{}
		for _, token := range cachedTokens[address] {
			for _, chain := range chainIDs {
				if _, ok := token.BalancesPerChain[chain]; ok {
					networkFound[chain] = true
				}
			}
		}

		for _, chain := range chainIDs {
			if !networkFound[chain] {
				updateAnyway = true
				return updateAnyway
			}
		}
	}

	return updateAnyway
}

func tokensToBalancesPerChain(cachedTokens map[common.Address][]token.StorageToken) map[uint64]map[common.Address]map[common.Address]*hexutil.Big {
	cachedBalancesPerChain := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{}
	for address, tokens := range cachedTokens {
		for _, token := range tokens {
			for _, balance := range token.BalancesPerChain {
				if _, ok := cachedBalancesPerChain[balance.ChainID]; !ok {
					cachedBalancesPerChain[balance.ChainID] = map[common.Address]map[common.Address]*hexutil.Big{}
				}
				if _, ok := cachedBalancesPerChain[balance.ChainID][address]; !ok {
					cachedBalancesPerChain[balance.ChainID][address] = map[common.Address]*hexutil.Big{}
				}

				bigBalance, _ := new(big.Int).SetString(balance.RawBalance, 10)
				cachedBalancesPerChain[balance.ChainID][address][balance.Address] = (*hexutil.Big)(bigBalance)
			}
		}
	}

	return cachedBalancesPerChain
}

func (r *Reader) fetchBalances(ctx context.Context, clients map[uint64]chain.ClientInterface, addresses []common.Address, tokenAddresses []common.Address) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	latestBalances, err := r.tokenManager.GetBalancesByChain(ctx, clients, addresses, tokenAddresses)
	if err != nil {
		logutils.ZapLogger().Error("tokenManager.GetBalancesByChain error", zap.Error(err))
		return nil, err
	}

	return latestBalances, nil
}

func toChainBalance(
	balances map[uint64]map[common.Address]map[common.Address]*hexutil.Big,
	tok *token.Token,
	address common.Address,
	decimals uint,
	cachedTokens map[common.Address][]token.StorageToken,
	hasError bool,
	isMandatoryToken bool,
) *token.ChainBalance {
	hexBalance := &big.Int{}
	if balances != nil {
		hexBalance = balances[tok.ChainID][address][tok.Address].ToInt()
	}

	balance := big.NewFloat(0.0)
	if hexBalance != nil {
		balance = new(big.Float).Quo(
			new(big.Float).SetInt(hexBalance),
			big.NewFloat(math.Pow(10, float64(decimals))),
		)
	}

	isVisible := balance.Cmp(big.NewFloat(0.0)) > 0 || isCachedToken(cachedTokens, address, tok.Symbol, tok.ChainID)
	if !isVisible && !isMandatoryToken {
		return nil
	}

	return &token.ChainBalance{
		RawBalance:     hexBalance.String(),
		Balance:        balance,
		Balance1DayAgo: "0",
		Address:        tok.Address,
		ChainID:        tok.ChainID,
		HasError:       hasError,
	}
}

func (r *Reader) getBalance1DayAgo(balance *token.ChainBalance, dayAgoTimestamp int64, symbol string, address common.Address) (*big.Int, error) {
	balance1DayAgo, err := r.tokenManager.GetTokenHistoricalBalance(address, balance.ChainID, symbol, dayAgoTimestamp)
	if err != nil {
		logutils.ZapLogger().Error("tokenManager.GetTokenHistoricalBalance error", zap.Error(err))
		return nil, err
	}

	return balance1DayAgo, nil
}

func (r *Reader) balancesToTokensByAddress(connectedPerChain map[uint64]bool, addresses []common.Address, allTokens []*token.Token, balances map[uint64]map[common.Address]map[common.Address]*hexutil.Big, cachedTokens map[common.Address][]token.StorageToken) map[common.Address][]token.StorageToken {
	verifiedTokens, unverifiedTokens := splitVerifiedTokens(allTokens)

	result := make(map[common.Address][]token.StorageToken)
	dayAgoTimestamp := time.Now().Add(-24 * time.Hour).Unix()

	for _, address := range addresses {
		for _, tokenList := range [][]*token.Token{verifiedTokens, unverifiedTokens} {
			for symbol, tokens := range getTokenBySymbols(tokenList) {
				balancesPerChain := r.createBalancePerChainPerSymbol(address, balances, tokens, cachedTokens, connectedPerChain, dayAgoTimestamp)
				if balancesPerChain == nil {
					continue
				}

				walletToken := token.StorageToken{
					Token: token.Token{
						Name:          tokens[0].Name,
						Symbol:        symbol,
						Decimals:      tokens[0].Decimals,
						PegSymbol:     token.GetTokenPegSymbol(symbol),
						Verified:      tokens[0].Verified,
						CommunityData: tokens[0].CommunityData,
						Image:         tokens[0].Image,
					},
					BalancesPerChain: balancesPerChain,
				}

				result[address] = append(result[address], walletToken)
			}
		}
	}

	return result
}

func (r *Reader) createBalancePerChainPerSymbol(
	address common.Address,
	balances map[uint64]map[common.Address]map[common.Address]*hexutil.Big,
	tokens []*token.Token,
	cachedTokens map[common.Address][]token.StorageToken,
	clientConnectionPerChain map[uint64]bool,
	dayAgoTimestamp int64,
) map[uint64]token.ChainBalance {
	var balancesPerChain map[uint64]token.ChainBalance
	decimals := tokens[0].Decimals
	isMandatoryToken := belongsToMandatoryTokens(tokens[0].Symbol) // we expect all tokens in the list to have the same symbol
	for _, tok := range tokens {
		hasError := false
		if connected, ok := clientConnectionPerChain[tok.ChainID]; ok {
			hasError = !connected
		}

		if _, ok := balances[tok.ChainID][address][tok.Address]; !ok {
			hasError = true
		}

		// TODO: Avoid passing the entire balances map to toChainBalance. Iterate over the balances map once and pass the balance per address per token to toChainBalance
		balance := toChainBalance(balances, tok, address, decimals, cachedTokens, hasError, isMandatoryToken)
		if balance != nil {
			balance1DayAgo, _ := r.getBalance1DayAgo(balance, dayAgoTimestamp, tok.Symbol, address) // Ignore error
			if balance1DayAgo != nil {
				balance.Balance1DayAgo = balance1DayAgo.String()
			}

			if balancesPerChain == nil {
				balancesPerChain = make(map[uint64]token.ChainBalance)
			}
			balancesPerChain[tok.ChainID] = *balance
		}
	}

	return balancesPerChain
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

func isCachedToken(cachedTokens map[common.Address][]token.StorageToken, address common.Address, symbol string, chainID uint64) bool {
	if tokens, ok := cachedTokens[address]; ok {
		for _, t := range tokens {
			if t.Symbol != symbol {
				continue
			}
			_, ok := t.BalancesPerChain[chainID]
			if ok {
				return true
			}
		}
	}
	return false
}

// getCachedWalletTokensWithoutMarketData returns the latest fetched balances, minus
// price information
func (r *Reader) getCachedWalletTokensWithoutMarketData() (map[common.Address][]token.StorageToken, error) {
	return r.persistence.GetTokens()
}

func (r *Reader) updateTokenUpdateTimestamp(addresses []common.Address) {
	for _, address := range addresses {
		r.lastWalletTokenUpdateTimestamp.Store(address, time.Now().Unix())
	}
}

func (r *Reader) FetchBalances(ctx context.Context, clients map[uint64]chain.ClientInterface, addresses []common.Address) (map[common.Address][]token.StorageToken, error) {
	cachedTokens, err := r.getCachedWalletTokensWithoutMarketData()
	if err != nil {
		return nil, err
	}

	chainIDs := maps.Keys(clients)
	allTokens, err := r.tokenManager.GetTokensByChainIDs(chainIDs)
	if err != nil {
		return nil, err
	}

	tokenAddresses := getTokenAddresses(allTokens)
	balances, err := r.fetchBalances(ctx, clients, addresses, tokenAddresses)
	if err != nil {
		logutils.ZapLogger().Error("failed to update balances", zap.Error(err))
		return nil, err
	}

	connectedPerChain := map[uint64]bool{}
	for chainID, client := range clients {
		connectedPerChain[chainID] = client.IsConnected()
	}

	tokens := r.balancesToTokensByAddress(connectedPerChain, addresses, allTokens, balances, cachedTokens)

	err = r.persistence.SaveTokens(tokens)
	if err != nil {
		logutils.ZapLogger().Error("failed to save tokens", zap.Error(err)) // Do not return error, as it is not critical
	}

	r.updateTokenUpdateTimestamp(addresses)
	r.balanceRefreshed()

	return tokens, err
}

func (r *Reader) GetCachedBalances(clients map[uint64]chain.ClientInterface, addresses []common.Address) (map[common.Address][]token.StorageToken, error) {
	cachedTokens, err := r.getCachedWalletTokensWithoutMarketData()
	if err != nil {
		return nil, err
	}

	chainIDs := maps.Keys(clients)
	allTokens, err := r.tokenManager.GetTokensByChainIDs(chainIDs)
	if err != nil {
		return nil, err
	}

	connectedPerChain := map[uint64]bool{}
	for chainID, client := range clients {
		connectedPerChain[chainID] = client.IsConnected()
	}

	balances := tokensToBalancesPerChain(cachedTokens)
	return r.balancesToTokensByAddress(connectedPerChain, addresses, allTokens, balances, cachedTokens), nil
}
