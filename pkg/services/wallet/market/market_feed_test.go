package market

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

type MarketTestSuite struct {
	suite.Suite
	feedSub    *FeedSubscription
	tokensKeys []string
	currencies []string
}

func (s *MarketTestSuite) SetupTest() {
	feed := new(event.Feed)
	s.feedSub = NewFeedSubscription(feed)

	// Create test tokens
	s.tokensKeys = []string{
		types.TokenKey(1, common.HexToAddress("0x0000000000000000000000000000000000000000")),
		types.TokenKey(1, common.HexToAddress("0x0000000000000000000000000000000000000001")),
	}
	s.currencies = []string{"USD", "EUR"}
}

func (s *MarketTestSuite) TearDownTest() {
	s.feedSub.Close()
}

func (s *MarketTestSuite) TestEventOnRpsError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	// GIVEN
	customErr := errors.New("request rate exceeded")
	priceProviderWithError := NewMockPriceProviderWithError(ctrl, customErr)
	manager := setupMarketManager(s.T(), []thirdparty.MarketDataProvider{priceProviderWithError}, s.feedSub.GetFeed())

	// WHEN
	_, err := manager.FetchPrices(s.tokensKeys, s.currencies)
	s.Require().Error(err, "expected error from FetchPrices due to MockPriceProviderWithError")
	_, ok := s.feedSub.WaitForEvent(500 * time.Millisecond)
	s.Require().False(ok, "expected no status event for non-critical rate limit error")
}

func (s *MarketTestSuite) TestEventOnNetworkError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	// GIVEN
	customErr := errors.New("dial tcp: lookup optimism-goerli.infura.io: no such host")
	priceProviderWithError := NewMockPriceProviderWithError(ctrl, customErr)
	manager := setupMarketManager(s.T(), []thirdparty.MarketDataProvider{priceProviderWithError}, s.feedSub.GetFeed())
	manager.downDebounce = 200 * time.Millisecond

	_, err := manager.FetchPrices(s.tokensKeys, s.currencies)
	s.Require().Error(err, "expected error from FetchPrices due to MockPriceProviderWithError")
	_, ok := s.feedSub.WaitForEvent(100 * time.Millisecond)
	s.Require().False(ok, "network errors must not emit market down before downDebounce")

	event, ok := s.feedSub.WaitForEvent(2 * time.Second)
	s.Require().True(ok, "expected a delayed down event")
	s.Require().Equal(EventMarketStatusChanged, event.Type)
	s.Require().Equal("down", event.Message)
}

func (s *MarketTestSuite) TestNetworkErrorThenRecoveryCancelsDown() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	failing := NewMockPriceProviderWithError(ctrl, errors.New("dial tcp: lookup api.coingecko.com: no such host"))
	okProvider := NewMockPriceProvider(ctrl)
	okProvider.SetMockPrices(mockPrices)

	failingManager := setupMarketManager(s.T(), []thirdparty.MarketDataProvider{failing}, s.feedSub.GetFeed())
	failingManager.downDebounce = 100 * time.Millisecond
	_, err := failingManager.FetchPrices(testTokensKeys, s.currencies)
	s.Require().Error(err)

	failingManager.providers = []thirdparty.MarketDataProvider{okProvider}
	_, err = failingManager.FetchPrices(testTokensKeys, s.currencies)
	s.Require().NoError(err)

	_, ok := s.feedSub.WaitForEvent(500 * time.Millisecond)
	s.Require().False(ok, "recovery before debounce must not emit market down")
	s.Require().True(failingManager.IsConnected)
}

func (s *MarketTestSuite) TestStopCancelsPendingDown() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	failing := NewMockPriceProviderWithError(ctrl, errors.New("dial tcp: lookup api.coingecko.com: no such host"))
	manager := setupMarketManager(s.T(), []thirdparty.MarketDataProvider{failing}, s.feedSub.GetFeed())
	manager.downDebounce = 100 * time.Millisecond
	_, err := manager.FetchPrices(testTokensKeys, s.currencies)
	s.Require().Error(err)

	manager.Stop()
	_, ok := s.feedSub.WaitForEvent(500 * time.Millisecond)
	s.Require().False(ok, "stop must cancel pending market down")
	s.Require().True(manager.IsConnected)
}

func TestMarketTestSuite(t *testing.T) {
	suite.Run(t, new(MarketTestSuite))
}
