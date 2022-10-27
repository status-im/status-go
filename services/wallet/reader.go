package wallet

import (
	"context"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/services/wallet/token"
)

func NewReader(s *Service) *Reader {
	return &Reader{s}
}

type Reader struct {
	s *Service
}

type ChainBalance struct {
	Balance *big.Float     `json:"balance"`
	Address common.Address `json:"address"`
	ChainID uint64         `json:"chainId"`
}

type Token struct {
	Name             string                  `json:"name"`
	Symbol           string                  `json:"symbol"`
	Color            string                  `json:"color"`
	Decimals         uint                    `json:"decimals"`
	BalancesPerChain map[uint64]ChainBalance `json:"balancesPerChain"`
	Description      string                  `json:"description"`
	AssetWebsiteURL  string                  `json:"assetWebsiteUrl"`
	BuiltOn          string                  `json:"builtOn"`
	MarketCap        string                  `json:"marketCap"`
	HighDay          string                  `json:"highDay"`
	LowDay           string                  `json:"lowDay"`
	ChangePctHour    string                  `json:"changePctHour"`
	ChangePctDay     string                  `json:"changePctDay"`
	ChangePct24hour  string                  `json:"changePct24hour"`
	Change24hour     string                  `json:"change24hour"`
	CurrencyPrice    float64                 `json:"currencyPrice"`
}

func getAddresses(accounts []*accounts.Account) []common.Address {
	addresses := make([]common.Address, len(accounts))
	for _, account := range accounts {
		addresses = append(addresses, common.Address(account.Address))
	}
	return addresses
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

func (r *Reader) Start(ctx context.Context, chainIDs []uint64) error {
	accounts, err := r.s.accountsDB.GetAccounts()
	if err != nil {
		return err
	}

	return r.s.transferController.CheckRecentHistory(chainIDs, getAddresses(accounts))
}

func (r *Reader) GetWalletToken(ctx context.Context) (map[common.Address][]Token, error) {
	networks, err := r.s.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}

	chainIDs := make([]uint64, 0)
	for _, network := range networks {
		chainIDs = append(chainIDs, network.ChainID)
	}

	currency, err := r.s.accountsDB.GetCurrency()
	if err != nil {
		return nil, err
	}

	allTokens, err := r.s.tokenManager.GetAllTokens()
	if err != nil {
		return nil, err
	}
	for _, network := range networks {
		allTokens = append(allTokens, r.s.tokenManager.ToToken(network))
	}

	tokenSymbols := getTokenSymbols(allTokens)
	tokenAddresses := getTokenAddresses(allTokens)

	accounts, err := r.s.accountsDB.GetAccounts()
	if err != nil {
		return nil, err
	}

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
		clients, err := chain.NewClients(r.s.rpcClient, chainIDs)
		if err != nil {
			return err
		}

		balances, err = r.s.tokenManager.GetBalancesByChain(ctx, clients, getAddresses(accounts), tokenAddresses)
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
	for _, account := range accounts {
		commonAddress := common.Address(account.Address)
		for symbol, tokens := range getTokenBySymbols(allTokens) {
			balancesPerChain := make(map[uint64]ChainBalance)
			decimals := tokens[0].Decimals
			for _, token := range tokens {
				hexBalance := balances[token.ChainID][commonAddress][token.Address]
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

			walletToken := Token{
				Name:             tokens[0].Name,
				Color:            tokens[0].Color,
				Symbol:           symbol,
				BalancesPerChain: balancesPerChain,
				Decimals:         decimals,
				Description:      tokenDetails[symbol].Description,
				AssetWebsiteURL:  tokenDetails[symbol].AssetWebsiteURL,
				BuiltOn:          tokenDetails[symbol].BuiltOn,
				MarketCap:        tokenMarketValues[symbol].MKTCAP,
				HighDay:          tokenMarketValues[symbol].HIGHDAY,
				LowDay:           tokenMarketValues[symbol].LOWDAY,
				ChangePctHour:    tokenMarketValues[symbol].CHANGEPCTHOUR,
				ChangePctDay:     tokenMarketValues[symbol].CHANGEPCTDAY,
				ChangePct24hour:  tokenMarketValues[symbol].CHANGEPCT24HOUR,
				Change24hour:     tokenMarketValues[symbol].CHANGE24HOUR,
				CurrencyPrice:    prices[symbol],
			}

			result[commonAddress] = append(result[commonAddress], walletToken)
		}
	}
	return result, nil
}
