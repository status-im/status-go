package relay

import (
	"sync"

	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Subscription handles the subscrition to a particular pubsub topic
type Subscription struct {
	sync.RWMutex

	// C is channel used for receiving envelopes
	C chan *protocol.Envelope

	closed bool
	once   sync.Once
	quit   chan struct{}
}

// Unsubscribe will close a subscription from a pubsub topic. Will close the message channel
func (subs *Subscription) Unsubscribe() {
	subs.once.Do(func() {
		close(subs.quit)
	})

}

// IsClosed determine whether a Subscription is still open for receiving messages
func (subs *Subscription) IsClosed() bool {
	subs.RLock()
	defer subs.RUnlock()
	return subs.closed
}
