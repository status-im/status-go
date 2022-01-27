package local_pairing

import (
	"time"

	"github.com/status-im/status-go/signal"
)

func NewAPI() *API {
	return &API{}
}

type API struct {}

func (a *API) StartSendingServer(password string) (string, error){

	go func() {
		time.Sleep(time.Second)
		signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess})

		time.Sleep(time.Second)
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess})

		time.Sleep(time.Second)
		signal.SendLocalPairingEvent(Event{Type: EventSuccess})
	}()

	return password, nil
}