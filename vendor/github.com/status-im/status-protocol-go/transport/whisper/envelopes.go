package whisper

import (
	"context"
	"errors"
	"sync"

	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	statusproto "github.com/status-im/status-protocol-go/types"
	"go.uber.org/zap"
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
)

type EnvelopesMonitorConfig struct {
	EnvelopeEventsHandler          EnvelopeEventsHandler
	MaxAttempts                    int
	MailserverConfirmationsEnabled bool
	IsMailserver                   func(whispertypes.EnodeID) bool
	Logger                         *zap.Logger
}

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent([][]byte)
	EnvelopeExpired([][]byte, error)
	MailServerRequestCompleted(statusproto.Hash, statusproto.Hash, []byte, error)
	MailServerRequestExpired(statusproto.Hash)
}

// NewEnvelopesMonitor returns a pointer to an instance of the EnvelopesMonitor.
func NewEnvelopesMonitor(w whispertypes.Whisper, config EnvelopesMonitorConfig) *EnvelopesMonitor {
	logger := config.Logger

	if logger == nil {
		logger = zap.NewNop()
	}

	var whisperAPI whispertypes.PublicWhisperAPI
	if w != nil {
		whisperAPI = w.PublicWhisperAPI()
	}

	return &EnvelopesMonitor{
		w:                      w,
		whisperAPI:             whisperAPI,
		handler:                config.EnvelopeEventsHandler,
		mailServerConfirmation: config.MailserverConfirmationsEnabled,
		maxAttempts:            config.MaxAttempts,
		isMailserver:           config.IsMailserver,
		logger:                 logger.With(zap.Namespace("EnvelopesMonitor")),

		// key is envelope hash (event.Hash)
		envelopes:   map[statusproto.Hash]EnvelopeState{},
		messages:    map[statusproto.Hash]*whispertypes.NewMessage{},
		attempts:    map[statusproto.Hash]int{},
		identifiers: make(map[statusproto.Hash][][]byte),

		// key is hash of the batch (event.Batch)
		batches: map[statusproto.Hash]map[statusproto.Hash]struct{}{},
	}
}

// EnvelopesMonitor is responsible for monitoring whisper envelopes state.
type EnvelopesMonitor struct {
	w                      whispertypes.Whisper
	whisperAPI             whispertypes.PublicWhisperAPI
	handler                EnvelopeEventsHandler
	mailServerConfirmation bool
	maxAttempts            int

	mu        sync.Mutex
	envelopes map[statusproto.Hash]EnvelopeState
	batches   map[statusproto.Hash]map[statusproto.Hash]struct{}

	messages    map[statusproto.Hash]*whispertypes.NewMessage
	attempts    map[statusproto.Hash]int
	identifiers map[statusproto.Hash][][]byte

	wg           sync.WaitGroup
	quit         chan struct{}
	isMailserver func(peer whispertypes.EnodeID) bool

	logger *zap.Logger
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
func (m *EnvelopesMonitor) Add(identifiers [][]byte, envelopeHash statusproto.Hash, message whispertypes.NewMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.envelopes[envelopeHash] = EnvelopePosted
	m.identifiers[envelopeHash] = identifiers
	m.messages[envelopeHash] = &message
	m.attempts[envelopeHash] = 1
}

func (m *EnvelopesMonitor) GetState(hash statusproto.Hash) EnvelopeState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exist := m.envelopes[hash]
	if !exist {
		return NotRegistered
	}
	return state
}

