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
	EnvelopeEventsHandler            EnvelopeEventsHandler
	AwaitOnlyMailServerConfirmations bool
	IsMailserver                     func(types.EnodeID) bool
	Logger                           *zap.Logger
}

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent([][]byte)
	EnvelopeExpired([][]byte, error)
	MailServerRequestCompleted(types2.Hash, types2.Hash, []byte, error)
	MailServerRequestExpired(types2.Hash)
}

// NewEnvelopesMonitor returns a pointer to an instance of the EnvelopesMonitor.
func NewEnvelopesMonitor(w types.Waku, config EnvelopesMonitorConfig) *EnvelopesMonitor {
	logger := config.Logger

	if logger == nil {
		logger = zap.NewNop()
	}

	return &EnvelopesMonitor{
		w:                                w,
		handler:                          config.EnvelopeEventsHandler,
		awaitOnlyMailServerConfirmations: config.AwaitOnlyMailServerConfirmations,
		isMailserver:                     config.IsMailserver,
		logger:                           logger.With(zap.Namespace("EnvelopesMonitor")),

		// key is envelope hash (event.Hash)
		envelopes: map[types2.Hash]*monitoredEnvelope{},

		// key is hash of the batch (event.Batch)
		batches: map[types2.Hash]map[types2.Hash]struct{}{},

		// key is stringified message identifier
		messageEnvelopeHashes: make(map[string][]types2.Hash),

		// key is an SDS message identifier; value is its application message
		// identifier. SDS aliases are tracked for retrieval hints but must never
		// be reported as independently sent UI messages.
		sdsApplicationMessageIDs: make(map[string]string),
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
	handler EnvelopeEventsHandler

	mu sync.Mutex

	envelopes                map[types2.Hash]*monitoredEnvelope
	batches                  map[types2.Hash]map[types2.Hash]struct{}
	messageEnvelopeHashes    map[string][]types2.Hash
	sdsApplicationMessageIDs map[string]string

	awaitOnlyMailServerConfirmations bool

	wg           sync.WaitGroup
	quit         chan struct{}
	isMailserver func(peer types.EnodeID) bool

	logger *zap.Logger
}

// AddSDSAlias records the SDS identifier associated with an application
// message. The alias shares its envelope hashes for retrieval hints but is not
// added to monitoredEnvelope.messageIDs, so publish acknowledgements only emit
// the application identifier.
func (m *EnvelopesMonitor) AddSDSAlias(applicationMessageID, sdsMessageID []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	applicationKey := types2.HexBytes(applicationMessageID).String()
	sdsKey := types2.HexBytes(sdsMessageID).String()
	hashes, ok := m.messageEnvelopeHashes[applicationKey]
	if !ok {
		return
	}
	m.messageEnvelopeHashes[sdsKey] = hashes
	m.sdsApplicationMessageIDs[sdsKey] = applicationKey
}

// TakeApplicationMessageIDForSDS resolves and consumes an SDS delivery
// association. Subsequent duplicate callbacks are ignored.
func (m *EnvelopesMonitor) TakeApplicationMessageIDForSDS(sdsMessageID []byte) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sdsKey := types2.HexBytes(sdsMessageID).String()
	applicationKey, ok := m.sdsApplicationMessageIDs[sdsKey]
	if !ok {
		return nil, false
	}
	applicationMessageID, err := types2.DecodeHex(applicationKey)
	if err != nil {
		return nil, false
	}
	delete(m.sdsApplicationMessageIDs, sdsKey)
	delete(m.messageEnvelopeHashes, sdsKey)
	return applicationMessageID, true
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
	handlers := map[types.EventType]func(types.EnvelopeEvent){
		types.EventEnvelopeSent:      m.handleEventEnvelopeSent,
		types.EventEnvelopeExpired:   m.handleEventEnvelopeExpired,
		types.EventBatchAcknowledged: m.handleAcknowledgedBatch,
		types.EventEnvelopeReceived:  m.handleEventEnvelopeReceived,
	}
	if handler, ok := handlers[event.Event]; ok {
		handler(event)
	}
}

