package sdk

import (
	"time"
)

// MsgHandler is a callback function that processes messages delivered to
// asynchronous subscribers.
type MsgHandler func(msg *Msg)

// Subscription is a polling helper for a specific channel
type Subscription struct {
	unsubscribe chan bool
	channel     *Channel
}

// Subscribe polls on specific channel topic and executes given function if
// any message is received
func (s *Subscription) Subscribe(channel *Channel, fn MsgHandler) {
	s.channel = channel
	for {
		select {
		case <-s.unsubscribe:
			return
		default:
			if msg := channel.pollMessages(); msg != nil {
				fn(msg)
			}
		}
		// TODO(adriacidre) : move this period to configuration
		time.Sleep(time.Second * 3)
	}
}

// Unsubscribe stops polling on the current subscription channel
func (s *Subscription) Unsubscribe() {
	s.unsubscribe <- true
	s.channel.removeSubscription(s)
}
