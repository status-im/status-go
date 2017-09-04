package whisperv5

import (
	"crypto/ecdsa"
	"runtime"
	"sync"
)

// Server defines parameters for a running Whisper Server
type Server struct {
	*Whisper // whisper protocol

	listenersMu sync.Mutex  // Mutex to sync the active listener set and the listening flag
	listening   bool        // listening flag specifies the server status
	listeners   []*Listener // Set of currently active listeners

	messageQueue chan *MessageParams // Message queue for responses

	doneChan chan struct{} // Channel used for graceful exit
}

// NewWhisperServer returns a new whisper server instance
func NewWhisperServer(whisper *Whisper) *Server {
	return &Server{
		Whisper:      whisper,
		messageQueue: make(chan *MessageParams, messageQueueLimit),
		doneChan:     make(chan struct{}),
	}
}

// Handler
type Handler interface {
	ServeWhisper(*MessageParams, *ReceivedMessage)
}

// HandlerFunc type is an adapter to allow the use of
// ordinary functions as Whisper Handlers
type HandlerFunc func(*MessageParams, *ReceivedMessage)

// ServeWhisper calls f(w, r).
func (f HandlerFunc) ServeWhisper(resp *MessageParams, req *ReceivedMessage) {
	f(resp, req)
}

// HandleFunc registers the handler function for a given topic and key, in the specified server
func (s *Server) HandleFunc(topic string, key interface{}, handler func(resp *MessageParams, msg *ReceivedMessage)) error {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()

	rawTopic := TopicFromString(topic)

	// create filter & subscribe
	filter := &Filter{
		Topics: [][]byte{rawTopic},
	}
	switch key.(type) {
	case *ecdsa.PrivateKey:
		filter.KeyAsym = key.(*ecdsa.PrivateKey)
	case []byte:
		filter.KeySym = key.([]byte)
	default:
		// TODO Error
	}

	// create listener
	listener := newListener(s, filter, HandlerFunc(handler))
	s.listeners = append(s.listeners, listener)
	if s.listening {
		listener.start()
	}

	return nil
}

// ListenAndServe
func (s *Server) ListenAndServe() {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()

	numCPU := runtime.NumCPU()
	for i := 0; i < numCPU; i++ {
		go s.processQueue()
	}
	s.listening = true
	for _, listener := range s.listeners {
		listener.start()
	}
}

// Stop terminantes the server activities
func (s *Server) Stop() {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()
	for _, listener := range s.listeners {
		listener.stop()
	}
	s.listening = false
	close(s.doneChan)
}

// processQueue processes the response queue
func (s *Server) processQueue() {
	for {
		select {
		case params := <-s.messageQueue:
			if err := s.publish(params); err != nil {
				// log error
			}
		case <-s.doneChan:
			return
		}
	}
}

// publish creates the response message, an envelope with the
// message and delivers the envelope via whisper protocol.
func (s *Server) publish(params *MessageParams) error {
	response, err := NewSentMessage(params)
	if err != nil {
		return err
	}
	env, err := response.Wrap(params)
	if err != nil {
		return err
	}
	if err := s.Send(env); err != nil {
		return err
	}
	return nil
}
