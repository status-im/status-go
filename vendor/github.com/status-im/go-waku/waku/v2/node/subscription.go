package node

import (
	"sync"

	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Subscription to a pubsub topic
type Subscription struct {
	// Channel for receiving messages
	C chan *protocol.Envelope

	closed bool
	mutex  sync.Mutex
	quit   chan struct{}
}

// Unsubscribe from a pubsub topic. Will close the message channel
func (subs *Subscription) Unsubscribe() {
	if !subs.closed {
		close(subs.quit)
	}
}

// Determine whether a Subscription is open or not
func (subs *Subscription) IsClosed() bool {
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	return subs.closed
}
