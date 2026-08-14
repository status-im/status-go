package reliability

import (
	"crypto/ecdsa"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/waku-org/sds-go-bindings/sds"

	mvdsnode "github.com/status-im/mvds/node"
	mvdsproto "github.com/status-im/mvds/protobuf"
	mvdsstate "github.com/status-im/mvds/state"

	"github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync"
	datasyncpeer "github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync/peer"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability/protobuf"
)

// pausedDuration is the tick interval handed to the mvds node while the host is
// backgrounded and service paused
const pausedDuration = 5 * time.Minute

var errNotStarted = errors.New("reliability layer not started")

// MessageDispatcher is a function that dispatches messages to a given public key.
//
//	publicKey: the recipient public key
//	wrappedPayload: the datasync wrapped payload
//	messages: the original messages that were wrapped
type MessageDispatcher func(publicKey *ecdsa.PublicKey, wrappedPayload []byte, messages [][]byte) error

// MissingDependenciesHandler is triggered when SDS reports missing dependencies
// for an incoming message.
type MissingDependenciesHandler func(messageID string, missingDeps []string, channelID string) error

// RetrievalHintProvider returns a transport-level retrieval hint for a given
// SDS message ID.
type RetrievalHintProvider func(messageID string) []byte

// MessageSentHandler is triggered when SDS confirms an outgoing message was
// observed in an incoming message's bloom filter or causal history.
type MessageSentHandler func(messageID string)

type Reliability struct {
	identity              *ecdsa.PrivateKey
	mvdsPersistence       mvdsnode.Persistence
	mvdsStatusChangeEvent chan mvdsnode.PeerStatusChangeEvent
	sdsManager            *sds.ReliabilityManager
	logger                *zap.Logger

	// mu guards lifecycle transitions (Start/Stop/SetPaused/buildNode) so they
	// don't interleave. The hot path (Unwrap / WrapAndQueue) never takes mu;
	// datasync uses atomic loads and SDS uses sdsOpsMu.
	mu       sync.Mutex
	datasync atomic.Pointer[datasync.DataSync]
	dispatch MessageDispatcher
	paused   bool
	tick     time.Duration // outbound-loop interval the active node was built with

	missingDepsHandlerMu sync.RWMutex
	missingDepsHandler   MissingDependenciesHandler

	retrievalHintProviderMu sync.RWMutex
	retrievalHintProvider   RetrievalHintProvider

	messageSentHandlerMu sync.RWMutex
	messageSentHandler   MessageSentHandler

	sdsManagerFactory sdsManagerFactoryFunc

	// sdsOpsMu coordinates SDS manager calls with shutdown. SDS hot-path calls
	// hold RLock for the full native call; Close holds Lock before Cleanup so
	// Cleanup cannot race with in-flight SDS manager usage.
	sdsOpsMu sync.RWMutex
}

func NewReliability(datasyncPersistence mvdsnode.Persistence, identity *ecdsa.PrivateKey, missingDepsHandler MissingDependenciesHandler, logger *zap.Logger) (*Reliability, error) {
	return newReliabilityWithSDSFactory(datasyncPersistence, identity, missingDepsHandler, logger, sds.NewReliabilityManager)
}

func newReliabilityWithSDSFactory(
	datasyncPersistence mvdsnode.Persistence,
	identity *ecdsa.PrivateKey,
	missingDepsHandler MissingDependenciesHandler,
	logger *zap.Logger,
	sdsManagerFactory sdsManagerFactoryFunc,
) (*Reliability, error) {
	if missingDepsHandler == nil {
		return nil, errors.New("missingDepsHandler is required")
	}
	if sdsManagerFactory == nil {
		return nil, errors.New("sdsManagerFactory is required")
	}
	logger = logger.Named("reliability")
	r := &Reliability{
		identity:              identity,
		mvdsPersistence:       datasyncPersistence,
		mvdsStatusChangeEvent: make(chan mvdsnode.PeerStatusChangeEvent, 5),
		logger:                logger,
		missingDepsHandler:    missingDepsHandler,
		sdsManagerFactory:     sdsManagerFactory,
	}

	if err := r.buildSdsManager(); err != nil {
		return nil, err
	}

	return r, nil
}

// buildSdsManager installs a fresh SDS reliability manager. Close() destroys the
// previous one, so this must run again on every Start().
func (r *Reliability) buildSdsManager() error {
	manager, err := newSdsReliabilityManager(
		r.sdsManagerFactory,
		r.logger.Named("sds"),
		r.handleSDSMessageSent,
		r.handleMissingDependencies,
		r.provideRetrievalHint,
	)
	if err != nil {
		return err
	}

	r.sdsOpsMu.Lock()
	r.sdsManager = manager
	r.sdsOpsMu.Unlock()
	return nil
}

