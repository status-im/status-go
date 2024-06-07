package connection

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// Client expects a single event with all states
type StatusNotification map[string]State // id -> State

type StatusNotifier struct {
	statuses  *sync.Map // id -> Status
	eventType walletevent.EventType
	feed      *event.Feed
}

func NewStatusNotifier(statuses *sync.Map, eventType walletevent.EventType, feed *event.Feed) *StatusNotifier {
	n := StatusNotifier{
		statuses:  statuses,
		eventType: eventType,
		feed:      feed,
	}

	statuses.Range(func(_, value interface{}) bool {
		value.(*Status).SetStateChangeCb(n.notify)
		return true
	})

	return &n
}

func (n *StatusNotifier) notify(state State) {
	// state is ignored, as client expects all valid states in
	// a single event, so we fetch them from the map
	if n.feed != nil {
		statusMap := make(StatusNotification)
		n.statuses.Range(func(id, value interface{}) bool {
			state := value.(*Status).GetState()
			if state.Value == StateValueUnknown {
				return true
			}
			statusMap[id.(string)] = state
			return true
		})

		encodedMessage, err := json.Marshal(statusMap)
		if err != nil {
			return
		}

		n.feed.Send(walletevent.Event{
			Type:     n.eventType,
			Accounts: []common.Address{},
			Message:  string(encodedMessage),
			At:       time.Now().Unix(),
		})
	}
}
