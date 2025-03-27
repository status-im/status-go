package newsfeed

import (
	"context"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/suite"
)

type NewsFeedManagerSuite struct {
	suite.Suite
}

type MockParser struct {
	Feed *gofeed.Feed
	Err  error
}

func (mp *MockParser) ParseURL(url string) (*gofeed.Feed, error) {
	return mp.Feed, mp.Err
}

type MockHandler struct {
	callback func(item *gofeed.Item)
}

func (mh *MockHandler) HandleFeed(item *gofeed.Item) {
	mh.callback(item)
}

func TestNewsFeedManagerSuite(t *testing.T) {
	suite.Run(t, new(NewsFeedManagerSuite))
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func (s *NewsFeedManagerSuite) TestFetchOnlyItemNewerThanTwoHours() {
	now := time.Now()
	items := []*gofeed.Item{
		{Title: "Old Item", PublishedParsed: ptrTime(now.Add(-48 * time.Hour))},
		{Title: "New Item", PublishedParsed: ptrTime(now.Add(-1 * time.Hour))},
	}
	mockFeed := &gofeed.Feed{
		Title: "Test Feed",
		Items: items,
	}
	captured := []*gofeed.Item{}

	myCallback := func(item *gofeed.Item) {
		captured = append(captured, item)
	}

	twoHoursAgo := now.Add(-2 * time.Hour)

	newsFeedManager := NewNewsFeedManager(
		WithURL("mock-url"),
		WithParser(&MockParser{Feed: mockFeed}),
		WithHandler(&MockHandler{callback: myCallback}),
		WithPollingInterval(60*time.Second),
		WithFetchFrom(twoHoursAgo),
	)

	err := newsFeedManager.fetchRSS()
	s.Require().NoError(err)

	s.Require().Len(captured, 1)
	s.Require().Equal("New Item", captured[0].Title)
}

func (s *NewsFeedManagerSuite) TestStartAndStopFetching() {
	now := time.Now()
	items := []*gofeed.Item{
		{Title: "Old Item", PublishedParsed: ptrTime(now.Add(-48 * time.Hour))},
		{Title: "New Item", PublishedParsed: ptrTime(now.Add(-1 * time.Hour))},
	}
	mockFeed := &gofeed.Feed{
		Title: "Test Feed",
		Items: items,
	}
	captured := []*gofeed.Item{}

	myCallback := func(item *gofeed.Item) {
		captured = append(captured, item)
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

	newsFeedManager.StartFetching(ctx)

	// The start fetching does an initial fetch immediately
	s.Require().Len(captured, 1)
	s.Require().Equal("New Item", captured[0].Title)

	newsFeedManager.StopFetching()
}