// SetMissingDependenciesHandler configures how SDS missing dependencies should be handled.
func (r *Reliability) SetMissingDependenciesHandler(handler MissingDependenciesHandler) {
	r.missingDepsHandlerMu.Lock()
	defer r.missingDepsHandlerMu.Unlock()
	r.missingDepsHandler = handler
}

// SetRetrievalHintProvider configures how SDS retrieval hints are resolved.
func (r *Reliability) SetRetrievalHintProvider(provider RetrievalHintProvider) {
	r.retrievalHintProviderMu.Lock()
	defer r.retrievalHintProviderMu.Unlock()
	r.retrievalHintProvider = provider
}

// SetMessageSentHandler configures the handler for outgoing SDS delivery
// confirmations.
func (r *Reliability) SetMessageSentHandler(handler MessageSentHandler) {
	r.messageSentHandlerMu.Lock()
	defer r.messageSentHandlerMu.Unlock()
	r.messageSentHandler = handler
}

func (r *Reliability) handleSDSMessageSent(messageID sds.MessageID, _ string) {
	r.messageSentHandlerMu.RLock()
	handler := r.messageSentHandler
	r.messageSentHandlerMu.RUnlock()
	if handler != nil {
		handler(string(messageID))
	}
}

func (r *Reliability) provideRetrievalHint(messageID sds.MessageID) []byte {
	r.retrievalHintProviderMu.RLock()
	provider := r.retrievalHintProvider
	r.retrievalHintProviderMu.RUnlock()

	if provider == nil {
		return nil
	}

	return provider(string(messageID))
}

func (r *Reliability) handleMissingDependencies(messageID sds.MessageID, missingDeps []sds.HistoryEntry, channelID string) {
	r.missingDepsHandlerMu.RLock()
	handler := r.missingDepsHandler
	r.missingDepsHandlerMu.RUnlock()

	if handler == nil || len(missingDeps) == 0 {
		return
	}

	missingDepsAsString := make([]string, 0, len(missingDeps))
	for _, dep := range missingDeps {
		if len(dep.RetrievalHint) == 0 {
			continue
		}

		var hint protobuf.RetrievalHint
		if err := proto.Unmarshal(dep.RetrievalHint, &hint); err != nil {
			r.logger.Debug("failed to unmarshal retrieval hint",
				zap.String("messageId", string(dep.MessageID)),
				zap.Error(err))
			continue
		}

		for _, envelopeHash := range hint.EnvelopeHashes {
			if len(envelopeHash) > 0 {
				missingDepsAsString = append(missingDepsAsString, string(envelopeHash))
			}
		}
	}
	if len(missingDepsAsString) == 0 {
		return
	}

	if err := handler(string(messageID), missingDepsAsString, channelID); err != nil {
		r.logger.Debug("failed to fetch missing dependencies from sds callback", zap.Error(err))
	}
}

func (r *Reliability) Start(dispatch MessageDispatcher) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dispatch = dispatch
	if r.sdsManager == nil {
		if err := r.buildSdsManager(); err != nil {
			return err
		}
	}
	duration := datasync.DatasyncTicker
	if r.paused {
		duration = pausedDuration
	}
	return r.buildNode(duration)
}

// buildNode (re)creates the mvds data-sync node with the given outbound-loop
// interval and installs it. Caller must hold r.mu.
//
// NewPersistentNode reloads the last persisted epoch from r.mvdsPersistence
// (SQLite) on construction, so the epoch counter survives a recreate; the
// status-change channel is reused so peer-online events keep flowing.
func (r *Reliability) buildNode(duration time.Duration) error {
	transport := datasync.NewNodeTransport()
	node, err := mvdsnode.NewPersistentNode(
		r.mvdsPersistence,
		transport,
		datasyncpeer.PublicKeyToPeerID(r.identity.PublicKey),
		mvdsnode.BATCH,
		datasync.CalculateSendTime,
		r.mvdsStatusChangeEvent,
		r.logger,
	)
	if err != nil {
		return err
	}
	ds := datasync.New(node, transport, true, r.logger)
	ds.Init(r.mvdsDispatch, r.logger)
	ds.Start(duration)
	r.tick = duration
	r.datasync.Store(ds)
	return nil
}

// currentTick returns the outbound-loop interval the active mvds node was built
// with — DatasyncTicker when running, pausedDuration when paused. Used by tests.
func (r *Reliability) currentTick() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tick
}