func (m *EnvelopesMonitor) handleEventEnvelopeSent(event types.EnvelopeEvent) {
	// Mailserver confirmations for WakuV2 are disabled
	if m.w == nil && m.awaitOnlyMailServerConfirmations {
		if !m.isMailserver(event.Peer) {
			return
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	confirmationExpected := event.Batch != (types2.Hash{})

	envelope, ok := m.envelopes[event.Hash]

	// If confirmations are not expected, we keep track of the envelope
	// being sent
	if !ok && !confirmationExpected {
		m.envelopes[event.Hash] = &monitoredEnvelope{envelopeHashID: event.Hash, state: EnvelopeSent}
		return
	}

	// if message was already confirmed - skip it
	if envelope.state == EnvelopeSent {
		return
	}
	m.logger.Debug("envelope is sent", zap.String("hash", event.Hash.String()), zap.String("peer", event.Peer.String()))
	if confirmationExpected {
		if _, ok := m.batches[event.Batch]; !ok {
			m.batches[event.Batch] = map[types2.Hash]struct{}{}
		}
		m.batches[event.Batch][event.Hash] = struct{}{}
		m.logger.Debug("waiting for a confirmation", zap.String("batch", event.Batch.String()))
	} else {
		m.logger.Debug("confirmation not expected, marking as sent")
		envelope.state = EnvelopeSent
		m.processMessageIDs(envelope.messageIDs)
	}
}

func (m *EnvelopesMonitor) handleAcknowledgedBatch(event types.EnvelopeEvent) {

	if m.awaitOnlyMailServerConfirmations && !m.isMailserver(event.Peer) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	envelopes, ok := m.batches[event.Batch]
	if !ok {
		m.logger.Debug("batch is not found", zap.String("batch", event.Batch.String()))
	}
	m.logger.Debug("received a confirmation", zap.String("batch", event.Batch.String()), zap.String("peer", event.Peer.String()))
	envelopeErrors, ok := event.Data.([]types.EnvelopeError)
	if event.Data != nil && !ok {
		m.logger.Error("received unexpected data in the the confirmation event", zap.Any("data", event.Data))
	}
	failedEnvelopes := map[types2.Hash]struct{}{}
	for i := range envelopeErrors {
		envelopeError := envelopeErrors[i]
		_, exist := m.envelopes[envelopeError.Hash]
		if exist {
			m.logger.Warn("envelope that was posted by us is discarded", zap.String("hash", envelopeError.Hash.String()), zap.String("peer", event.Peer.String()), zap.String("error", envelopeError.Description))
			var err error
			switch envelopeError.Code {
			case types.EnvelopeTimeNotSynced:
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
		envelope, ok := m.envelopes[hash]
		if !ok || envelope.state == EnvelopeSent {
			continue
		}
		envelope.state = EnvelopeSent
		m.processMessageIDs(envelope.messageIDs)
	}
	delete(m.batches, event.Batch)
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
			m.handler.EnvelopeExpired(envelope.messageIDs, err)
		}
	}
}

func (m *EnvelopesMonitor) handleEventEnvelopeReceived(event types.EnvelopeEvent) {
	if m.awaitOnlyMailServerConfirmations && !m.isMailserver(event.Peer) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	envelope, ok := m.envelopes[event.Hash]
	if !ok || envelope.state != EnvelopePosted {
		return
	}
	m.logger.Debug("expected envelope received", zap.String("hash", event.Hash.String()), zap.String("peer", event.Peer.String()))
	envelope.state = EnvelopeSent
	m.processMessageIDs(envelope.messageIDs)
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
		m.handler.EnvelopeSent(sentMessageIDs)
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
		messageKey := types2.HexBytes(messageID).String()
		delete(m.messageEnvelopeHashes, messageKey)
		for sdsKey, applicationKey := range m.sdsApplicationMessageIDs {
			if applicationKey == messageKey {
				delete(m.sdsApplicationMessageIDs, sdsKey)
				delete(m.messageEnvelopeHashes, sdsKey)
			}
		}
	}
}
