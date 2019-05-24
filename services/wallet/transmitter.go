package wallet

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/signal"
)

type publisher interface {
	Subscribe(interface{}) event.Subscription
}

type SignalsTransmitter struct {
	publisher

	wg   sync.WaitGroup
	quit chan struct{}
}

func (tmr *SignalsTransmitter) Start() error {
	if tmr.quit != nil {
		return errors.New("already running")
	}
	tmr.quit = make(chan struct{})
	events := make(chan Event, 10)
	sub := tmr.publisher.Subscribe(events)

	tmr.wg.Add(1)
	go func() {
		defer tmr.wg.Done()
		for {
			select {
			case <-tmr.quit:
				sub.Unsubscribe()
				return
			case err := <-sub.Err():
				// technically event.Feed cannot send an error to subscription.Err channel.
				// the only time we will get an event is when that channel is closed.
				if err != nil {
					log.Error("wallet signals transmitter failed with", "error", err)
				}
				return
			case event := <-events:
				signal.SendWalletEvent(event)
			}
		}
	}()
	return nil
}

func (tmr *SignalsTransmitter) Stop() {
	if tmr.quit == nil {
		return
	}
	close(tmr.quit)
	tmr.wg.Wait()
	tmr.quit = nil
}
