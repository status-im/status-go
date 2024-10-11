package market

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/event"
	mock_common "github.com/status-im/status-go/services/wallet/common/mock"
	mock_market "github.com/status-im/status-go/services/wallet/market/mock"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type MarketTestSuite struct {
	suite.Suite
	feedSub    *mock_common.FeedSubscription
	symbols    []string
	currencies []string
}

func (s *MarketTestSuite) SetupTest() {
	feed := new(event.Feed)
	s.feedSub = mock_common.NewFeedSubscription(feed)

	s.symbols = []string{"BTC", "ETH"}
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
	priceProviderWithError := mock_market.NewMockPriceProviderWithError(ctrl, customErr)
	manager := NewManager([]thirdparty.MarketDataProvider{priceProviderWithError}, s.feedSub.GetFeed())

	// WHEN
	_, err := manager.FetchPrices(s.symbols, s.currencies)
	s.Require().Error(err, "expected error from FetchPrices due to MockPriceProviderWithError")
	event, ok := s.feedSub.WaitForEvent(5 * time.Second)
	s.Require().True(ok, "expected an event, but none was received")

	// THEN
	s.Require().Equal(event.Type, EventMarketStatusChanged)
}

func (s *MarketTestSuite) TestNoEventOnNetworkError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	// GIVEN
	customErr := errors.New("dial tcp: lookup optimism-goerli.infura.io: no such host")
	priceProviderWithError := mock_market.NewMockPriceProviderWithError(ctrl, customErr)
	manager := NewManager([]thirdparty.MarketDataProvider{priceProviderWithError}, s.feedSub.GetFeed())

	_, err := manager.FetchPrices(s.symbols, s.currencies)
	s.Require().Error(err, "expected error from FetchPrices due to MockPriceProviderWithError")
	_, ok := s.feedSub.WaitForEvent(time.Millisecond * 500)

	//THEN
	s.Require().False(ok, "expected no event, but one was received")
}

func TestMarketTestSuite(t *testing.T) {
	suite.Run(t, new(MarketTestSuite))
}
