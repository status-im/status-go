package wallet

import (
	"context"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// WalletTickReload emitted every 15mn to reload the wallet balance and history
const EventWalletTickReload walletevent.EventType = "wallet-tick-reload"

func NewReader(rpcClient *rpc.Client, tokenManager *token.Manager, accountsDB *accounts.Database, walletFeed *event.Feed) *Reader {
	return &Reader{rpcClient, tokenManager, accountsDB, walletFeed, nil}
}

type Reader struct {
	rpcClient    *rpc.Client
	tokenManager *token.Manager
	accountsDB   *accounts.Database
	walletFeed   *event.Feed
	cancel       context.CancelFunc
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
}

type ChainBalance struct {
	Balance *big.Float     `json:"balance"`
	Address common.Address `json:"address"`
	ChainID uint64         `json:"chainId"`
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
	Peg                     string                       `json:"peg"`
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

	currency, err := r.accountsDB.GetCurrency()
	if err != nil {
		return nil, err
	}

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
		prices            = map[string]float64{}
		tokenDetails      = map[string]Coin{}
		tokenMarketValues = map[string]MarketCoinValues{}
		balances          = map[uint64]map[common.Address]map[common.Address]*hexutil.Big{}
	)

	group.Add(func(parent context.Context) error {
		prices, err = fetchCryptoComparePrices(tokenSymbols, currency)
		if err != nil {
			return err
		}
		return nil
	})

	group.Add(func(parent context.Context) error {
		tokenDetails, err = fetchCryptoCompareTokenDetails(tokenSymbols)
		if err != nil {
			return err
		}
		return nil
	})

	group.Add(func(parent context.Context) error {
		tokenMarketValues, err = fetchTokenMarketValues(tokenSymbols, currency)
		if err != nil {
			return err
		}
		return nil
	})

	group.Add(func(parent context.Context) error {
		clients, err := chain.NewClients(r.rpcClient, chainIDs)
		if err != nil {
			return err
		}

		balances, err = r.tokenManager.GetBalancesByChain(ctx, clients, addresses, tokenAddresses)
		if err != nil {
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
	if err != nil {
		return nil, err
	}
	result := make(map[common.Address][]Token)
	for _, address := range addresses {
		for symbol, tokens := range getTokenBySymbols(allTokens) {
			balancesPerChain := make(map[uint64]ChainBalance)
			decimals := tokens[0].Decimals
			for _, token := range tokens {
				hexBalance := balances[token.ChainID][address][token.Address]
				balance := big.NewFloat(0.0)
				if hexBalance != nil {
					balance = new(big.Float).Quo(
						new(big.Float).SetInt(hexBalance.ToInt()),
						big.NewFloat(math.Pow(10, float64(decimals))),
					)
				}
				balancesPerChain[token.ChainID] = ChainBalance{
					Balance: balance,
					Address: token.Address,
					ChainID: token.ChainID,
				}
			}

			marketValuesPerCurrency := make(map[string]TokenMarketValues)
			marketValuesPerCurrency[currency] = TokenMarketValues{
				MarketCap:       tokenMarketValues[symbol].MKTCAP,
				HighDay:         tokenMarketValues[symbol].HIGHDAY,
				LowDay:          tokenMarketValues[symbol].LOWDAY,
				ChangePctHour:   tokenMarketValues[symbol].CHANGEPCTHOUR,
				ChangePctDay:    tokenMarketValues[symbol].CHANGEPCTDAY,
				ChangePct24hour: tokenMarketValues[symbol].CHANGEPCT24HOUR,
				Change24hour:    tokenMarketValues[symbol].CHANGE24HOUR,
				Price:           prices[symbol],
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
				Peg:                     tokens[0].Peg,
			}

			result[address] = append(result[address], walletToken)
		}
	}
	return result, nil
}
