package shhext

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

// EnvelopeState in local tracker
type EnvelopeState int

const (
	// EnvelopePosted is set when envelope was added to a local whisper queue.
	EnvelopePosted EnvelopeState = iota
	// EnvelopeSent is set when envelope is sent to atleast one peer.
	EnvelopeSent
)

// ConfirmationHandler used as a callback for confirming that envelopes were sent.
type ConfirmationHandler func(common.Hash)

// Service is a service that provides some additional Whisper API.
type Service struct {
	w       *whisper.Whisper
	tracker *tracker
}

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// New returns a new Service.
func New(w *whisper.Whisper, handler ConfirmationHandler) *Service {
	track := &tracker{
		w:       w,
		handler: handler,
		cache:   map[common.Hash]EnvelopeState{},
	}
	return &Service{
		w:       w,
		tracker: track,
	}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "shhext",
			Version:   "1.0",
			Service:   NewPublicAPI(s.w, s.tracker),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Start(server *p2p.Server) error {
	s.tracker.Start()
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Stop() error {
	s.tracker.Stop()
	return nil
}

// tracker responsible for processing events for envelopes that we are interested in
// and calling specified handler.
type tracker struct {
	w       *whisper.Whisper
	handler ConfirmationHandler

	mu    sync.Mutex
	cache map[common.Hash]EnvelopeState

	wg   sync.WaitGroup
	quit chan struct{}
}

// Start processing events.
func (t *tracker) Start() {
	t.quit = make(chan struct{})
	t.wg.Add(1)
	go func() {
		t.handleEnvelopeEvents()
		t.wg.Done()
	}()
}

// Stop process events.
func (t *tracker) Stop() {
	close(t.quit)
	t.wg.Wait()
}

// Add hash to a tracker.
func (t *tracker) Add(hash common.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache[hash] = EnvelopePosted
}

// handleEnvelopeEvents processes whisper envelope events
func (t *tracker) handleEnvelopeEvents() {
	events := make(chan whisper.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	sub := t.w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	for {
		select {
		case <-t.quit:
			return
		case event := <-events:
			t.handleEvent(event)
		}
	}
}

// handleEvent based on type of the event either triggers
// confirmation handler or removes hash from tracker
func (t *tracker) handleEvent(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch event.Event {
	case whisper.EventEnvelopeSent:
		state, ok := t.cache[event.Hash]
		// if we didn't send a message using extension - skip it
		// if message was already confirmed - skip it
		if !ok || state == EnvelopeSent {
			return
		}
		if t.handler != nil {
			log.Debug("envelope is sent", "hash", event.Hash, "peer", event.Peer)
			t.handler(event.Hash)
			t.cache[event.Hash] = EnvelopeSent
		}
	case whisper.EventEnvelopeExpired:
		if _, ok := t.cache[event.Hash]; ok {
			log.Debug("envelope expired", "hash", event.Hash)
			delete(t.cache, event.Hash)
		}
	}
}
