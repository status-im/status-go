package market

import (
	"errors"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/stretchr/testify/require"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	mock_market "github.com/status-im/status-go/services/wallet/market/mock"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

// MockTokenManager implements TokenManagerInterface for testing
type MockTokenManager struct{}

func (m *MockTokenManager) GetTokensForFetchingMarketData() ([]*tokentypes.Token, error) {
	return m.GetTokensOfInterestForActiveNetworksMode()
}

func (m *MockTokenManager) GetTokensByKeysForFetchingMarketData(tokenKeys []string) ([]*tokentypes.Token, error) {
	return m.GetTokensByKeys(tokenKeys)
}

func (m *MockTokenManager) GetTokensByKeys(tokenKeys []string) ([]*tokentypes.Token, error) {
	var tokens []*tokentypes.Token
	for _, key := range tokenKeys {
		for i, testKey := range testTokensKeys {
			if key == testKey {
				address := common.HexToAddress(fmt.Sprintf("0x000000000000000000000000000000000000000%d", i+1))
				token := &tokentypes.Token{
					Token: &types.Token{
						ChainID:  1,
						Address:  address,
						Symbol:   fmt.Sprintf("TOKEN%d", i+1),
						Name:     fmt.Sprintf("Test Token %d", i+1),
						Decimals: 18,
					},
				}
				tokens = append(tokens, token)
				break
			}
		}
	}
	return tokens, nil
}

func (m *MockTokenManager) GetTokensOfInterestForActiveNetworksMode() ([]*tokentypes.Token, error) {
	return m.GetTokensByKeys(testTokensKeys)
}

func (m *MockTokenManager) GetTokenByKey(tokenKey string) (*tokentypes.Token, error) {
	tokens, err := m.GetTokensByKeys([]string{tokenKey})
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("token not found for key: %s", tokenKey)
	}
	return tokens[0], nil
}

func setupMarketManager(t *testing.T, providers []thirdparty.MarketDataProvider, feedEvent *event.Feed) *Manager {
	mockTokenManager := &MockTokenManager{}
	manager := NewManager(providers, mockTokenManager, feedEvent)
	return manager
}

var mockPrices = map[string]map[string]float64{
	"1-0x0000000000000000000000000000000000000001": {
		"USD": 1.23456,
		"EUR": 2.34567,
		"DAI": 3.45678,
		"ARS": 9.87654,
	},
	"1-0x0000000000000000000000000000000000000002": {
		"USD": 4.56789,
		"EUR": 5.67891,
		"DAI": 6.78912,
		"ARS": 8.76543,
	},
	"1-0x0000000000000000000000000000000000000003": {
		"USD": 7.654,
		"EUR": 6.0,
		"DAI": 1455.12,
		"ARS": 0.0,
	},
}

var testTokensKeys = []string{
	types.TokenKey(1, common.HexToAddress("0x0000000000000000000000000000000000000001")),
	types.TokenKey(1, common.HexToAddress("0x0000000000000000000000000000000000000002")),
	types.TokenKey(1, common.HexToAddress("0x0000000000000000000000000000000000000003")),
}

func TestPrice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	priceProvider := mock_market.NewMockPriceProvider(ctrl)
	priceProvider.SetMockPrices(mockPrices)

	manager := setupMarketManager(t, []thirdparty.MarketDataProvider{priceProvider, priceProvider}, &event.Feed{})

	{
		rst := manager.priceCache.Get()
		require.Empty(t, rst)
	}

	{
		currencies := []string{"USD"}
		rst, err := manager.FetchPrices(testTokensKeys, currencies)
		require.NoError(t, err)
		for _, tokenKey := range testTokensKeys {
			for _, currency := range currencies {
				require.Equal(t, rst[tokenKey][currency], mockPrices[tokenKey][currency])
			}
		}
	}

	{
		currencies := []string{"USD", "EUR", "DAI", "ARS"}
		rst, err := manager.FetchPrices(testTokensKeys, currencies)
		require.NoError(t, err)
		for _, tokenKey := range testTokensKeys {
			for _, currency := range currencies {
				require.Equal(t, rst[tokenKey][currency], mockPrices[tokenKey][currency])
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

	currencies := []string{"USD", "EUR"}

	manager := setupMarketManager(t, []thirdparty.MarketDataProvider{priceProviderWithError, priceProvider}, &event.Feed{})

	rst, err := manager.FetchPrices(testTokensKeys, currencies)
	require.NoError(t, err)
	for _, tokenKey := range testTokensKeys {
		for _, currency := range currencies {
			require.Equal(t, rst[tokenKey][currency], mockPrices[tokenKey][currency])
		}
	}
}