// handleEnvelopeEvents processes whisper envelope events
func (m *EnvelopesMonitor) handleEnvelopeEvents() {
	events := make(chan whispertypes.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	sub := m.w.SubscribeEnvelopeEvents(events)
	defer func() {
		close(events)
		sub.Unsubscribe()
	}()
	for {
		select {
		case <-m.quit:
			close(events)
			return
		case event := <-events:
			m.handleEvent(event)
		}
	}
}

// handleEvent based on type of the event either triggers
// confirmation handler or removes hash from tracker
func (m *EnvelopesMonitor) handleEvent(event whispertypes.EnvelopeEvent) {
	handlers := map[whispertypes.EventType]func(whispertypes.EnvelopeEvent){
		whispertypes.EventEnvelopeSent:      m.handleEventEnvelopeSent,
		whispertypes.EventEnvelopeExpired:   m.handleEventEnvelopeExpired,
		whispertypes.EventBatchAcknowledged: m.handleAcknowledgedBatch,
		whispertypes.EventEnvelopeReceived:  m.handleEventEnvelopeReceived,
	}
	if handler, ok := handlers[event.Event]; ok {
		handler(event)
	}
}

func (m *EnvelopesMonitor) handleEventEnvelopeSent(event whispertypes.EnvelopeEvent) {
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
	m.logger.Debug("envelope is sent", zap.String("hash", event.Hash.String()), zap.String("peer", event.Peer.String()))
	if event.Batch != (statusproto.Hash{}) {
		if _, ok := m.batches[event.Batch]; !ok {
			m.batches[event.Batch] = map[statusproto.Hash]struct{}{}
		}
		m.batches[event.Batch][event.Hash] = struct{}{}
		m.logger.Debug("waiting for a confirmation", zap.String("batch", event.Batch.String()))
	} else {
		m.envelopes[event.Hash] = EnvelopeSent
		if m.handler != nil {
			m.handler.EnvelopeSent(m.identifiers[event.Hash])
		}
	}
}

func (m *EnvelopesMonitor) handleAcknowledgedBatch(event whispertypes.EnvelopeEvent) {
	if m.mailServerConfirmation {
		if !m.isMailserver(event.Peer) {
			return
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	envelopes, ok := m.batches[event.Batch]
	if !ok {
		m.logger.Debug("batch is not found", zap.String("batch", event.Batch.String()))
	}
	m.logger.Debug("received a confirmation", zap.String("batch", event.Batch.String()), zap.String("peer", event.Peer.String()))
	envelopeErrors, ok := event.Data.([]whispertypes.EnvelopeError)
	if event.Data != nil && !ok {
		m.logger.Error("received unexpected data in the the confirmation event", zap.String("batch", event.Batch.String()))
	}
	failedEnvelopes := map[statusproto.Hash]struct{}{}
	for i := range envelopeErrors {
		envelopeError := envelopeErrors[i]
		_, exist := m.envelopes[envelopeError.Hash]
		if exist {
			m.logger.Warn("envelope that was posted by us is discarded", zap.String("hash", envelopeError.Hash.String()), zap.String("peer", event.Peer.String()), zap.String("error", envelopeError.Description))
			var err error
			switch envelopeError.Code {
			case whispertypes.EnvelopeTimeNotSynced:
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
			m.handler.EnvelopeSent(m.identifiers[hash])
		}
	}
	delete(m.batches, event.Batch)
}

func (m *EnvelopesMonitor) handleEventEnvelopeExpired(event whispertypes.EnvelopeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handleEnvelopeFailure(event.Hash, errors.New("envelope expired due to connectivity issues"))
}

// handleEnvelopeFailure is a common code path for processing envelopes failures. not thread safe, lock
// must be used on a higher level.
func (m *EnvelopesMonitor) handleEnvelopeFailure(hash statusproto.Hash, err error) {
	if state, ok := m.envelopes[hash]; ok {
		message, exist := m.messages[hash]
		if !exist {
			m.logger.Error("message was deleted erroneously", zap.String("envelope hash", hash.String()))
		}
		attempt := m.attempts[hash]
		identifiers := m.identifiers[hash]
		m.clearMessageState(hash)
		if state == EnvelopeSent {
			return
		}
		if attempt < m.maxAttempts {
			m.logger.Debug("retrying to send a message", zap.String("hash", hash.String()), zap.Int("attempt", attempt+1))
			hex, err := m.whisperAPI.Post(context.TODO(), *message)
			if err != nil {
				m.logger.Error("failed to retry sending message", zap.String("hash", hash.String()), zap.Int("attempt", attempt+1), zap.Error(err))
				if m.handler != nil {
					m.handler.EnvelopeExpired(identifiers, err)
				}

			}
			envelopeID := statusproto.BytesToHash(hex)
			m.envelopes[envelopeID] = EnvelopePosted
			m.messages[envelopeID] = message
			m.attempts[envelopeID] = attempt + 1
			m.identifiers[envelopeID] = identifiers
		} else {
			m.logger.Debug("envelope expired", zap.String("hash", hash.String()))
			if m.handler != nil {
				m.handler.EnvelopeExpired(identifiers, err)
			}
		}
	}
}

func (m *EnvelopesMonitor) handleEventEnvelopeReceived(event whispertypes.EnvelopeEvent) {
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
	m.logger.Debug("expected envelope received", zap.String("hash", event.Hash.String()), zap.String("peer", event.Peer.String()))
	m.envelopes[event.Hash] = EnvelopeSent
	if m.handler != nil {
		m.handler.EnvelopeSent(m.identifiers[event.Hash])
	}
}

// clearMessageState removes all message and envelope state.
// not thread-safe, should be protected on a higher level.
func (m *EnvelopesMonitor) clearMessageState(envelopeID statusproto.Hash) {
	delete(m.envelopes, envelopeID)
	delete(m.messages, envelopeID)
	delete(m.attempts, envelopeID)
	delete(m.identifiers, envelopeID)
}
