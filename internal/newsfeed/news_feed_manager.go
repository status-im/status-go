package newsfeed

import (
	"context"
	"time"

	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
)

// TODO replace with the real Status feed URL
const STATUS_FEED_URL = "https://hnrss.org/frontpage"

type FeedParser interface {
	ParseURL(url string) (*gofeed.Feed, error)
}

type FeedHandler interface {
	HandleFeedItemAndSend(item *gofeed.Item) error
}

type NewsFeedManager struct {
	url             string
	parser          FeedParser
	handler         FeedHandler
	fetchFrom       time.Time
	pollingInterval time.Duration
	polling         bool
	cancel          context.CancelFunc
	logger          *zap.Logger
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

func WithLogger(logger *zap.Logger) Option {
	return func(nfm *NewsFeedManager) {
		nfm.logger = logger.Named("NewsFeedManager")
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
		polling:         false,
	}

	for _, opt := range opts {
		opt(nfm)
	}

	return nfm
}

func (n *NewsFeedManager) FetchRSS() ([]*gofeed.Item, error) {
	feed, err := n.parser.ParseURL(n.url)
	if err != nil {
		n.logger.Error("error fetching feed", zap.Error(err))
		return nil, err
	}

	filteredItems := []*gofeed.Item{}
	for _, item := range feed.Items {
		if item.PublishedParsed != nil && item.PublishedParsed.After(n.fetchFrom) {
			filteredItems = append(filteredItems, item)
		}
	}

	if len(filteredItems) > 0 {
		// Update fetchFrom to now since we have new items
		n.fetchFrom = time.Now()
	}

	return filteredItems, nil
}

func (n *NewsFeedManager) fetchRSSAndHandle() error {
	itemsToHandle, err := n.FetchRSS()
	if err != nil {
		n.logger.Error("error fetching feed", zap.Error(err))
		return err
	}

	for _, item := range itemsToHandle {
		err := n.handler.HandleFeedItemAndSend(item)
		if err != nil {
			n.logger.Error("error handling item", zap.Error(err))
			return err
		}
	}

	return nil
}

func (n *NewsFeedManager) IsPolling() bool {
	return n.polling
}

func (n *NewsFeedManager) StartPolling(ctx context.Context) {
	if n.polling {
		return
	}
	n.polling = true

	// Derive the given context, save the CancelFunc
	ctx, n.cancel = context.WithCancel(ctx)

	go func() {
		defer gocommon.LogOnPanic()

		// Initialize interval to 0 for immediate execution
		var interval time.Duration = 0

		for {
			select {
			case <-time.After(interval):
				// Immediate execution on first run, then set to regular interval
				interval = n.pollingInterval
				_ = n.fetchRSSAndHandle()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (n *NewsFeedManager) StopPolling() {
	if !n.polling {
		return
	}
	n.cancel()
	n.polling = false
}
