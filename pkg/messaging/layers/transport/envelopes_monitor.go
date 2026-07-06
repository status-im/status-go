package transport

import (
	"errors"
	"sync"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// EnvelopeState in local tracker
type EnvelopeState int

const (
	// NotRegistered returned if asked hash wasn't registered in the tracker.
	NotRegistered EnvelopeState = -1
	// EnvelopePosted is set when envelope was added to a local waku queue.
	EnvelopePosted EnvelopeState = iota + 1
	// EnvelopeSent is set when envelope is sent to at least one peer.
	EnvelopeSent
)

type EnvelopesMonitorConfig struct {
	Logger *zap.Logger
}

// MessageEventsHandler receives message-level delivery events produced by the
// transport. This is the transport's outbound event contract: waku envelope
// events stay inside the transport, which aggregates them per message and
// reports message IDs to the layer above.
type MessageEventsHandler interface {
	// MessagesSent reports messages whose envelopes were all confirmed as sent.
	MessagesSent(messageIDs [][]byte)
	// MessagesExpired reports messages that failed to be sent.
	MessagesExpired(messageIDs [][]byte, err error)
}

// NewEnvelopesMonitor returns a pointer to an instance of the EnvelopesMonitor.
func NewEnvelopesMonitor(w types.Waku, config EnvelopesMonitorConfig) *EnvelopesMonitor {
	logger := config.Logger

	if logger == nil {
		logger = zap.NewNop()
	}

	return &EnvelopesMonitor{
		w:      w,
		logger: logger.With(zap.Namespace("EnvelopesMonitor")),

		// key is envelope hash (event.Hash)
		envelopes: map[types2.Hash]*monitoredEnvelope{},

		// key is stringified message identifier
		messageEnvelopeHashes: make(map[string][]types2.Hash),
	}
}

type monitoredEnvelope struct {
	envelopeHashID types2.Hash
	state          EnvelopeState
	messageIDs     [][]byte
}

// EnvelopesMonitor is responsible for monitoring waku envelopes state.
type EnvelopesMonitor struct {
	gocommon.PauseBroadcaster

	w       types.Waku
	handler MessageEventsHandler

	mu sync.Mutex

	envelopes             map[types2.Hash]*monitoredEnvelope
	messageEnvelopeHashes map[string][]types2.Hash

	wg   sync.WaitGroup
	quit chan struct{}

	logger *zap.Logger
}

// Start processing events.
func (m *EnvelopesMonitor) Start() {
	m.quit = make(chan struct{})
	m.wg.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer m.wg.Done()
		m.handleEnvelopeEvents()
	}()
}

// Stop process events.
func (m *EnvelopesMonitor) Stop() {
	close(m.quit)
	m.wg.Wait()
}

// Add hashes to a tracker.
// Identifiers may be backed by multiple envelopes. It happens when message is split in segmentation layer.
func (m *EnvelopesMonitor) Add(messageIDs [][]byte, envelopeHashes []types2.Hash, messages []*types.NewMessage) error {
	if len(envelopeHashes) != len(messages) {
		return errors.New("hashes don't match messages")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, messageID := range messageIDs {
		m.messageEnvelopeHashes[types2.HexBytes(messageID).String()] = envelopeHashes
	}

	for _, envelopeHash := range envelopeHashes {
		if _, ok := m.envelopes[envelopeHash]; !ok {
			m.envelopes[envelopeHash] = &monitoredEnvelope{
				envelopeHashID: envelopeHash,
				state:          EnvelopePosted,
				messageIDs:     messageIDs,
			}
		}
	}

	m.processMessageIDs(messageIDs)

	return nil
}

func (m *EnvelopesMonitor) GetState(hash types2.Hash) EnvelopeState {
	m.mu.Lock()
	defer m.mu.Unlock()
	envelope, exist := m.envelopes[hash]
	if !exist {
		return NotRegistered
	}
	return envelope.state
}

// handleEnvelopeEvents processes waku envelope events
func (m *EnvelopesMonitor) handleEnvelopeEvents() {
	events := make(chan types.EnvelopeEvent, 100) // must be buffered to prevent blocking waku
	sub := m.w.SubscribeEnvelopeEvents(events)
	defer func() {
		sub.Unsubscribe()
	}()
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
func (m *EnvelopesMonitor) handleEvent(event types.EnvelopeEvent) {
	switch event.Event {
	case types.EventEnvelopeSent:
		m.handleEventEnvelopeSent(event)
	case types.EventEnvelopeExpired:
		m.handleEventEnvelopeExpired(event)
	}
}

func (m *EnvelopesMonitor) handleEventEnvelopeSent(event types.EnvelopeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	envelope, ok := m.envelopes[event.Hash]

	// If we don't track this envelope, keep track of it as already sent, so a
	// later Add for the same envelope resolves immediately.
	if !ok {
		m.envelopes[event.Hash] = &monitoredEnvelope{envelopeHashID: event.Hash, state: EnvelopeSent}
		return
	}

	// if message was already confirmed - skip it
	if envelope.state == EnvelopeSent {
		return
	}
	m.logger.Debug("envelope is sent", zap.String("hash", event.Hash.String()))
	envelope.state = EnvelopeSent
	m.processMessageIDs(envelope.messageIDs)
}

func (m *EnvelopesMonitor) handleEventEnvelopeExpired(event types.EnvelopeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handleEnvelopeFailure(event.Hash, errors.New("envelope expired due to connectivity issues"))
}

// handleEnvelopeFailure is a common code path for processing envelopes failures. not thread safe, lock
// must be used on a higher level.
//
// Transport-level retry was removed (logos-messaging/pm#380): logos-delivery
// performs retransmission internally. A failed envelope is now reported as
// expired immediately; the message stays unconfirmed and the app layer (raw
// message resend) remains the backstop until the Messaging API is integrated.
func (m *EnvelopesMonitor) handleEnvelopeFailure(hash types2.Hash, err error) {
	if envelope, ok := m.envelopes[hash]; ok {
		m.clearMessageState(hash)
		if envelope.state == EnvelopeSent {
			return
		}
		m.logger.Debug("envelope expired", zap.String("hash", hash.String()))
		if m.handler != nil {
			m.handler.MessagesExpired(envelope.messageIDs, err)
		}
	}
}

func (m *EnvelopesMonitor) processMessageIDs(messageIDs [][]byte) {
	sentMessageIDs := make([][]byte, 0, len(messageIDs))

	for _, messageID := range messageIDs {
		hashes, ok := m.messageEnvelopeHashes[types2.HexBytes(messageID).String()]
		if !ok {
			continue
		}

		sent := true
		// Consider message as sent if all corresponding envelopes are in EnvelopeSent state
		for _, hash := range hashes {
			envelope, ok := m.envelopes[hash]
			if !ok || envelope.state != EnvelopeSent {
				sent = false
				break
			}
		}
		if sent {
			sentMessageIDs = append(sentMessageIDs, messageID)
		}
	}

	if len(sentMessageIDs) > 0 && m.handler != nil {
		m.handler.MessagesSent(sentMessageIDs)
	}
}

// clearMessageState removes all message and envelope state.
// not thread-safe, should be protected on a higher level.
func (m *EnvelopesMonitor) clearMessageState(envelopeID types2.Hash) {
	envelope, ok := m.envelopes[envelopeID]
	if !ok {
		return
	}
	delete(m.envelopes, envelopeID)
	for _, messageID := range envelope.messageIDs {
		delete(m.messageEnvelopeHashes, types2.HexBytes(messageID).String())
	}
}
