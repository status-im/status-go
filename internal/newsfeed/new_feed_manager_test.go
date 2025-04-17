package newsfeed

import (
	"context"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
)

type NewsFeedManagerSuite struct {
	suite.Suite
	logger *zap.Logger
}

type MockParser struct {
	Feed *gofeed.Feed
	Err  error
}

func (mp *MockParser) ParseURL(url string) (*gofeed.Feed, error) {
	return mp.Feed, mp.Err
}

type MockHandler struct {
	callback func(item *gofeed.Item) error
}

func (mh *MockHandler) HandleFeedItemAndSend(item *gofeed.Item) error {
	return mh.callback(item)
}

func TestNewsFeedManagerSuite(t *testing.T) {
	suite.Run(t, new(NewsFeedManagerSuite))
}

func (s *NewsFeedManagerSuite) SetupTest() {
	var err error
	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)
}

func (s *NewsFeedManagerSuite) TestFetchRSS() {
	now := time.Now()
	items := []*gofeed.Item{
		{Title: "Old Item", PublishedParsed: common.Ptr(now.Add(-48 * time.Hour))},
		{Title: "New Item", PublishedParsed: common.Ptr(now.Add(-1 * time.Hour))},
	}
	mockFeed := &gofeed.Feed{
		Title: "Test Feed",
		Items: items,
	}

	twoHoursAgo := now.Add(-2 * time.Hour)

	newsFeedManager := NewNewsFeedManager(
		WithURL("mock-url"),
		WithParser(&MockParser{Feed: mockFeed}),
		WithLogger(s.logger),
		WithPollingInterval(60*time.Second),
		WithFetchFrom(twoHoursAgo),
	)

	items, err := newsFeedManager.FetchRSS()
	s.Require().NoError(err)

	s.Require().Len(items, 1)
	s.Require().Equal("New Item", items[0].Title)

	// Fetching again should not return anything as the last fetch time is now
	items, err = newsFeedManager.FetchRSS()
	s.Require().NoError(err)
	s.Require().Len(items, 0)
}

func (s *NewsFeedManagerSuite) TestFetchRSSAndHandle() {
	now := time.Now()
	items := []*gofeed.Item{
		{Title: "Old Item", PublishedParsed: common.Ptr(now.Add(-48 * time.Hour))},
		{Title: "New Item", PublishedParsed: common.Ptr(now.Add(-1 * time.Hour))},
	}
	mockFeed := &gofeed.Feed{
		Title: "Test Feed",
		Items: items,
	}
	captured := []*gofeed.Item{}

	myCallback := func(item *gofeed.Item) error {
		captured = append(captured, item)
		return nil
	}

	twoHoursAgo := now.Add(-2 * time.Hour)

	newsFeedManager := NewNewsFeedManager(
		WithURL("mock-url"),
		WithParser(&MockParser{Feed: mockFeed}),
		WithHandler(&MockHandler{callback: myCallback}),
		WithLogger(s.logger),
		WithPollingInterval(60*time.Second),
		WithFetchFrom(twoHoursAgo),
	)

	err := newsFeedManager.fetchRSSAndHandle()
	s.Require().NoError(err)

	s.Require().Len(captured, 1)
	s.Require().Equal("New Item", captured[0].Title)

	// Fetching again should not return anything as the last fetch time is now
	// Reset captured
	captured = []*gofeed.Item{}
	err = newsFeedManager.fetchRSSAndHandle()
	s.Require().NoError(err)
	s.Require().Len(captured, 0)
}

func (s *NewsFeedManagerSuite) TestStartAndStopFetching() {
	now := time.Now()
	items := []*gofeed.Item{
		{Title: "Old Item", PublishedParsed: common.Ptr(now.Add(-48 * time.Hour))},
		{Title: "New Item", PublishedParsed: common.Ptr(now.Add(-1 * time.Hour))},
	}
	mockFeed := &gofeed.Feed{
		Title: "Test Feed",
		Items: items,
	}
	captured := []*gofeed.Item{}

	myCallback := func(item *gofeed.Item) error {
		captured = append(captured, item)
		return nil
	}

	twoHoursAgo := now.Add(-2 * time.Hour)

	newsFeedManager := NewNewsFeedManager(
		WithURL("mock-url"),
		WithParser(&MockParser{Feed: mockFeed}),
		WithHandler(&MockHandler{callback: myCallback}),
		WithPollingInterval(60*time.Second),
		WithFetchFrom(twoHoursAgo),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	newsFeedManager.StartPolling(ctx)

	time.Sleep(1 * time.Millisecond) // Leave time for the go routine to run and process

	// The start fetching does an initial fetch immediately
	s.Require().Len(captured, 1)
	s.Require().Equal("New Item", captured[0].Title)
	s.Require().True(newsFeedManager.IsPolling())

	newsFeedManager.StopPolling()
	s.Require().False(newsFeedManager.IsPolling())
}
