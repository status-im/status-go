package mock_common

import (
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

type FeedSubscription struct {
	events chan walletevent.Event
	feed   *event.Feed
	done   chan struct{}
}

func NewFeedSubscription(feed *event.Feed) *FeedSubscription {
	events := make(chan walletevent.Event, 100)
	done := make(chan struct{})

	subscription := feed.Subscribe(events)

	go func() {
		defer common.LogOnPanic()
		<-done
		subscription.Unsubscribe()
		close(events)
	}()

	return &FeedSubscription{events: events, feed: feed, done: done}
}

func (f *FeedSubscription) WaitForEvent(timeout time.Duration) (walletevent.Event, bool) {
	select {
	case evt := <-f.events:
		return evt, true
	case <-time.After(timeout):
		return walletevent.Event{}, false
	}
}

func (f *FeedSubscription) GetFeed() *event.Feed {
	return f.feed
}

func (f *FeedSubscription) Close() {
	close(f.done)
}
