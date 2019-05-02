package subscriptions

import "github.com/status-im/status-go/signal"

type filterSignal struct {
	filterID string
}

func newFilterSignal(filterID string) *filterSignal {
	return &filterSignal{filterID}
}

func (s *filterSignal) SendError(err error) {
	signal.SendSubscriptionErrorEvent(s.filterID, 10, err.Error())
}

func SendData(data []interface{}) {
	signal.SendSubscriptionDataEvent(s.filterID, data)
}
