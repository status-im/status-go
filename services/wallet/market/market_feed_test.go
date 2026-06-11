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

	"github.com/status-im/status-go/services/wallet/thirdparty"
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

	_, err := manager.FetchPrices(s.tokensKeys, s.currencies)
	s.Require().Error(err, "expected error from FetchPrices due to MockPriceProviderWithError")
	event, ok := s.feedSub.WaitForEvent(500 * time.Millisecond)
	s.Require().True(ok, "expected an event, but none was received")

	// THEN
	s.Require().Equal(event.Type, EventMarketStatusChanged)
}

func TestMarketTestSuite(t *testing.T) {
	suite.Run(t, new(MarketTestSuite))
}