// mvdsDispatch adapts mvds's dispatch signature to r.dispatch.
func (r *Reliability) mvdsDispatch(receiver mvdsstate.PeerID, payload *mvdsproto.Payload) error {
	if !payload.IsValid() {
		return errors.New("payload is invalid")
	}
	marshalledPayload, err := proto.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal payload")
	}
	publicKey, err := datasyncpeer.IDToPublicKey(receiver)
	if err != nil {
		return errors.Wrap(err, "failed to convert id to public key")
	}
	messages := make([][]byte, 0, len(payload.Messages))
	for _, msg := range payload.Messages {
		messages = append(messages, msg.Body)
	}
	return r.dispatch(publicKey, marshalledPayload, messages)
}

// SetPaused idles (paused==true) or re-arms (paused==false) the mvds outbound
// loop by stopping the data-sync node and recreating it with a different tick
// interval (pausedDuration vs DatasyncTicker). While paused the loop's outbound
// work runs only every pausedDuration; inbound processing and peer-status
// handling keep running.
func (r *Reliability) SetPaused(paused bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.paused == paused {
		return nil
	}
	r.paused = paused
	old := r.datasync.Load()
	if old == nil {
		return nil // not started yet; Start will pick the right interval
	}
	start := time.Now()
	// Stop accepting inbound packets into the (about-to-be-stopped) node
	old.SetSendingEnabled(false)
	old.Stop()
	stopDur := time.Since(start)
	duration := datasync.DatasyncTicker
	if paused {
		duration = pausedDuration
	}
	buildStart := time.Now()
	if err := r.buildNode(duration); err != nil {
		r.datasync.Store(nil)
		r.logger.Error("reliability SetPaused: rebuild failed; data-sync node is down until next Start", zap.Error(err))
		return err
	}
	r.logger.Info("reliability SetPaused",
		zap.Bool("paused", paused),
		zap.Duration("stop", stopDur),
		zap.Duration("build", time.Since(buildStart)),
		zap.Duration("total", time.Since(start)))
	return nil
}

// Stop idles the data-sync node. The SDS manager deliberately stays alive: its
// bloom filter and causal history are what let us detect messages missed while
// offline, so a connectivity drop must not discard them. Use Close for shutdown.
func (r *Reliability) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopLocked()
}

func (r *Reliability) stopLocked() {
	if old := r.datasync.Swap(nil); old != nil {
		old.SetSendingEnabled(false)
		old.Stop()
	}
}

// Close stops the data-sync node and releases the SDS manager's native resources.
func (r *Reliability) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopLocked()
	r.sdsOpsMu.Lock()
	defer r.sdsOpsMu.Unlock()
	if old := r.sdsManager; old != nil {
		r.sdsManager = nil
		err := old.Cleanup()
		if err != nil {
			r.logger.Error("failed to cleanup sds reliability manager", zap.Error(err))
		}
	}
}

func (r *Reliability) Started() bool {
	return r.datasync.Load() != nil
}

// WrapAndQueueMessageForDispatch wraps the message in the reliability layer,
// then queues it for delivery to the target public key using configured MVDSDispatcher.
func (r *Reliability) WrapAndQueueMessageForDispatch(publicKey *ecdsa.PublicKey, message []byte) (mvdsstate.MessageID, error) {
	ds := r.datasync.Load()
	if ds == nil {
		return mvdsstate.MessageID{}, errNotStarted
	}
	groupID := datasync.ToOneToOneGroupID(&r.identity.PublicKey, publicKey)
	peerID := datasyncpeer.PublicKeyToPeerID(*publicKey)
	exist, err := ds.IsPeerInGroup(groupID, peerID)
	if err != nil {
		return mvdsstate.MessageID{}, errors.Wrap(err, "failed to check if peer is in group")
	}
	if !exist {
		if err := ds.AddPeer(groupID, peerID); err != nil {
			return mvdsstate.MessageID{}, errors.Wrap(err, "failed to add peer")
		}
	}
	return ds.AppendMessage(groupID, message)
}

// UnwrapAndAcknowledge tries to unwrap received datasync message,
// and potentially acknowledges it.
func (r *Reliability) UnwrapAndAcknowledgeMessage(publicKey *ecdsa.PublicKey, message []byte) (*mvdsproto.Payload, error) {
	ds := r.datasync.Load()
	if ds == nil {
		return nil, errNotStarted
	}
	return ds.Unwrap(publicKey, message)
}

// ReportPeerOnline reports to MVDS that a peer is online at a given event time.
func (r *Reliability) ReportPeerOnline(publicKey *ecdsa.PublicKey, eventTime uint64) {
	select {
	case r.mvdsStatusChangeEvent <- mvdsnode.PeerStatusChangeEvent{
		PeerID:    datasyncpeer.PublicKeyToPeerID(*publicKey),
		Status:    mvdsnode.OnlineStatus,
		EventTime: eventTime,
	}:
	default:
		r.logger.Debug("mvdsStatusChangeEvent channel is full")
	}
}
