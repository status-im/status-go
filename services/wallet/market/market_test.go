package market

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/event"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type MockPriceProvider struct {
	mockPrices map[string]map[string]float64
}

func NewMockPriceProvider() *MockPriceProvider {
	return &MockPriceProvider{}
}

func (mpp *MockPriceProvider) setMockPrices(prices map[string]map[string]float64) {
	mpp.mockPrices = prices
}

func (mpp *MockPriceProvider) FetchHistoricalDailyPrices(symbol string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	return nil, errors.New("not implmented")
}
func (mpp *MockPriceProvider) FetchHistoricalHourlyPrices(symbol string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	return nil, errors.New("not implmented")
}
func (mpp *MockPriceProvider) FetchTokenMarketValues(symbols []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	return nil, errors.New("not implmented")
}
func (mpp *MockPriceProvider) FetchTokenDetails(symbols []string) (map[string]thirdparty.TokenDetails, error) {
	return nil, errors.New("not implmented")
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

func setupTestPrice(t *testing.T, providers []thirdparty.MarketDataProvider) *Manager {
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
	priceProvider := NewMockPriceProvider()
	priceProvider.setMockPrices(mockPrices)

	manager := setupTestPrice(t, []thirdparty.MarketDataProvider{priceProvider, priceProvider})

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
	priceProvider := NewMockPriceProvider()
	priceProvider.setMockPrices(mockPrices)
	priceProviderWithError := &MockPriceProviderWithError{}
	symbols := []string{"BTC", "ETH"}
	currencies := []string{"USD", "EUR"}

	manager := setupTestPrice(t, []thirdparty.MarketDataProvider{priceProviderWithError, priceProvider})
	rst, err := manager.FetchPrices(symbols, currencies)
	require.NoError(t, err)
	for _, symbol := range symbols {
		for _, currency := range currencies {
			require.Equal(t, rst[symbol][currency], mockPrices[symbol][currency])
		}
	}
}
