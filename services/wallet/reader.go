package wallet

import (
	"context"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/accounts"
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

func NewReader(rpcClient *rpc.Client, tokenManager *token.Manager, marketManager *market.Manager, accountsDB *accounts.Database, walletFeed *event.Feed) *Reader {
	return &Reader{rpcClient, tokenManager, marketManager, accountsDB, walletFeed, nil}
}

type Reader struct {
	rpcClient     *rpc.Client
	tokenManager  *token.Manager
	marketManager *market.Manager
	accountsDB    *accounts.Database
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
	Balance  *big.Float     `json:"balance"`
	Address  common.Address `json:"address"`
	ChainID  uint64         `json:"chainId"`
	HasError bool           `json:"hasError"`
}

type Token struct {
	Name                    string                       `json:"name"`
	Symbol                  string                       `json:"symbol"`
	Color                   string                       `json:"color"`
	Decimals                uint                         `json:"decimals"`
	BalancesPerChain        map[uint64]ChainBalance      `json:"balancesPerChain"`
	Description             string                       `json:"description"`
	AssetWebsiteURL         string                       `json:"assetWebsiteUrl"`
	BuiltOn                 string                       `json:"builtOn"`
	MarketValuesPerCurrency map[string]TokenMarketValues `json:"marketValuesPerCurrency"`
	PegSymbol               string                       `json:"pegSymbol"`
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

func getTokenSymbols(tokens []*token.Token) []string {
	tokensBySymbols := getTokenBySymbols(tokens)
	res := make([]string, 0)
	for symbol := range tokensBySymbols {
		res = append(res, symbol)
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
	networks, err := r.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}

	chainIDs := make([]uint64, 0)
	for _, network := range networks {
		chainIDs = append(chainIDs, network.ChainID)
	}

	currencies := make([]string, 0)
	currency, err := r.accountsDB.GetCurrency()
	if err != nil {
		return nil, err
	}
	currencies = append(currencies, currency)
	currencies = append(currencies, getFixedCurrencies()...)

	allTokens, err := r.tokenManager.GetAllTokens()
	if err != nil {
		return nil, err
	}
	for _, network := range networks {
		allTokens = append(allTokens, r.tokenManager.ToToken(network))
	}

	tokenSymbols := getTokenSymbols(allTokens)
	tokenAddresses := getTokenAddresses(allTokens)

	var (
		group             = async.NewAtomicGroup(ctx)
		prices            = map[string]map[string]float64{}
		tokenDetails      = map[string]thirdparty.TokenDetails{}
		tokenMarketValues = map[string]thirdparty.TokenMarketValues{}
		balances          = map[uint64]map[common.Address]map[common.Address]*hexutil.Big{}
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

	clients, err := r.rpcClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}
	group.Add(func(parent context.Context) error {
		balances, err = r.tokenManager.GetBalancesByChain(ctx, clients, addresses, tokenAddresses)
		if err != nil {
			for _, client := range clients {
				client.SetIsConnected(false)
			}
			log.Info("tokenManager.GetBalancesByChain err", err)
			return err
		}
		return nil
	})

	select {
	case <-group.WaitAsync():
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	err = group.Error()
	result := make(map[common.Address][]Token)
	for _, address := range addresses {
		for symbol, tokens := range getTokenBySymbols(allTokens) {
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
					hasError = err != nil || !client.IsConnected
				}
				if !anyPositiveBalance {
					anyPositiveBalance = balance.Cmp(big.NewFloat(0.0)) > 0
				}
				balancesPerChain[token.ChainID] = ChainBalance{
					Balance:  balance,
					Address:  token.Address,
					ChainID:  token.ChainID,
					HasError: hasError,
				}
			}

			if !anyPositiveBalance && !belongsToMandatoryTokens(symbol) {
				continue
			}

			marketValuesPerCurrency := make(map[string]TokenMarketValues)
			for _, currency := range currencies {
				marketValuesPerCurrency[currency] = TokenMarketValues{
					MarketCap:       tokenMarketValues[symbol].MKTCAP,
					HighDay:         tokenMarketValues[symbol].HIGHDAY,
					LowDay:          tokenMarketValues[symbol].LOWDAY,
					ChangePctHour:   tokenMarketValues[symbol].CHANGEPCTHOUR,
					ChangePctDay:    tokenMarketValues[symbol].CHANGEPCTDAY,
					ChangePct24hour: tokenMarketValues[symbol].CHANGEPCT24HOUR,
					Change24hour:    tokenMarketValues[symbol].CHANGE24HOUR,
					Price:           prices[symbol][currency],
					HasError:        !r.marketManager.IsConnected,
				}
			}

			walletToken := Token{
				Name:                    tokens[0].Name,
				Color:                   tokens[0].Color,
				Symbol:                  symbol,
				BalancesPerChain:        balancesPerChain,
				Decimals:                decimals,
				Description:             tokenDetails[symbol].Description,
				AssetWebsiteURL:         tokenDetails[symbol].AssetWebsiteURL,
				BuiltOn:                 tokenDetails[symbol].BuiltOn,
				MarketValuesPerCurrency: marketValuesPerCurrency,
				PegSymbol:               token.GetTokenPegSymbol(symbol),
			}

			result[address] = append(result[address], walletToken)
		}
	}
	return result, nil
}
