package shhext

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/services/shhext/mailservers"
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

// tracker responsible for processing events for envelopes that we are interested in
// and calling specified handler.
type tracker struct {
	w                      *whisper.Whisper
	handler                EnvelopeEventsHandler
	mailServerConfirmation bool

	mu      sync.Mutex
	cache   map[common.Hash]EnvelopeState
	batches map[common.Hash]map[common.Hash]struct{}

	mailPeers *mailservers.PeerStore

	requestsRegistry *RequestsRegistry

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

func (t *tracker) GetState(hash common.Hash) EnvelopeState {
	t.mu.Lock()
	defer t.mu.Unlock()
	state, exist := t.cache[hash]
	if !exist {
		return NotRegistered
	}
	return state
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
		whisper.EventEnvelopeSent:               t.handleEventEnvelopeSent,
		whisper.EventEnvelopeReceived:           t.handleEventEnvelopeReceived,
		whisper.EventEnvelopeExpired:            t.handleEventEnvelopeExpired,
		whisper.EventBatchAcknowledged:          t.handleAcknowledgedBatch,
		whisper.EventMailServerRequestSent:      t.handleRequestSent,
		whisper.EventMailServerRequestCompleted: t.handleEventMailServerRequestCompleted,
		whisper.EventMailServerRequestExpired:   t.handleEventMailServerRequestExpired,
	}

	if handler, ok := handlers[event.Event]; ok {
		fmt.Println("received event")
		handler(event)
	}
}

func (t *tracker) handleEventEnvelopeSent(event whisper.EnvelopeEvent) {
	if t.mailServerConfirmation {
		if !t.isMailserver(event.Peer) {
			return
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	state, ok := t.cache[event.Hash]
	// if we didn't send a message using extension - skip it
	// if message was already confirmed - skip it
	if !ok || state == EnvelopeSent {
		return
	}
	log.Debug("envelope is sent", "hash", event.Hash, "peer", event.Peer)
	if event.Batch != (common.Hash{}) {
		if _, ok := t.batches[event.Batch]; !ok {
			t.batches[event.Batch] = map[common.Hash]struct{}{}
		}
		t.batches[event.Batch][event.Hash] = struct{}{}
		log.Debug("waiting for a confirmation", "batch", event.Batch)
	} else {
		t.cache[event.Hash] = EnvelopeSent
		if t.handler != nil {
			t.handler.EnvelopeSent(event.Hash)
		}
	}
}

func (t *tracker) handleEventEnvelopeReceived(event whisper.EnvelopeEvent) {
	if t.mailServerConfirmation {
		if !t.isMailserver(event.Peer) {
			return
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	state, ok := t.cache[event.Hash]
	if !ok || state != EnvelopePosted {
		return
	}
	log.Debug("expected envelope received", "hash", event.Hash, "peer", event.Peer)
	delete(t.cache, event.Hash)
	if t.handler != nil {
		t.handler.EnvelopeSent(event.Hash)
	}
}

func (t *tracker) isMailserver(peer enode.ID) bool {
	return t.mailPeers.Exist(peer)
}

func (t *tracker) handleAcknowledgedBatch(event whisper.EnvelopeEvent) {
	if t.mailServerConfirmation {
		if !t.isMailserver(event.Peer) {
			return
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	envelopes, ok := t.batches[event.Batch]
	if !ok {
		log.Debug("batch is not found", "batch", event.Batch)
	}
	log.Debug("received a confirmation", "batch", event.Batch, "peer", event.Peer)
	for hash := range envelopes {
		state, ok := t.cache[hash]
		if !ok || state == EnvelopeSent {
			continue
		}
		t.cache[hash] = EnvelopeSent
		if t.handler != nil {
			t.handler.EnvelopeSent(hash)
		}
	}
	delete(t.batches, event.Batch)
}

func (t *tracker) handleEventEnvelopeExpired(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if state, ok := t.cache[event.Hash]; ok {
		delete(t.cache, event.Hash)
		if state == EnvelopeSent {
			return
		}
		log.Debug("envelope expired", "hash", event.Hash, "state", state)
		if t.handler != nil {
			t.handler.EnvelopeExpired(event.Hash)
		}
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
