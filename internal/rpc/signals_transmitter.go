package rpc

import (
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/signal"
)

type SignalsTransmitter struct {
	publisher *pubsub.Publisher

	quit chan struct{}
}

func NewSignalsTransmitter(publisher *pubsub.Publisher) *SignalsTransmitter {
	return &SignalsTransmitter{
		publisher: publisher,
	}
}

func (tmr *SignalsTransmitter) Start() error {
	if tmr.quit != nil {
		// already running, nothing to do
		return nil
	}
	tmr.quit = make(chan struct{})
	ch, unsubFn := pubsub.Subscribe[EventBlockchainHealthChanged](tmr.publisher, 10)

	go func() {
		defer gocommon.LogOnPanic()
		defer unsubFn()
		for {
			select {
			case <-tmr.quit:
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				signal.SendNetworksBlockchainHealthChanged(event.FullStatus)
			}
		}
	}()
	return nil
}

func (tmr *SignalsTransmitter) Stop() {
	if tmr.quit != nil {
		close(tmr.quit)
		tmr.quit = nil
	}
}
