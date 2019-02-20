package shhext

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/services/shhext/mailservers"
	whisper "github.com/status-im/whisper/whisperv6"
)

// EnvelopesMonitor is responsible for monitoring whisper envelopes state.
type EnvelopesMonitor struct {
	w                      *whisper.Whisper
	handler                EnvelopeEventsHandler
	mailServerConfirmation bool

	mu      sync.Mutex
	cache   map[common.Hash]EnvelopeState
	batches map[common.Hash]map[common.Hash]struct{}

	mailPeers *mailservers.PeerStore

	wg   sync.WaitGroup
	quit chan struct{}
}

// Start processing events.
func (m *EnvelopesMonitor) Start() {
	m.quit = make(chan struct{})
	m.wg.Add(1)
	go func() {
		m.handleEnvelopeEvents()
		m.wg.Done()
	}()
}

// Stop process events.
func (m *EnvelopesMonitor) Stop() {
	close(m.quit)
	m.wg.Wait()
}

// Add hash to a tracker.
func (m *EnvelopesMonitor) Add(hash common.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[hash] = EnvelopePosted
}

func (m *EnvelopesMonitor) GetState(hash common.Hash) EnvelopeState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exist := m.cache[hash]
	if !exist {
		return NotRegistered
	}
	return state
}

// handleEnvelopeEvents processes whisper envelope events
func (m *EnvelopesMonitor) handleEnvelopeEvents() {
	events := make(chan whisper.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	sub := m.w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	for {
		select {
		case <-m.quit:
			return
		case event := <-events:
			m.handleEvent(event)
		}
	}
}

// handleEvent based on type of the event either triggers
// confirmation handler or removes hash from tracker
func (m *EnvelopesMonitor) handleEvent(event whisper.EnvelopeEvent) {
	handlers := map[whisper.EventType]func(whisper.EnvelopeEvent){
		whisper.EventEnvelopeSent:      m.handleEventEnvelopeSent,
		whisper.EventEnvelopeExpired:   m.handleEventEnvelopeExpired,
		whisper.EventBatchAcknowledged: m.handleAcknowledgedBatch,
		whisper.EventEnvelopeReceived:  m.handleEventEnvelopeReceived,
	}
	if handler, ok := handlers[event.Event]; ok {
		handler(event)
	}
}

func (m *EnvelopesMonitor) handleEventEnvelopeSent(event whisper.EnvelopeEvent) {
	if m.mailServerConfirmation {
		if !m.isMailserver(event.Peer) {
			return
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.cache[event.Hash]
	// if we didn't send a message using extension - skip it
	// if message was already confirmed - skip it
	if !ok || state == EnvelopeSent {
		return
	}
	log.Debug("envelope is sent", "hash", event.Hash, "peer", event.Peer)
	if event.Batch != (common.Hash{}) {
		if _, ok := m.batches[event.Batch]; !ok {
			m.batches[event.Batch] = map[common.Hash]struct{}{}
		}
		m.batches[event.Batch][event.Hash] = struct{}{}
		log.Debug("waiting for a confirmation", "batch", event.Batch)
	} else {
		m.cache[event.Hash] = EnvelopeSent
		if m.handler != nil {
			m.handler.EnvelopeSent(event.Hash)
		}
	}
}

func (m *EnvelopesMonitor) isMailserver(peer enode.ID) bool {
	return m.mailPeers.Exist(peer)
}

func (m *EnvelopesMonitor) handleAcknowledgedBatch(event whisper.EnvelopeEvent) {
	if m.mailServerConfirmation {
		if !m.isMailserver(event.Peer) {
			return
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	envelopes, ok := m.batches[event.Batch]
	if !ok {
		log.Debug("batch is not found", "batch", event.Batch)
	}
	log.Debug("received a confirmation", "batch", event.Batch, "peer", event.Peer)
	for hash := range envelopes {
		state, ok := m.cache[hash]
		if !ok || state == EnvelopeSent {
			continue
		}
		m.cache[hash] = EnvelopeSent
		if m.handler != nil {
			m.handler.EnvelopeSent(hash)
		}
	}
	delete(m.batches, event.Batch)
}

func (m *EnvelopesMonitor) handleEventEnvelopeExpired(event whisper.EnvelopeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, ok := m.cache[event.Hash]; ok {
		delete(m.cache, event.Hash)
		if state == EnvelopeSent {
			return
		}
		log.Debug("envelope expired", "hash", event.Hash, "state", state)
		if m.handler != nil {
			m.handler.EnvelopeExpired(event.Hash)
		}
	}
}

func (m *EnvelopesMonitor) handleEventEnvelopeReceived(event whisper.EnvelopeEvent) {
	if m.mailServerConfirmation {
		if !m.isMailserver(event.Peer) {
			return
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.cache[event.Hash]
	if !ok || state != EnvelopePosted {
		return
	}
	log.Debug("expected envelope received", "hash", event.Hash, "peer", event.Peer)
	delete(m.cache, event.Hash)
	if m.handler != nil {
		m.handler.EnvelopeSent(event.Hash)
	}
}
