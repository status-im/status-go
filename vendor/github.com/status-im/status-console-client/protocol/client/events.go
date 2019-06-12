package client

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-console-client/protocol/v1"
)

type DatabaseWithEvents struct {
	Database
	feed *event.Feed
}

func NewDatabaseWithEvents(db Database, feed *event.Feed) DatabaseWithEvents {
	return DatabaseWithEvents{Database: db, feed: feed}
}

func (db DatabaseWithEvents) SaveMessages(c Contact, msgs []*protocol.Message) (int64, error) {
	rowid, err := db.Database.SaveMessages(c, msgs)
	if err != nil {
		return rowid, err
	}
	for _, m := range msgs {
		ev := messageEvent{
			baseEvent: baseEvent{
				Contact: c,
				Type:    EventTypeMessage,
			},
			Message: m,
		}
		db.feed.Send(Event{ev})
	}
	return rowid, err
}
