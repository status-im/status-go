package connection

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

type Status struct {
	eventType       walletevent.EventType
	feed            *event.Feed
	isConnected     bool
	lastCheckedAt   int64
	isConnectedLock sync.RWMutex
}

func NewStatus(eventType walletevent.EventType, feed *event.Feed) *Status {
	return &Status{
		eventType:     eventType,
		feed:          feed,
		isConnected:   true,
		lastCheckedAt: time.Now().Unix(),
	}
}

func (c *Status) SetIsConnected(value bool) {
	c.isConnectedLock.Lock()
	defer c.isConnectedLock.Unlock()

	c.lastCheckedAt = time.Now().Unix()
	if value != c.isConnected {
		message := "down"
		if value {
			message = "up"
		}
		if c.feed != nil {
			c.feed.Send(walletevent.Event{
				Type:     c.eventType,
				Accounts: []common.Address{},
				Message:  message,
				At:       time.Now().Unix(),
			})
		}
	}
	c.isConnected = value
}

func (c *Status) IsConnected() bool {
	c.isConnectedLock.RLock()
	defer c.isConnectedLock.RUnlock()

	return c.isConnected
}
