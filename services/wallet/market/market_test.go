package market

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/stretchr/testify/require"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	provider_errors "github.com/status-im/status-go/internal/healthmanager/provider_errors"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
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
	priceProvider := NewMockPriceProvider(ctrl)
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
	priceProvider := NewMockPriceProvider(ctrl)
	priceProvider.SetMockPrices(mockPrices)

	customErr := errors.New("error")
	priceProviderWithError := NewMockPriceProviderWithError(ctrl, customErr)

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

type staticTokenManager struct {
	tokensByKey map[string]*tokentypes.Token
}

func newStaticTokenManager(tokens ...*tokentypes.Token) *staticTokenManager {
	tokensByKey := make(map[string]*tokentypes.Token, len(tokens))
	for _, token := range tokens {
		tokensByKey[token.Key()] = token
	}
	return &staticTokenManager{tokensByKey: tokensByKey}
}

func (m *staticTokenManager) GetTokensForFetchingMarketData() ([]*tokentypes.Token, error) {
	tokens := make([]*tokentypes.Token, 0, len(m.tokensByKey))
	for _, token := range m.tokensByKey {
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (m *staticTokenManager) GetTokensByKeysForFetchingMarketData(tokenKeys []string) ([]*tokentypes.Token, error) {
	tokens := make([]*tokentypes.Token, 0, len(tokenKeys))
	for _, key := range tokenKeys {
		if token, ok := m.tokensByKey[key]; ok {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (m *staticTokenManager) GetTokenByKey(tokenKey string) (*tokentypes.Token, error) {
	token, ok := m.tokensByKey[tokenKey]
	if !ok {
		return nil, fmt.Errorf("token not found for key: %s", tokenKey)
	}
	return token, nil
}

type historicalPriceProvider struct {
	id            string
	historyByKey  map[string][]thirdparty.HistoricalPrice
	defaultErr    error
	requestedKeys []string
}

func (p *historicalPriceProvider) ID() string {
	if p.id == "" {
		return "historical-provider"
	}
	return p.id
}

func (p *historicalPriceProvider) FetchPrices(tokens []*tokentypes.Token, currencies []string) (map[string]map[string]float64, error) {
	return map[string]map[string]float64{}, nil
}

func (p *historicalPriceProvider) FetchHistoricalDailyPrices(token *tokentypes.Token, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	p.requestedKeys = append(p.requestedKeys, token.Key())
	if history, ok := p.historyByKey[token.Key()]; ok {
		return history, nil
	}
	return nil, p.defaultErr
}

func (p *historicalPriceProvider) FetchHistoricalHourlyPrices(token *tokentypes.Token, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	return p.FetchHistoricalDailyPrices(token, currency, limit, false, aggregate)
}

func (p *historicalPriceProvider) FetchTokenMarketValues(tokens []*tokentypes.Token, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	return map[string]thirdparty.TokenMarketValues{}, nil
}

func (p *historicalPriceProvider) FetchTokenDetails(tokens []*tokentypes.Token) (map[string]thirdparty.TokenDetails, error) {
	return map[string]thirdparty.TokenDetails{}, nil
}

func TestFetchHistoricalDailyPricesFallsBackToMainnetSibling(t *testing.T) {
	baseToken := &tokentypes.Token{Token: &types.Token{
		ChainID:      walletcommon.BaseMainnet,
		Address:      common.HexToAddress("0x662015ec830df08c0fc45896fab726542e8ac09e"),
		CrossChainID: walletcommon.StatusMainnetTokenCrossChainID,
	}}
	ethToken := &tokentypes.Token{Token: &types.Token{
		ChainID:      walletcommon.EthereumMainnet,
		Address:      common.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"),
		CrossChainID: walletcommon.StatusMainnetTokenCrossChainID,
	}}

	provider := &historicalPriceProvider{
		historyByKey: map[string][]thirdparty.HistoricalPrice{
			ethToken.Key(): []thirdparty.HistoricalPrice{{Timestamp: 1000, Value: 0.1}},
		},
		defaultErr: thirdparty.ErrTokenNotMapped,
	}

	manager := NewManager([]thirdparty.MarketDataProvider{provider}, newStaticTokenManager(baseToken, ethToken), &event.Feed{})
	prices, err := manager.FetchHistoricalDailyPrices(baseToken.Key(), "usd", 30, false, 1)
	require.NoError(t, err)
	require.Equal(t, []thirdparty.HistoricalPrice{{Timestamp: 1000, Value: 0.1}}, prices)
	require.Equal(t, []string{baseToken.Key(), ethToken.Key()}, provider.requestedKeys)
}

func TestFetchHistoricalDailyPricesPropagatesJoinedUnmappedAndConnectivityError(t *testing.T) {
	baseToken := &tokentypes.Token{Token: &types.Token{
		ChainID:      walletcommon.BaseMainnet,
		Address:      common.HexToAddress("0x662015ec830df08c0fc45896fab726542e8ac09e"),
		CrossChainID: walletcommon.StatusMainnetTokenCrossChainID,
	}}

	unmappedProvider := &historicalPriceProvider{
		id:         "unmapped-provider",
		defaultErr: thirdparty.ErrTokenNotMapped,
	}
	failingProvider := &historicalPriceProvider{
		id:         "failing-provider",
		defaultErr: &net.OpError{Op: "dial", Err: errors.New("connection refused")},
	}

	manager := NewManager(
		[]thirdparty.MarketDataProvider{unmappedProvider, failingProvider},
		newStaticTokenManager(baseToken),
		&event.Feed{},
	)
	prices, err := manager.FetchHistoricalDailyPrices(baseToken.Key(), "usd", 30, false, 1)
	require.Error(t, err)
	require.Nil(t, prices)
	require.False(t, provider_errors.IsIgnorableForConnectivity(err))
}

func TestFetchHistoricalDailyPricesReturnsEmptyAndStaysConnectedForUnmappedToken(t *testing.T) {
	baseToken := &tokentypes.Token{Token: &types.Token{
		ChainID:      walletcommon.BaseMainnet,
		Address:      common.HexToAddress("0x662015ec830df08c0fc45896fab726542e8ac09e"),
		CrossChainID: walletcommon.StatusMainnetTokenCrossChainID,
	}}

	provider := &historicalPriceProvider{
		defaultErr: thirdparty.ErrTokenNotMapped,
	}

	manager := NewManager([]thirdparty.MarketDataProvider{provider}, newStaticTokenManager(baseToken), &event.Feed{})
	prices, err := manager.FetchHistoricalDailyPrices(baseToken.Key(), "usd", 30, false, 1)
	require.NoError(t, err)
	require.Empty(t, prices)
	require.True(t, manager.IsConnected)
}
