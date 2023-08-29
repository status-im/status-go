package relay

import "github.com/waku-org/go-waku/waku/v2/protocol"

// Subscription handles the details of a particular Topic subscription. There may be many subscriptions for a given topic.
type Subscription struct {
	Unsubscribe func()
	Ch          <-chan *protocol.Envelope
}

// NoopSubscription creates a noop subscription that will not receive any envelope
func NoopSubscription() Subscription {
	ch := make(chan *protocol.Envelope)
	close(ch)
	return Subscription{
		Unsubscribe: func() {},
		Ch:          ch,
	}
}

// ArraySubscription creates a subscription for a list of envelopes
func ArraySubscription(msgs []*protocol.Envelope) Subscription {
	ch := make(chan *protocol.Envelope, len(msgs))
	for _, msg := range msgs {
		ch <- msg
	}
	close(ch)
	return Subscription{
		Unsubscribe: func() {},
		Ch:          ch,
	}
}
