package rest

import (
	"context"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

type Adder func(msg *protocol.Envelope)

type runnerService struct {
	broadcaster relay.Broadcaster
	sub         *relay.Subscription
	cancel      context.CancelFunc
	adder       Adder
}

func newRunnerService(broadcaster relay.Broadcaster, adder Adder) *runnerService {
	return &runnerService{
		broadcaster: broadcaster,
		adder:       adder,
	}
}

func (r *runnerService) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.sub = r.broadcaster.RegisterForAll(relay.WithBufferSize(relay.DefaultRelaySubscriptionBufferSize))
	for {
		select {
		case <-ctx.Done():
			return
		case envelope, ok := <-r.sub.Ch:
			if ok {
				r.adder(envelope)
			}
		}
	}
}

func (r *runnerService) Stop() {
	if r.cancel == nil {
		return
	}
	r.sub.Unsubscribe()
	r.cancel()
}
