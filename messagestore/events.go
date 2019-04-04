package messagestore

import (
	"time"

	"github.com/ethereum/go-ethereum/event"
	whisper "github.com/status-im/whisper/whisperv6"
)

// EventHistoryPersisted used to notify about newly received timestamp for a particular topic.
type EventHistoryPersisted struct {
	Topic     whisper.TopicType
	Timestamp time.Time
}

// NewStoreWithHistoryEvents returns instance of the StoreWithHistoryEvents.
func NewStoreWithHistoryEvents(store SQLMessageStore) *StoreWithHistoryEvents {
	return &StoreWithHistoryEvents{SQLMessageStore: store}
}

// StoreWithHistoryEvents notifies when history message got persisted.
type StoreWithHistoryEvents struct {
	SQLMessageStore

	feed event.Feed
}

// Add notifies subscribers if message got persisted succesfully.
func (store *StoreWithHistoryEvents) Add(msg *whisper.ReceivedMessage) error {
	err := store.SQLMessageStore.Add(msg)
	if err == nil {
		store.feed.Send(EventHistoryPersisted{
			Topic:     msg.Topic,
			Timestamp: time.Unix(int64(msg.Sent), 0),
		})
	}
	return err
}

// Subscribe allows to subscribe for history events.
func (store *StoreWithHistoryEvents) Subscribe(events chan<- EventHistoryPersisted) event.Subscription {
	return store.feed.Subscribe(events)
}
