package wallet

import (
	"context"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// WalletTickReload emitted every 15mn to reload the wallet balance and history
const EventWalletTickReload walletevent.EventType = "wallet-tick-reload"
const EventWalletTickCheckConnected walletevent.EventType = "wallet-tick-check-connected"

func getFixedCurrencies() []string {
	return []string{"USD"}
}

func belongsToMandatoryTokens(symbol string) bool {
	var mandatoryTokens = []string{"ETH", "DAI", "SNT", "STT"}
	for _, t := range mandatoryTokens {
		if t == symbol {
			return true
		}
	}
	return false
}

func NewReader(rpcClient *rpc.Client, tokenManager *token.Manager, marketManager *market.Manager, accountsDB *accounts.Database, persistence *Persistence, walletFeed *event.Feed) *Reader {
	return &Reader{
		rpcClient,
		tokenManager,
		marketManager,
		accountsDB,
		persistence,
		walletFeed,
		nil}
}

type Reader struct {
	rpcClient     *rpc.Client
	tokenManager  *token.Manager
	marketManager *market.Manager
	accountsDB    *accounts.Database
	persistence   *Persistence
	walletFeed    *event.Feed
	cancel        context.CancelFunc
}

type TokenMarketValues struct {
	MarketCap       float64 `json:"marketCap"`
	HighDay         float64 `json:"highDay"`
	LowDay          float64 `json:"lowDay"`
	ChangePctHour   float64 `json:"changePctHour"`
	ChangePctDay    float64 `json:"changePctDay"`
	ChangePct24hour float64 `json:"changePct24hour"`
	Change24hour    float64 `json:"change24hour"`
	Price           float64 `json:"price"`
	HasError        bool    `json:"hasError"`
}

type ChainBalance struct {
	RawBalance string         `json:"rawBalance"`
	Balance    *big.Float     `json:"balance"`
	Address    common.Address `json:"address"`
	ChainID    uint64         `json:"chainId"`
	HasError   bool           `json:"hasError"`
}

type Token struct {
	Name                    string                       `json:"name"`
	Symbol                  string                       `json:"symbol"`
	Decimals                uint                         `json:"decimals"`
	BalancesPerChain        map[uint64]ChainBalance      `json:"balancesPerChain"`
	Description             string                       `json:"description"`
	AssetWebsiteURL         string                       `json:"assetWebsiteUrl"`
	BuiltOn                 string                       `json:"builtOn"`
	MarketValuesPerCurrency map[string]TokenMarketValues `json:"marketValuesPerCurrency"`
	PegSymbol               string                       `json:"pegSymbol"`
	Verified                bool                         `json:"verified"`
	CommunitID              string                       `json:"communityId"`
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

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.walletFeed.Send(walletevent.Event{
					Type: EventWalletTickReload,
				})
			}
		}
	}()
	return nil
}

