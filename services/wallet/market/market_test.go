package market

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/ethereum/go-ethereum/event"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	mock_thirdparty "github.com/status-im/status-go/services/wallet/thirdparty/mock"
)

type MockPriceProvider struct {
	mock_thirdparty.MockMarketDataProvider
	mockPrices map[string]map[string]float64
}

func NewMockPriceProvider(ctrl *gomock.Controller) *MockPriceProvider {
	return &MockPriceProvider{
		MockMarketDataProvider: *mock_thirdparty.NewMockMarketDataProvider(ctrl),
	}
}

func (mpp *MockPriceProvider) setMockPrices(prices map[string]map[string]float64) {
	mpp.mockPrices = prices
}

func (mpp *MockPriceProvider) ID() string {
	return "MockPriceProvider"
}

func (mpp *MockPriceProvider) FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	res := make(map[string]map[string]float64)
	for _, symbol := range symbols {
		res[symbol] = make(map[string]float64)
		for _, currency := range currencies {
			res[symbol][currency] = mpp.mockPrices[symbol][currency]
		}
	}
	return res, nil
}

type MockPriceProviderWithError struct {
	MockPriceProvider
}

func (mpp *MockPriceProviderWithError) FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	return nil, errors.New("error")
}

func setupMarketManager(t *testing.T, providers []thirdparty.MarketDataProvider) *Manager {
	return NewManager(providers, &event.Feed{})
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
	priceProvider := NewMockPriceProvider(ctrl)
	priceProvider.setMockPrices(mockPrices)

	manager := setupMarketManager(t, []thirdparty.MarketDataProvider{priceProvider, priceProvider})

	{
		rst := manager.GetCachedPrices()
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

	cache := manager.GetCachedPrices()
	for symbol, pricePerCurrency := range mockPrices {
		for currency, price := range pricePerCurrency {
			require.Equal(t, price, cache[symbol][currency].Price)
		}
	}
}

func TestFetchPriceErrorFirstProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	priceProvider := NewMockPriceProvider(ctrl)
	priceProvider.setMockPrices(mockPrices)
	priceProviderWithError := &MockPriceProviderWithError{}
	symbols := []string{"BTC", "ETH"}
	currencies := []string{"USD", "EUR"}

	manager := setupMarketManager(t, []thirdparty.MarketDataProvider{priceProviderWithError, priceProvider})
	rst, err := manager.FetchPrices(symbols, currencies)
	require.NoError(t, err)
	for _, symbol := range symbols {
		for _, currency := range currencies {
			require.Equal(t, rst[symbol][currency], mockPrices[symbol][currency])
		}
	}
}

func TestFetchTokenMarketValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	symbols := []string{"BTC", "ETH"}
	currency := "EUR"
	expectedMarketValues := map[string]thirdparty.TokenMarketValues{
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

	// Can't use fake provider, because the key {receiver, method} will be different, no match
	provider := mock_thirdparty.NewMockMarketDataProvider(ctrl)
	provider.EXPECT().ID().Return("MockPriceProvider").AnyTimes()
	provider.EXPECT().FetchTokenMarketValues(symbols, currency).Return(expectedMarketValues, nil)
	manager := setupMarketManager(t, []thirdparty.MarketDataProvider{provider})
	marketValues, err := manager.FetchTokenMarketValues(symbols, currency)
	require.NoError(t, err)
	require.Equal(t, expectedMarketValues, marketValues)

	// Test error
	provider.EXPECT().FetchTokenMarketValues(symbols, currency).Return(nil, errors.New("error"))
	marketValues, err = manager.FetchTokenMarketValues(symbols, currency)
	require.Error(t, err)
	require.Nil(t, marketValues)
}

func TestGetOrFetchTokenMarketValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	symbols := []string{"BTC", "ETH"}
	currency := "EUR"
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

	provider := mock_thirdparty.NewMockMarketDataProvider(ctrl)
	provider.EXPECT().ID().Return("MockMarketProvider").AnyTimes()
	manager := setupMarketManager(t, []thirdparty.MarketDataProvider{provider})

	// Test: ensure errors are propagated
	provider.EXPECT().FetchTokenMarketValues(symbols, currency).Return(nil, errors.New("error"))
	marketValues, err := manager.GetOrFetchTokenMarketValues(symbols, currency, 0)
	require.Error(t, err)
	require.Nil(t, marketValues)

	// Test: ensure token market values are retrieved
	provider.EXPECT().FetchTokenMarketValues(symbols, currency).Return(initialTokenMarketValues, nil)
	marketValues, err = manager.GetOrFetchTokenMarketValues(symbols, currency, 10)
	require.NoError(t, err)
	require.Equal(t, initialTokenMarketValues, marketValues)

	// Test: ensure token market values are cached
	provider.EXPECT().FetchTokenMarketValues(symbols, currency).Return(updatedTokenMarketValues, nil)
	marketValues, err = manager.GetOrFetchTokenMarketValues(symbols, currency, 10)
	require.NoError(t, err)
	require.Equal(t, initialTokenMarketValues, marketValues)

	// Test: ensure token market values are updated
	marketValues, err = manager.GetOrFetchTokenMarketValues(symbols, currency, -1)
	require.NoError(t, err)
	require.Equal(t, updatedTokenMarketValues, marketValues)
}

func TestGetCachedTokenMarketValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	symbols := []string{"BTC", "ETH"}
	currency := "EUR"
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

	provider := mock_thirdparty.NewMockMarketDataProvider(ctrl)
	provider.EXPECT().ID().Return("MockMarketProvider").AnyTimes()
	manager := setupMarketManager(t, []thirdparty.MarketDataProvider{provider})

	// Test: ensure token market cache is empty
	tokenMarketCache := manager.GetCachedTokenMarketValues()
	require.Empty(t, tokenMarketCache)

	// Test: ensure token market values are retrieved
	provider.EXPECT().FetchTokenMarketValues(symbols, currency).Return(initialTokenMarketValues, nil)
	marketValues, err := manager.GetOrFetchTokenMarketValues(symbols, currency, 10)
	tokenMarketCache = manager.GetCachedTokenMarketValues()
	require.NoError(t, err)

	for _, token := range symbols {
		tokenMarketValues := marketValues[token]
		cachedTokenMarketValues := tokenMarketCache[currency][token]
		require.Equal(t, cachedTokenMarketValues.MarketValues, tokenMarketValues)
	}
}
