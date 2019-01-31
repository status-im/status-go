package shhext

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	whisper "github.com/status-im/whisper/whisperv6"
)

// EnvelopeState in local tracker
type EnvelopeState int

const (
	// NotRegistered returned if asked hash wasn't registered in the tracker.
	NotRegistered EnvelopeState = -1
	// EnvelopePosted is set when envelope was added to a local whisper queue.
	EnvelopePosted EnvelopeState = iota
	// EnvelopeSent is set when envelope is sent to atleast one peer.
	EnvelopeSent
	// MailServerRequestSent is set when p2p request is sent to the mailserver
	MailServerRequestSent
)

// TODO rename tracker to HistoryRequestMonitor to watch only history requests.
// tracker responsible for processing events for envelopes that we are interested in
// and calling specified handler.
type tracker struct {
	w       *whisper.Whisper
	handler EnvelopeEventsHandler

	mu    sync.Mutex
	cache map[common.Hash]EnvelopeState

	requestsRegistry *RequestsRegistry

	wg   sync.WaitGroup
	quit chan struct{}
}

func (t *tracker) GetState(hash common.Hash) EnvelopeState {
	t.mu.Lock()
	defer t.mu.Unlock()
	state, exist := t.cache[hash]
	if !exist {
		return NotRegistered
	}
	return state
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
	handlers := map[whisper.EventType]func(whisper.EnvelopeEvent){
		whisper.EventMailServerRequestSent:      t.handleRequestSent,
		whisper.EventMailServerRequestCompleted: t.handleEventMailServerRequestCompleted,
		whisper.EventMailServerRequestExpired:   t.handleEventMailServerRequestExpired,
	}

	if handler, ok := handlers[event.Event]; ok {
		handler(event)
	}
}

func (t *tracker) handleRequestSent(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache[event.Hash] = MailServerRequestSent
}

func (t *tracker) handleEventMailServerRequestCompleted(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.requestsRegistry.Unregister(event.Hash)
	state, ok := t.cache[event.Hash]
	if !ok || state != MailServerRequestSent {
		return
	}
	log.Debug("mailserver response received", "hash", event.Hash)
	delete(t.cache, event.Hash)
	if t.handler != nil {
		if resp, ok := event.Data.(*whisper.MailServerResponse); ok {
			t.handler.MailServerRequestCompleted(event.Hash, resp.LastEnvelopeHash, resp.Cursor, resp.Error)
		}
	}
}

func (t *tracker) handleEventMailServerRequestExpired(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.requestsRegistry.Unregister(event.Hash)
	state, ok := t.cache[event.Hash]
	if !ok || state != MailServerRequestSent {
		return
	}
	log.Debug("mailserver response expired", "hash", event.Hash)
	delete(t.cache, event.Hash)
	if t.handler != nil {
		t.handler.MailServerRequestExpired(event.Hash)
	}
}
