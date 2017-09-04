package whisperv5

import (
	"time"
)

const (
	// pollingFreq dictates the polling interval
	pollingInterval = 100 * time.Millisecond
)

// Listener represents a whisper subscriber.
type Listener struct {
	host     *Server       // Whisper Server
	filterID string        // Filter ID
	filter   *Filter       // FOR NOW, it's being used to fetch messages
	handler  HandlerFunc   // handles the received messages
	doneChan chan struct{} // Channel used for graceful exit
}

// newListener returns a new whisper subscriber
func newListener(host *Server, filter *Filter, handler HandlerFunc) *Listener {
	return &Listener{
		host:     host,
		filter:   filter,
		handler:  handler,
		doneChan: make(chan struct{}),
	}
}

// listen retrieves whisper protocol messages
func (l *Listener) listen() {
	polling := time.NewTicker(pollingInterval)

	filterID, err := l.host.Subscribe(l.filter)
	l.filterID = filterID
	if err != nil {
		// Error
		return
	}
	defer l.host.Unsubscribe(l.filterID)

	for {
		select {
		case <-polling.C:
			messages := l.filter.Retrieve()
			for _, req := range messages {
				resp := &MessageParams{}
				l.handler.ServeWhisper(resp, req)
				// response delivery is delegated due to proof of work
				l.host.messageQueue <- resp
			}
		case <-l.doneChan:
			return
		}
	}
}

// Start initiates the listener activity
func (l *Listener) start() {
	go l.listen()
}

// Stop terminates the listener activity
func (l *Listener) stop() {
	close(l.doneChan)
}
