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

	datasync2 "github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync"
	datasyncpeer "github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync/peer"
)

// pausedDuration is the tick interval handed to the mvds node while the host is
// backgrounded. We don't modify the mvds package: the interval is the `duration`
// argument to Node.Start, so we stop the node and recreate it with a different
// one. 5 minutes is rare enough that the SoC deep-sleeps essentially the whole
// interval (battery-wise indistinguishable from never ticking) yet still lets
// the outbound loop drain its in-memory ack buffer (mvds `payloads`) every 5
// minutes — so acks owed for messages received while backgrounded go out within
// ~5 min instead of waiting for the next foreground, and a queued message that
// ages past its SendEpoch dispatches on the first tick after resume.
const pausedDuration = 5 * time.Minute

var errNotStarted = errors.New("reliability layer not started")

// MessageDispatcher is a function that dispatches messages to a given public key.
//
//	publicKey: the recipient public key
//	wrappedPayload: the datasync wrapped payload
//	messages: the original messages that were wrapped
type MessageDispatcher func(publicKey *ecdsa.PublicKey, wrappedPayload []byte, messages [][]byte) error

type Reliability struct {
	identity              *ecdsa.PrivateKey
	mvdsPersistence       mvdsnode.Persistence
	mvdsStatusChangeEvent chan mvdsnode.PeerStatusChangeEvent
	sdsManager            *sds.ReliabilityManager
	logger                *zap.Logger

	// mu guards lifecycle transitions (Start/Stop/SetPaused/buildNode) so they
	// don't interleave. The hot path (Unwrap / WrapAndQueue) only does an atomic
	// Load of `datasync` and never takes mu.
	mu       sync.Mutex
	datasync atomic.Pointer[datasync2.DataSync]
	dispatch MessageDispatcher
	paused   bool
	tick     time.Duration // outbound-loop interval the active node was built with
}

func NewReliability(datasyncPersistence mvdsnode.Persistence, identity *ecdsa.PrivateKey, logger *zap.Logger) *Reliability {
	logger = logger.Named("reliability")
	return &Reliability{
		identity:              identity,
		mvdsPersistence:       datasyncPersistence,
		mvdsStatusChangeEvent: make(chan mvdsnode.PeerStatusChangeEvent, 5),
		sdsManager:            newSdsReliabilityManager(logger.Named("sds")),
		logger:                logger,
	}
}

func (r *Reliability) Start(dispatch MessageDispatcher) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dispatch = dispatch
	duration := datasync2.DatasyncTicker
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
	transport := datasync2.NewNodeTransport()
	node, err := mvdsnode.NewPersistentNode(
		r.mvdsPersistence,
		transport,
		datasyncpeer.PublicKeyToPeerID(r.identity.PublicKey),
		mvdsnode.BATCH,
		datasync2.CalculateSendTime,
		r.mvdsStatusChangeEvent,
		r.logger,
	)
	if err != nil {
		return err
	}
	ds := datasync2.New(node, transport, true, r.logger)
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
// handling keep running. Safe to call before Start (the state is recorded and
// applied when Start runs).
//
// Caveat: messages queued via WrapAndQueueMessageForDispatch while paused are
// persisted to SQLite but not transmitted until either the next pausedDuration
// tick or the next Resume (whichever comes first) — callers that must send
// promptly while backgrounded (e.g. the Android notification quick-reply) should
// Resume("messaging") first. Acks owed for messages received while paused live
// in the mvds node's in-memory `payloads` buffer; the pausedDuration tick drains
// them, but the resume-recreate drops whatever accumulated since the last tick
// (senders re-ack on retransmit, so no message loss — only delayed "delivered").
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
	// Stop accepting inbound packets into the (about-to-be-stopped) node: with
	// AddPacket non-blocking on a buffered channel, an in-flight Unwrap wouldn't
	// hang here — but once the node's Watch goroutine exits the packets it pushed
	// would be silently dropped. Short-circuit Unwrap so we don't decode and
	// then discard packets during the recreate window.
	old.SetSendingEnabled(false)
	old.Stop()
	stopDur := time.Since(start)
	duration := datasync2.DatasyncTicker
	if paused {
		duration = pausedDuration
	}
	buildStart := time.Now()
	if err := r.buildNode(duration); err != nil {
		// The old node is already stopped; don't leave the atomic pointer aimed
		// at a corpse. Started() now reports false and Unwrap/WrapAndQueue return
		// errNotStarted; the next Reliability.Start (e.g. on a connection change)
		// will rebuild.
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

func (r *Reliability) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old := r.datasync.Swap(nil); old != nil {
		old.SetSendingEnabled(false)
		old.Stop()
	}
	if r.sdsManager != nil {
		err := r.sdsManager.Cleanup()
		if err != nil {
			r.logger.Error("failed to cleanup sds reliability manager", zap.Error(err))
		}
		r.sdsManager = nil
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
	groupID := datasync2.ToOneToOneGroupID(&r.identity.PublicKey, publicKey)
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
