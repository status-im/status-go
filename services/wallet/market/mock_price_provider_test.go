package market

import (
	"go.uber.org/mock/gomock"

	mock_thirdparty "github.com/status-im/status-go/services/wallet/thirdparty/mock"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
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

func (mpp *MockPriceProvider) SetMockPrices(prices map[string]map[string]float64) {
	mpp.mockPrices = prices
}

func (mpp *MockPriceProvider) ID() string {
	return "MockPriceProvider"
}

func (mpp *MockPriceProvider) FetchPrices(tokens []*tokentypes.Token, currencies []string) (map[string]map[string]float64, error) {
	res := make(map[string]map[string]float64)
	for _, token := range tokens {
		res[token.Key()] = make(map[string]float64)
		for _, currency := range currencies {
			res[token.Key()][currency] = mpp.mockPrices[token.Key()][currency]
		}
	}
	return res, nil
}

type MockPriceProviderWithError struct {
	MockPriceProvider
	err error
}

// NewMockPriceProviderWithError creates a new MockPriceProviderWithError with the specified error
func NewMockPriceProviderWithError(ctrl *gomock.Controller, err error) *MockPriceProviderWithError {
	return &MockPriceProviderWithError{
		MockPriceProvider: *NewMockPriceProvider(ctrl),
		err:               err,
	}
}

func (mpp *MockPriceProviderWithError) FetchPrices(tokens []*tokentypes.Token, currencies []string) (map[string]map[string]float64, error) {
	return nil, mpp.err
}
