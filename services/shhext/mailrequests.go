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

// MailRequestMonitor is responsible for monitoring history request to mailservers.
type MailRequestMonitor struct {
	w       *whisper.Whisper
	handler EnvelopeEventsHandler

	mu    sync.Mutex
	cache map[common.Hash]EnvelopeState

	requestsRegistry *RequestsRegistry

	wg   sync.WaitGroup
	quit chan struct{}
}

// Start processing events.
func (m *MailRequestMonitor) Start() {
	m.quit = make(chan struct{})
	m.wg.Add(1)
	go func() {
		m.handleEnvelopeEvents()
		m.wg.Done()
	}()
}

// Stop process events.
func (m *MailRequestMonitor) Stop() {
	close(m.quit)
	m.wg.Wait()
}

func (m *MailRequestMonitor) GetState(hash common.Hash) EnvelopeState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exist := m.cache[hash]
	if !exist {
		return NotRegistered
	}
	return state
}

// handleEnvelopeEvents processes whisper envelope events
func (m *MailRequestMonitor) handleEnvelopeEvents() {
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
// confirmation handler or removes hash from MailRequestMonitor
func (m *MailRequestMonitor) handleEvent(event whisper.EnvelopeEvent) {
	handlers := map[whisper.EventType]func(whisper.EnvelopeEvent){
		whisper.EventMailServerRequestSent:      m.handleRequestSent,
		whisper.EventMailServerRequestCompleted: m.handleEventMailServerRequestCompleted,
		whisper.EventMailServerRequestExpired:   m.handleEventMailServerRequestExpired,
	}

	if handler, ok := handlers[event.Event]; ok {
		handler(event)
	}
}

func (m *MailRequestMonitor) handleRequestSent(event whisper.EnvelopeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[event.Hash] = MailServerRequestSent
}

func (m *MailRequestMonitor) handleEventMailServerRequestCompleted(event whisper.EnvelopeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestsRegistry.Unregister(event.Hash)
	state, ok := m.cache[event.Hash]
	if !ok || state != MailServerRequestSent {
		return
	}
	log.Debug("mailserver response received", "hash", event.Hash)
	delete(m.cache, event.Hash)
	if m.handler != nil {
		if resp, ok := event.Data.(*whisper.MailServerResponse); ok {
			m.handler.MailServerRequestCompleted(event.Hash, resp.LastEnvelopeHash, resp.Cursor, resp.Error)
		}
	}
}

func (m *MailRequestMonitor) handleEventMailServerRequestExpired(event whisper.EnvelopeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestsRegistry.Unregister(event.Hash)
	state, ok := m.cache[event.Hash]
	if !ok || state != MailServerRequestSent {
		return
	}
	log.Debug("mailserver response expired", "hash", event.Hash)
	delete(m.cache, event.Hash)
	if m.handler != nil {
		m.handler.MailServerRequestExpired(event.Hash)
	}
}
