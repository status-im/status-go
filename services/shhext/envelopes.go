package shhext

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/services/shhext/mailservers"
	whisper "github.com/status-im/whisper/whisperv6"
)

func messageID(message whisper.NewMessage) common.Hash {
	hash := fnv.New32()
	_, _ = hash.Write(message.Payload)
	_, _ = hash.Write(message.Topic[:])
	return common.BytesToHash(hash.Sum(nil))
}

// NewEnvelopesMonitor returns a pointer to an instance of the EnvelopesMonitor.
func NewEnvelopesMonitor(w *whisper.Whisper, handler EnvelopeEventsHandler, mailServerConfirmation bool, mailPeers *mailservers.PeerStore, maxAttempts int) *EnvelopesMonitor {
	return &EnvelopesMonitor{
		w:                      w,
		whisperAPI:             whisper.NewPublicWhisperAPI(w),
		handler:                handler,
		mailServerConfirmation: mailServerConfirmation,
		mailPeers:              mailPeers,
		maxAttempts:            maxAttempts,

		// key is envelope hash (event.Hash)
		envelopes: map[common.Hash]EnvelopeState{},
		messages:  map[common.Hash]whisper.NewMessage{},
		attempts:  map[common.Hash]int{},

		// key is messageID
		messageToEnvelope: map[common.Hash]common.Hash{},

		// key is hash of the batch (event.Batch)
		batches: map[common.Hash]map[common.Hash]struct{}{},
	}
}

// EnvelopesMonitor is responsible for monitoring whisper envelopes state.
type EnvelopesMonitor struct {
	w                      *whisper.Whisper
	whisperAPI             *whisper.PublicWhisperAPI
	handler                EnvelopeEventsHandler
	mailServerConfirmation bool
	maxAttempts            int

	mu        sync.Mutex
	envelopes map[common.Hash]EnvelopeState
	batches   map[common.Hash]map[common.Hash]struct{}

	messageToEnvelope map[common.Hash]common.Hash
	messages          map[common.Hash]whisper.NewMessage
	attempts          map[common.Hash]int

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
func (m *EnvelopesMonitor) Add(envelopeHash common.Hash, message whisper.NewMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.envelopes[envelopeHash] = EnvelopePosted
	m.messages[envelopeHash] = message
	m.attempts[envelopeHash] = 1
	m.messageToEnvelope[messageID(message)] = envelopeHash
}

func (m *EnvelopesMonitor) GetState(hash common.Hash) EnvelopeState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exist := m.envelopes[hash]
	if !exist {
		return NotRegistered
	}
	return state
}

func (m *EnvelopesMonitor) GetMessageState(mID common.Hash) EnvelopeState {
	m.mu.Lock()
	defer m.mu.Unlock()
	envelope, exist := m.messageToEnvelope[mID]
	if !exist {
		return NotRegistered
	}
	state, exist := m.envelopes[envelope]
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

	state, ok := m.envelopes[event.Hash]
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
		m.envelopes[event.Hash] = EnvelopeSent
		if m.handler != nil {
			m.handler.EnvelopeSent(messageID(m.messages[event.Hash]))
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
	envelopeErrors, ok := event.Data.([]whisper.EnvelopeError)
	if !ok {
		log.Error("received unexpected data in the the confirmation event", "batch", event.Batch)
	}
	failedEnvelopes := map[common.Hash]struct{}{}
	for i := range envelopeErrors {
		envelopeError := envelopeErrors[i]
		_, exist := m.envelopes[envelopeError.Hash]
		if exist {
			log.Warn("envelope that was posted by us is discarded", "hash", envelopeError.Hash, "peer", event.Peer, "error", envelopeError.Description)
			var err error
			switch envelopeError.Code {
			case whisper.EnvelopeTimeNotSynced:
				err = errors.New("envelope wasn't delivered due to time sync issues")
			}
			m.handleEnvelopeFailure(envelopeError.Hash, err)
		}
		failedEnvelopes[envelopeError.Hash] = struct{}{}
	}

	for hash := range envelopes {
		if _, exist := failedEnvelopes[hash]; exist {
			continue
		}
		state, ok := m.envelopes[hash]
		if !ok || state == EnvelopeSent {
			continue
		}
		m.envelopes[hash] = EnvelopeSent
		if m.handler != nil {
			m.handler.EnvelopeSent(messageID(m.messages[hash]))
		}
	}
	delete(m.batches, event.Batch)
}

func (m *EnvelopesMonitor) handleEventEnvelopeExpired(event whisper.EnvelopeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handleEnvelopeFailure(event.Hash, errors.New("envelope expired due to connectivity issues"))
}

// handleEnvelopeFailure is a common code path for processing envelopes failures. not thread safe, lock
// must be used on a higher level.
func (m *EnvelopesMonitor) handleEnvelopeFailure(hash common.Hash, err error) {
	if state, ok := m.envelopes[hash]; ok {
		message, exist := m.messages[hash]
		if !exist {
			log.Error("message was deleted erroneously", "envelope hash", hash)
		}
		mID := messageID(message)
		attempt := m.attempts[hash]
		m.clearMessageState(hash)
		if state == EnvelopeSent {
			return
		}
		if attempt < m.maxAttempts {
			log.Debug("retrying to send a message", "message id", mID, "attempt", attempt+1)
			hex, err := m.whisperAPI.Post(context.TODO(), message)
			if err != nil {
				log.Error("failed to retry sending message", "message id", mID, "attempt", attempt+1)
			}
			envelopeID := common.BytesToHash(hex)
			m.messageToEnvelope[mID] = envelopeID
			m.envelopes[envelopeID] = EnvelopePosted
			m.messages[envelopeID] = message
			m.attempts[envelopeID] = attempt + 1
		} else {
			log.Debug("envelope expired", "hash", hash, "state", state)
			if m.handler != nil {
				m.handler.EnvelopeExpired(mID, err)
			}
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
	state, ok := m.envelopes[event.Hash]
	if !ok || state != EnvelopePosted {
		return
	}
	log.Debug("expected envelope received", "hash", event.Hash, "peer", event.Peer)
	m.envelopes[event.Hash] = EnvelopeSent
	if m.handler != nil {
		m.handler.EnvelopeSent(messageID(m.messages[event.Hash]))
	}
}

// clearMessageState removes all message and envelope state.
// not thread-safe, should be protected on a higher level.
func (m *EnvelopesMonitor) clearMessageState(envelopeID common.Hash) {
	delete(m.envelopes, envelopeID)
	mID := messageID(m.messages[envelopeID])
	delete(m.messageToEnvelope, mID)
	delete(m.messages, envelopeID)
	delete(m.attempts, envelopeID)
}
