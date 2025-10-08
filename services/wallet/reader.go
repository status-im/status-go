package wallet

//go:generate go tool mockgen -package=mock_reader -source=reader.go -destination=mock/reader/reader.go

import (
	"context"
	"math"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/bep/debounce"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc20"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
	"github.com/status-im/status-go/services/wallet/transferdetector"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// WalletTickReload emitted every 15mn to reload the wallet balance and history
const EventWalletTickReload walletevent.EventType = "wallet-tick-reload"
const reloadDebounceTime = 5 * time.Second

type AccountAddress = tokenbalances.AccountAddress
type ContractAddress = tokenbalances.ContractAddress

func belongsToMandatoryTokens(symbol string) bool {
	var mandatoryTokens = []string{"ETH", "DAI", "SNT", "STT", "USDC", "BNB"}
	for _, t := range mandatoryTokens {
		if t == symbol {
			return true
		}
	}
	return false
}

type ReaderInterface interface {
	Start() error
	Stop()
	GetCachedBalances(chainIDs []uint64, addresses []common.Address) (map[common.Address][]tokenTypes.StorageToken, error)
	GetLastTokenUpdateTimestamps() map[common.Address]int64
}

func NewReader(
	tokenManager token.ManagerInterface,
	marketManager *market.Manager,
	persistence token.TokenBalancesStorage,
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
		persistence:                   persistence,
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
	persistence                    token.TokenBalancesStorage
	walletFeed                     *event.Feed
	lastWalletTokenUpdateTimestamp sync.Map
	reloadDebounceFn               func(f func())

	stopCh chan struct{}
}

func splitVerifiedTokens(tokens []*tokenTypes.Token) ([]*tokenTypes.Token, []*tokenTypes.Token) {
	verified := make([]*tokenTypes.Token, 0)
	unverified := make([]*tokenTypes.Token, 0)

	for _, t := range tokens {
		if t.Verified {
			verified = append(verified, t)
		} else {
			unverified = append(unverified, t)
		}
	}

	return verified, unverified
}

func getTokenBySymbols(tokens []*tokenTypes.Token) map[string][]*tokenTypes.Token {
	res := make(map[string][]*tokenTypes.Token)

	for _, t := range tokens {
		if _, ok := res[t.Symbol]; !ok {
			res[t.Symbol] = make([]*tokenTypes.Token, 0)
		}

		res[t.Symbol] = append(res[t.Symbol], t)
	}

	return res
}

func getTokenAddresses(tokens []*tokenTypes.Token) []common.Address {
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
					if !event.BalanceChanged {
						continue
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
						r.processERC20TransferEvent(msg.ChainID, unpackedEvent)
					}
				}
			}
		}
	}()
}

func (r *Reader) processERC20TransferEvent(chainID uint64, event erc20.Erc20Transfer) {
	// Find token in db or if this is a community token, find its metadata
	token := r.tokenManager.FindOrCreateTokenByAddress(context.TODO(), chainID, event.Raw.Address)
	if token != nil {
		isFirst := false
		if token.Verified || token.CommunityData != nil {
			isFirst, _ = r.tokenManager.MarkAsPreviouslyOwnedToken(token, event.To)
		}
		if token.CommunityData != nil {
			go func() {
				defer gocommon.LogOnPanic()
				r.tokenManager.SignalCommunityTokenReceived(event.To, event.Raw.TxHash, event.Value, token, isFirst)
			}()
		}
	}
}

func tokensToBalancesPerChain(cachedTokens map[common.Address][]tokenTypes.StorageToken) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
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

				bigBalance, ok := new(big.Int).SetString(balance.RawBalance, 10)
				if !ok {
					return nil, gocommon.ErrBigIntSetFromString(balance.RawBalance)
				}
				cachedBalancesPerChain[balance.ChainID][address][balance.Address] = (*hexutil.Big)(bigBalance)
			}
		}
	}

	return cachedBalancesPerChain, nil
}

func (r *Reader) fetchBalances(ctx context.Context, chainIDs []uint64, addresses []AccountAddress, tokenAddresses []ContractAddress) (map[uint64]map[AccountAddress]map[ContractAddress]*hexutil.Big, error) {
	balances := make(map[uint64]map[AccountAddress]map[ContractAddress]*hexutil.Big)
	for _, chainID := range chainIDs {
		balances[chainID] = make(map[AccountAddress]map[ContractAddress]*hexutil.Big)
		chainBalances, err := r.tokenBalancesStorage.GetBalances(ctx, chainID, tokenAddresses, addresses)
		if err != nil {
			logutils.ZapLogger().Error("tokenBalancesStorage.GetBalances error", zap.Error(err))
			return nil, err
		}
		for account, tokenBalances := range chainBalances {
			balances[chainID][account] = make(map[ContractAddress]*hexutil.Big)
			for token, balance := range tokenBalances {
				balances[chainID][account][token] = (*hexutil.Big)(balance)
			}
		}
	}

	return balances, nil
}

