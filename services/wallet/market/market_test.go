package market

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/event"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/rpc/network"
	mock_market "github.com/status-im/status-go/services/wallet/market/mock"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	mock_thirdparty "github.com/status-im/status-go/services/wallet/thirdparty/mock"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func setupTokenManager(t *testing.T) (*token.Manager, func()) {
	appDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	walletDb, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	nm := network.NewManager(appDb, nil, nil, nil)

	return token.NewTokenManager(walletDb, nil, nil, nm, appDb, nil, nil, nil, nil, token.NewPersistence(walletDb)),
		func() {
			require.NoError(t, appDb.Close())
			require.NoError(t, walletDb.Close())
		}
}

func setupMarketManager(t *testing.T, providers []thirdparty.MarketDataProvider, feedEvent *event.Feed) (*Manager, func()) {
	tokenManager, close := setupTokenManager(t)

	tokenManager.Start(context.Background(), 10000, 1000)

	return NewManager(providers, tokenManager, feedEvent), close
}

var mockPrices = map[string]map[string]float64{
	"BTC": {
		"USD": 1.23456,
		"EUR": 2.34567,
		"DAI": 3.45678,
		"ARS": 9.87654,
	},
	"ETH": {
		"USD": 4.56789,
		"EUR": 5.67891,
		"DAI": 6.78912,
		"ARS": 8.76543,
	},
	"SNT": {
		"USD": 7.654,
		"EUR": 6.0,
		"DAI": 1455.12,
		"ARS": 0.0,
	},
}

func TestPrice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	priceProvider := mock_market.NewMockPriceProvider(ctrl)
	priceProvider.SetMockPrices(mockPrices)

	manager, close := setupMarketManager(t, []thirdparty.MarketDataProvider{priceProvider, priceProvider}, &event.Feed{})
	t.Cleanup(close)

	{
		rst := manager.priceCache.Get()
		require.Empty(t, rst)
	}

	{
		symbols := []string{"BTC", "ETH"}
		currencies := []string{"USD", "EUR"}
		rst, err := manager.FetchPrices(symbols, currencies)
		require.NoError(t, err)
		for _, symbol := range symbols {
			for _, currency := range currencies {
				require.Equal(t, rst[symbol][currency], mockPrices[symbol][currency])
			}
		}
	}

	{
		symbols := []string{"BTC", "ETH", "SNT"}
		currencies := []string{"USD", "EUR", "DAI", "ARS"}
		rst, err := manager.FetchPrices(symbols, currencies)
		require.NoError(t, err)
		for _, symbol := range symbols {
			for _, currency := range currencies {
				require.Equal(t, rst[symbol][currency], mockPrices[symbol][currency])
			}
		}
	}

	cache := manager.priceCache.Get()
	for symbol, pricePerCurrency := range mockPrices {
		for currency, price := range pricePerCurrency {
			require.Equal(t, price, cache[symbol][currency].Price)
		}
	}
}

func TestFetchPriceErrorFirstProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	priceProvider := mock_market.NewMockPriceProvider(ctrl)
	priceProvider.SetMockPrices(mockPrices)

	customErr := errors.New("error")
	priceProviderWithError := mock_market.NewMockPriceProviderWithError(ctrl, customErr)

	symbols := []string{"BTC", "ETH"}
	currencies := []string{"USD", "EUR"}

	manager, close := setupMarketManager(t, []thirdparty.MarketDataProvider{priceProviderWithError, priceProvider}, &event.Feed{})
	t.Cleanup(close)

	rst, err := manager.FetchPrices(symbols, currencies)
	require.NoError(t, err)
	for _, symbol := range symbols {
		for _, currency := range currencies {
			require.Equal(t, rst[symbol][currency], mockPrices[symbol][currency])
		}
	}
}

func setMarketCacheForTesting(t *testing.T, manager *Manager, currency string, marketValues map[string]thirdparty.TokenMarketValues) {
	t.Helper()
	manager.updateMarketCache(currency, marketValues)
}

func TestGetOrFetchTokenMarketValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTokenMarketValues := map[string]thirdparty.TokenMarketValues{
		"BTC": {
			MKTCAP:          1000000000,
			HIGHDAY:         1.23456,
			LOWDAY:          1.00000,
			CHANGEPCTHOUR:   0.1,
			CHANGEPCTDAY:    0.2,
			CHANGEPCT24HOUR: 0.3,
			CHANGE24HOUR:    0.4,
		},
		"ETH": {
			MKTCAP:          2000000000,
			HIGHDAY:         4.56789,
			LOWDAY:          4.00000,
			CHANGEPCTHOUR:   0.5,
			CHANGEPCTDAY:    0.6,
			CHANGEPCT24HOUR: 0.7,
			CHANGE24HOUR:    0.8,
		},
	}
	updatedTokenMarketValues := map[string]thirdparty.TokenMarketValues{
		"BTC": {
			MKTCAP:          1000000000,
			HIGHDAY:         2.23456,
			LOWDAY:          1.00000,
			CHANGEPCTHOUR:   0.1,
			CHANGEPCTDAY:    0.2,
			CHANGEPCT24HOUR: 0.3,
			CHANGE24HOUR:    0.4,
		},
		"ETH": {
			MKTCAP:          2000000000,
			HIGHDAY:         5.56789,
			LOWDAY:          4.00000,
			CHANGEPCTHOUR:   0.5,
			CHANGEPCTDAY:    0.6,
			CHANGEPCT24HOUR: 0.7,
			CHANGE24HOUR:    0.8,
		},
	}
	requestCurrency := "EUR"
	requestSymbols := []string{"BTC", "ETH"}
	testCases := []struct {
		description                string
		requestMaxCachedAgeSeconds int64

		cachedTokenMarketValues map[string]thirdparty.TokenMarketValues
		fetchTokenMarketValues  map[string]thirdparty.TokenMarketValues
		fetchErr                error

		wantFetchSymbols []string
		wantValues       map[string]thirdparty.TokenMarketValues
		wantErr          error
	}{
		{
			description:                "fetch errors are propagated",
			requestMaxCachedAgeSeconds: 0,
			cachedTokenMarketValues:    nil,
			fetchTokenMarketValues:     nil,
			fetchErr:                   errors.New("explosion"),

			wantFetchSymbols: requestSymbols,
			wantValues:       nil,
			wantErr:          errors.New("explosion"),
		},
		{
			description:                "token values fetched if not cached",
			requestMaxCachedAgeSeconds: 10,
			cachedTokenMarketValues:    nil,
			fetchTokenMarketValues:     initialTokenMarketValues,
			fetchErr:                   nil,

			wantFetchSymbols: requestSymbols,
			wantValues:       initialTokenMarketValues,
			wantErr:          nil,
		},
		{
			description:                "token values returned from cache if fresh",
			requestMaxCachedAgeSeconds: 10,
			cachedTokenMarketValues:    initialTokenMarketValues,
			fetchTokenMarketValues:     nil,
			fetchErr:                   nil,

			wantFetchSymbols: requestSymbols,
			wantValues:       initialTokenMarketValues,
			wantErr:          nil,
		},
		{
			description:                "token values fetched if fetch forced",
			requestMaxCachedAgeSeconds: MaxAgeInSecondsForFresh, // N.B. Force a fetch
			cachedTokenMarketValues:    initialTokenMarketValues,
			fetchTokenMarketValues:     updatedTokenMarketValues,
			fetchErr:                   nil,

			wantFetchSymbols: requestSymbols,
			wantValues:       updatedTokenMarketValues,
			wantErr:          nil,

			// TODO: Implement more test cases
			// Test Case: There's cache, but we want fresh data, but fetch fails, we should fallback to cache
		},
	}

	for _, tc := range testCases {
		provider := mock_thirdparty.NewMockMarketDataProvider(ctrl)
		provider.EXPECT().ID().Return("MockMarketProvider").AnyTimes()
		manager, close := setupMarketManager(t, []thirdparty.MarketDataProvider{provider}, &event.Feed{})
		t.Cleanup(close)

		t.Run(tc.description, func(t *testing.T) {
			if tc.cachedTokenMarketValues != nil {
				setMarketCacheForTesting(t, manager, requestCurrency, tc.cachedTokenMarketValues)
			}

			if tc.fetchTokenMarketValues != nil || tc.fetchErr != nil {
				provider.EXPECT().FetchTokenMarketValues(tc.wantFetchSymbols, requestCurrency).Return(tc.fetchTokenMarketValues, tc.fetchErr)
			}

			gotValues, gotErr := manager.GetOrFetchTokenMarketValues(requestSymbols, requestCurrency, tc.requestMaxCachedAgeSeconds)
			if tc.wantErr != nil {
				require.ErrorContains(t, gotErr, tc.wantErr.Error())
			} else {
				require.NoError(t, gotErr)
			}
			require.Equal(t, tc.wantValues, gotValues)
		})
	}
}

func TestSymbolsMapping(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	provider := mock_thirdparty.NewMockMarketDataProvider(ctrl)
	provider.EXPECT().ID().Return("MockMarketProvider").AnyTimes()
	marketManager, close := setupMarketManager(t, []thirdparty.MarketDataProvider{provider}, &event.Feed{})
	t.Cleanup(close)

	// no symbols provided
	symbolsToProviderSymbols, providerSymbolsToSymbols, err := marketManager.symbolProviderSymbolMaps([]string{})
	require.NoError(t, err)
	require.Empty(t, symbolsToProviderSymbols)
	require.Empty(t, providerSymbolsToSymbols)

	// regular symbols
	symbols := []string{"ETH", "BTC"}
	expectedSymbolsToProviderSymbols := map[string]string{
		"ETH": "ETH",
		"BTC": "BTC",
	}
	expectedProviderSymbolsToSymbols := map[string][]string{
		"ETH": {"ETH"},
		"BTC": {"BTC"},
	}
	symbolsToProviderSymbols, providerSymbolsToSymbols, err = marketManager.symbolProviderSymbolMaps(symbols)
	require.NoError(t, err)
	require.Equal(t, expectedSymbolsToProviderSymbols, symbolsToProviderSymbols)
	require.Equal(t, expectedProviderSymbolsToSymbols, providerSymbolsToSymbols)

	// symbols with decimals
	symbols = []string{"SNT", "DAI", "USDC (EVM)", "USDC (BSC)", "ANYTHING"}
	expectedSymbolsToProviderSymbols = map[string]string{
		"SNT":        "SNT",
		"DAI":        "DAI",
		"USDC (EVM)": "USDC",
		"USDC (BSC)": "USDC",
		"ANYTHING":   "ANYTHING",
	}
	expectedProviderSymbolsToSymbols = map[string][]string{
		"SNT":      {"SNT"},
		"DAI":      {"DAI"},
		"USDC":     {"USDC (EVM)", "USDC (BSC)"},
		"ANYTHING": {"ANYTHING"},
	}
	symbolsToProviderSymbols, providerSymbolsToSymbols, err = marketManager.symbolProviderSymbolMaps(symbols)
	require.NoError(t, err)
	require.Equal(t, expectedSymbolsToProviderSymbols, symbolsToProviderSymbols)
	require.Equal(t, expectedProviderSymbolsToSymbols, providerSymbolsToSymbols)
}