func (r *Reader) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Reader) GetWalletToken(ctx context.Context, addresses []common.Address) (map[common.Address][]Token, error) {
	areTestNetworksEnabled, err := r.accountsDB.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	networks, err := r.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}
	availableNetworks := make([]*params.Network, 0)
	for _, network := range networks {
		if network.IsTest != areTestNetworksEnabled {
			continue
		}
		availableNetworks = append(availableNetworks, network)
	}

	chainIDs := make([]uint64, 0)
	for _, network := range availableNetworks {
		chainIDs = append(chainIDs, network.ChainID)
	}

	currencies := make([]string, 0)
	currency, err := r.accountsDB.GetCurrency()
	if err != nil {
		return nil, err
	}
	currencies = append(currencies, currency)
	currencies = append(currencies, getFixedCurrencies()...)
	allTokens, err := r.tokenManager.GetTokensByChainIDs(chainIDs)

	if err != nil {
		return nil, err
	}
	for _, network := range availableNetworks {
		allTokens = append(allTokens, r.tokenManager.ToToken(network))
	}

	tokenAddresses := getTokenAddresses(allTokens)

	clients, err := r.rpcClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	balances, err := r.tokenManager.GetBalancesByChain(ctx, clients, addresses, tokenAddresses)
	if err != nil {
		for _, client := range clients {
			client.SetIsConnected(false)
		}
		log.Info("tokenManager.GetBalancesByChain error", "err", err)
		return nil, err
	}

	verifiedTokens, unverifiedTokens := splitVerifiedTokens(allTokens)
	tokenSymbols := make([]string, 0)
	result := make(map[common.Address][]Token)

	for _, address := range addresses {
		for _, tokenList := range [][]*token.Token{verifiedTokens, unverifiedTokens} {
			for symbol, tokens := range getTokenBySymbols(tokenList) {
				balancesPerChain := make(map[uint64]ChainBalance)
				decimals := tokens[0].Decimals
				anyPositiveBalance := false
				for _, token := range tokens {
					hexBalance := balances[token.ChainID][address][token.Address]
					balance := big.NewFloat(0.0)
					if hexBalance != nil {
						balance = new(big.Float).Quo(
							new(big.Float).SetInt(hexBalance.ToInt()),
							big.NewFloat(math.Pow(10, float64(decimals))),
						)
					}
					hasError := false
					if client, ok := clients[token.ChainID]; ok {
						hasError = err != nil || !client.GetIsConnected()
					}
					if !anyPositiveBalance {
						anyPositiveBalance = balance.Cmp(big.NewFloat(0.0)) > 0
					}
					balancesPerChain[token.ChainID] = ChainBalance{
						RawBalance: hexBalance.ToInt().String(),
						Balance:    balance,
						Address:    token.Address,
						ChainID:    token.ChainID,
						HasError:   hasError,
					}
				}

				if !anyPositiveBalance && !belongsToMandatoryTokens(symbol) {
					continue
				}

				var communityID string
				if tokens[0].CommunityID != nil {
					communityID = *tokens[0].CommunityID
				}

				walletToken := Token{
					Name:             tokens[0].Name,
					Symbol:           symbol,
					BalancesPerChain: balancesPerChain,
					Decimals:         decimals,
					PegSymbol:        token.GetTokenPegSymbol(symbol),
					Verified:         tokens[0].Verified,
					CommunitID:       communityID,
				}

				tokenSymbols = append(tokenSymbols, symbol)
				result[address] = append(result[address], walletToken)
			}
		}
	}

	var (
		group             = async.NewAtomicGroup(ctx)
		prices            = map[string]map[string]float64{}
		tokenDetails      = map[string]thirdparty.TokenDetails{}
		tokenMarketValues = map[string]thirdparty.TokenMarketValues{}
	)

	group.Add(func(parent context.Context) error {
		prices, err = r.marketManager.FetchPrices(tokenSymbols, currencies)
		if err != nil {
			log.Info("marketManager.FetchPrices err", err)
		}
		return nil
	})

	group.Add(func(parent context.Context) error {
		tokenDetails, err = r.marketManager.FetchTokenDetails(tokenSymbols)
		if err != nil {
			log.Info("marketManager.FetchTokenDetails err", err)
		}
		return nil
	})

	group.Add(func(parent context.Context) error {
		tokenMarketValues, err = r.marketManager.FetchTokenMarketValues(tokenSymbols, currency)
		if err != nil {
			log.Info("marketManager.FetchTokenMarketValues err", err)
		}
		return nil
	})

	select {
	case <-group.WaitAsync():
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	err = group.Error()
	if err != nil {
		return nil, err
	}

	for address, tokens := range result {
		for index, token := range tokens {
			marketValuesPerCurrency := make(map[string]TokenMarketValues)
			for _, currency := range currencies {
				if _, ok := tokenMarketValues[token.Symbol]; !ok {
					continue
				}
				marketValuesPerCurrency[currency] = TokenMarketValues{
					MarketCap:       tokenMarketValues[token.Symbol].MKTCAP,
					HighDay:         tokenMarketValues[token.Symbol].HIGHDAY,
					LowDay:          tokenMarketValues[token.Symbol].LOWDAY,
					ChangePctHour:   tokenMarketValues[token.Symbol].CHANGEPCTHOUR,
					ChangePctDay:    tokenMarketValues[token.Symbol].CHANGEPCTDAY,
					ChangePct24hour: tokenMarketValues[token.Symbol].CHANGEPCT24HOUR,
					Change24hour:    tokenMarketValues[token.Symbol].CHANGE24HOUR,
					Price:           prices[token.Symbol][currency],
					HasError:        !r.marketManager.IsConnected,
				}
			}

			if _, ok := tokenDetails[token.Symbol]; !ok {
				continue
			}

			result[address][index].Description = tokenDetails[token.Symbol].Description
			result[address][index].AssetWebsiteURL = tokenDetails[token.Symbol].AssetWebsiteURL
			result[address][index].BuiltOn = tokenDetails[token.Symbol].BuiltOn
			result[address][index].MarketValuesPerCurrency = marketValuesPerCurrency
		}
	}

	return result, r.persistence.SaveTokens(result)
}

// GetCachedWalletTokensWithoutMarketData returns the latest fetched balances, minus
// price information
func (r *Reader) GetCachedWalletTokensWithoutMarketData() (map[common.Address][]Token, error) {
	return r.persistence.GetTokens()
}