func toChainBalance(
	balances map[uint64]map[common.Address]map[common.Address]*hexutil.Big,
	tok *tokenTypes.Token,
	address common.Address,
	decimals uint,
	cachedTokens map[common.Address][]tokenTypes.StorageToken,
	hasError bool,
	isMandatoryToken bool,
) *tokenTypes.ChainBalance {
	hexBalance := &big.Int{}
	if balances != nil {
		tokenBalance := balances[tok.ChainID][address][tok.Address]
		// Balances should be represented by a uint256 (32 bytes). Some spam tokens return
		// fake values larger than that, so we ignore them.
		if tokenBalance != nil && len(tokenBalance.ToInt().Bytes()) <= 32 {
			hexBalance = tokenBalance.ToInt()
		}
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

	return &tokenTypes.ChainBalance{
		RawBalance: hexBalance.String(),
		Balance:    balance,
		Address:    tok.Address,
		ChainID:    tok.ChainID,
		HasError:   hasError,
	}
}

func (r *Reader) balancesToTokensByAddress(addresses []common.Address, allTokens []*tokenTypes.Token, balances map[uint64]map[AccountAddress]map[ContractAddress]*hexutil.Big, cachedTokens map[common.Address][]tokenTypes.StorageToken) map[common.Address][]tokenTypes.StorageToken {
	verifiedTokens, unverifiedTokens := splitVerifiedTokens(allTokens)

	result := make(map[common.Address][]tokenTypes.StorageToken)
	dayAgoTimestamp := time.Now().Add(-24 * time.Hour).Unix()

	for _, address := range addresses {
		for _, tokenList := range [][]*tokenTypes.Token{verifiedTokens, unverifiedTokens} {
			for symbol, tokens := range getTokenBySymbols(tokenList) {
				balancesPerChain := r.createBalancePerChainPerSymbol(address, balances, tokens, cachedTokens, dayAgoTimestamp)
				if balancesPerChain == nil {
					continue
				}

				walletToken := tokenTypes.StorageToken{
					Token: tokenTypes.Token{
						Name:          tokens[0].Name,
						Symbol:        symbol,
						Decimals:      tokens[0].Decimals,
						PegSymbol:     tokenTypes.GetTokenPegSymbol(symbol),
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
	balances map[uint64]map[AccountAddress]map[ContractAddress]*hexutil.Big,
	tokens []*tokenTypes.Token,
	cachedTokens map[common.Address][]tokenTypes.StorageToken,
	dayAgoTimestamp int64,
) map[uint64]tokenTypes.ChainBalance {
	var balancesPerChain map[uint64]tokenTypes.ChainBalance
	decimals := tokens[0].Decimals
	isMandatoryToken := belongsToMandatoryTokens(tokens[0].Symbol) // we expect all tokens in the list to have the same symbol
	for _, tok := range tokens {
		hasError := false

		if _, ok := balances[tok.ChainID][address][tok.Address]; !ok {
			hasError = true
		}

		// TODO: Avoid passing the entire balances map to toChainBalance. Iterate over the balances map once and pass the balance per address per token to toChainBalance
		balance := toChainBalance(balances, tok, address, decimals, cachedTokens, hasError, isMandatoryToken)
		if balance != nil {
			if balancesPerChain == nil {
				balancesPerChain = make(map[uint64]tokenTypes.ChainBalance)
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

func isCachedToken(cachedTokens map[common.Address][]tokenTypes.StorageToken, address common.Address, symbol string, chainID uint64) bool {
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
func (r *Reader) getCachedWalletTokensWithoutMarketData() (map[common.Address][]tokenTypes.StorageToken, error) {
	return r.persistence.GetTokens()
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

	allTokens, err := r.tokenManager.GetTokensByChainIDs(chainIDs)
	if err != nil {
		logutils.ZapLogger().Error("failed to get tokens list", zap.Error(err))
		return
	}

	tokenAddresses := getTokenAddresses(allTokens)
	balances, err := r.fetchBalances(ctx, chainIDs, addresses, tokenAddresses)
	if err != nil {
		logutils.ZapLogger().Error("failed to update balances", zap.Error(err))
		return
	}

	tokens := r.balancesToTokensByAddress(addresses, allTokens, balances, cachedTokens)

	err = r.persistence.SaveTokens(tokens)
	if err != nil {
		logutils.ZapLogger().Error("failed to save tokens", zap.Error(err)) // Do not return error, as it is not critical
	}

	r.updateTokenUpdateTimestamp(addresses)

	r.reloadDebounceFn(r.triggerWalletReload)

	return
}

func (r *Reader) GetCachedBalances(chainIDs []uint64, addresses []common.Address) (map[common.Address][]tokenTypes.StorageToken, error) {
	cachedTokens, err := r.getCachedWalletTokensWithoutMarketData()
	if err != nil {
		return nil, err
	}

	allTokens, err := r.tokenManager.GetTokensByChainIDs(chainIDs)
	if err != nil {
		return nil, err
	}

	balances, err := tokensToBalancesPerChain(cachedTokens)
	if err != nil {
		return nil, err
	}

	return r.balancesToTokensByAddress(addresses, allTokens, balances, cachedTokens), nil
}
