package relay

import "github.com/waku-org/go-waku/waku/v2/protocol"

type Subscription struct {
	Unsubscribe func()
	Ch          <-chan *protocol.Envelope
}

func NoopSubscription() Subscription {
	ch := make(chan *protocol.Envelope)
	close(ch)
	return Subscription{
		Unsubscribe: func() {},
		Ch:          ch,
	}
}

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
