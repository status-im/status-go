package newsfeed

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"

	gocommon "github.com/status-im/status-go/common"
)

// TODO replace with the real Status feed URL
const STATUS_FEED_URL = "https://hnrss.org/frontpage"

type FeedParser interface {
	ParseURL(url string) (*gofeed.Feed, error)
}

type FeedHandler interface {
	HandleFeed(item *gofeed.Item)
}

type NewsFeedManager struct {
	url             string
	parser          FeedParser
	handler         FeedHandler
	fetchFrom       time.Time
	pollingInterval time.Duration
	cancel          context.CancelFunc
}

type Option func(*NewsFeedManager)

func WithURL(url string) Option {
	return func(nfm *NewsFeedManager) {
		nfm.url = url
	}
}

func WithParser(parser FeedParser) Option {
	return func(nfm *NewsFeedManager) {
		nfm.parser = parser
	}
}

func WithHandler(handler FeedHandler) Option {
	return func(nfm *NewsFeedManager) {
		nfm.handler = handler
	}
}

func WithPollingInterval(interval time.Duration) Option {
	return func(nfm *NewsFeedManager) {
		nfm.pollingInterval = interval
	}
}

func WithFetchFrom(t time.Time) Option {
	return func(nfm *NewsFeedManager) {
		nfm.fetchFrom = t
	}
}

func NewNewsFeedManager(opts ...Option) *NewsFeedManager {
	nfm := &NewsFeedManager{
		pollingInterval: time.Minute * 30,
		fetchFrom:       time.Now(),
	}

	for _, opt := range opts {
		opt(nfm)
	}

	return nfm
}

func (n *NewsFeedManager) fetchRSS() error {
	feed, err := n.parser.ParseURL(n.url)
	if err != nil {
		fmt.Println("Error fetching feed:", err)
		return err
	}

	fmt.Println("Feed Title:", feed.Title)
	for _, item := range feed.Items {
		if item.PublishedParsed != nil && item.PublishedParsed.After(n.fetchFrom) {
			fmt.Printf("NEW ITEM:\n  Title: %s\n  Link: %s\n  Published: %s\n\n",
				item.Title, item.Link, item.PublishedParsed)
			n.handler.HandleFeed(item)
		}
	}

	// Update fetchFrom to now
	n.fetchFrom = time.Now()

	return nil
}

func (n *NewsFeedManager) StartFetching(ctx context.Context) {
	// Derive the given context, save the CancelFunc
	ctx, n.cancel = context.WithCancel(ctx)

	// Initial fetch
	_ = n.fetchRSS()

	ticker := time.NewTicker(n.pollingInterval)

	go func() {
		defer gocommon.LogOnPanic()
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = n.fetchRSS()
			case <-ctx.Done():
				// TODO use logger
				fmt.Println("Polling stopped for:", n.url)
				return
			}
		}
	}()
}

func (n *NewsFeedManager) StopFetching() {
	n.cancel()
}
