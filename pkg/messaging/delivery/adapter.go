package delivery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/connection"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// sendHashTimeout bounds how long Send waits for the wire hash of a message
// the library accepted, when the caller's context has no deadline.
const sendHashTimeout = 30 * time.Second

// earlyResultTTL is how long an outcome that no Send is waiting for is kept.
// Most are for sends that already returned, so they are dropped unclaimed.
const earlyResultTTL = time.Minute

// ErrStoreQueryUnsupported is returned by StoreQuery: the Messaging API
// recovers history itself and exposes no store query.
var ErrStoreQueryUnsupported = errors.New("delivery: store queries are not exposed by the Messaging API")

// sendResult is the outcome of a send, keyed by its request ID.
type sendResult struct {
	hash []byte
	err  error
	at   time.Time
}

// Adapter is the messaging backend over the logos-delivery Messaging API.
// It satisfies types.Waku and transport.MessagingAPI.
type Adapter struct {
	client Client
	logger *zap.Logger

	envelopeFeed event.Feed

	// pending and early pair a request ID returned by Send with the event that
	// settles it, whichever arrives first.
	sendMu  sync.Mutex
	pending map[string]chan sendResult
	early   map[string]sendResult

	connMu          sync.Mutex
	connState       types.ConnectionState
	connSubscribers map[string]*types.ConnStatusSubscription

	historyReconcileNeeded chan types.HistoryReconcileWindow

	maxMessageSize atomic.Uint32

	quit     chan struct{}
	stopOnce sync.Once
	stopErr  error
	wg       sync.WaitGroup
}

// NewAdapter returns an Adapter driving client. The client must not be started.
func NewAdapter(client Client, logger *zap.Logger) *Adapter {
	return &Adapter{
		client:                 client,
		logger:                 logger.Named("delivery"),
		pending:                make(map[string]chan sendResult),
		early:                  make(map[string]sendResult),
		connSubscribers:        make(map[string]*types.ConnStatusSubscription),
		historyReconcileNeeded: make(chan types.HistoryReconcileWindow),
		quit:                   make(chan struct{}),
	}
}

var _ types.Waku = (*Adapter)(nil)

// Start starts the node and begins translating its events.
func (a *Adapter) Start() error {
	a.wg.Add(1)
	go a.eventLoop()
	if err := a.client.Start(); err != nil {
		return fmt.Errorf("delivery: start: %w", err)
	}
	size, err := a.client.MaxMessageSize()
	if err != nil {
		return fmt.Errorf("delivery: max message size: %w", err)
	}
	a.maxMessageSize.Store(uint32(size))
	return nil
}

// Stop stops the node and releases it. Later calls return the first result.
// The Adapter cannot be restarted.
func (a *Adapter) Stop() error {
	a.stopOnce.Do(func() {
		stopErr := a.client.Stop()
		closeErr := a.client.Close()
		close(a.quit)
		a.wg.Wait()

		a.connMu.Lock()
		for id, sub := range a.connSubscribers {
			sub.Unsubscribe()
			delete(a.connSubscribers, id)
		}
		a.connMu.Unlock()

		a.stopErr = errors.Join(stopErr, closeErr)
	})
	return a.stopErr
}

func (a *Adapter) eventLoop() {
	defer panics.LogOnPanic()
	defer a.wg.Done()

	events := a.client.Events()
	for {
		select {
		case <-a.quit:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			a.handleEvent(ev)
		}
	}
}

func (a *Adapter) handleEvent(ev Event) {
	switch e := ev.(type) {
	case ReceivedEvent:
		a.handleReceived(e)
	case SentEvent:
		a.settleSend(e.RequestID, sendResult{hash: decodeHash(e.MessageHash)})
	case PropagatedEvent:
		hash := decodeHash(e.MessageHash)
		a.settleSend(e.RequestID, sendResult{hash: hash})
		a.envelopeFeed.Send(types.EnvelopeEvent{
			Event: types.EventEnvelopeSent,
			Hash:  cryptotypes.BytesToHash(hash),
		})
	case ErrorEvent:
		hash := decodeHash(e.MessageHash)
		if a.settleSend(e.RequestID, sendResult{hash: hash, err: errors.New(e.Err)}) {
			return
		}
		a.envelopeFeed.Send(types.EnvelopeEvent{
			Event: types.EventEnvelopeExpired,
			Hash:  cryptotypes.BytesToHash(hash),
		})
	case ConnectionEvent:
		a.setConnectionState(toConnectionState(e.Status))
	}
}

// handleReceived reports a received message. It carries no pubsub topic: the
// Messaging API delivers by content topic.
func (a *Adapter) handleReceived(e ReceivedEvent) {
	hash := decodeHash(e.MessageHash)
	a.envelopeFeed.Send(types.EnvelopeEvent{
		Event: types.EventEnvelopeAvailable,
		Hash:  cryptotypes.BytesToHash(hash),
		Data: &types.ReceivedMessage{
			Hash:         hash,
			ContentTopic: e.ContentTopic,
			Payload:      e.Payload,
			Ephemeral:    e.Ephemeral,
			Meta:         e.Meta,
			Version:      e.Version,
			Timestamp:    e.Timestamp,
		},
	})
}

// settleSend hands result to the Send waiting on requestID, or keeps it for
// the Send that has not registered yet. The first outcome of a request wins.
// It reports whether a Send was waiting.
func (a *Adapter) settleSend(requestID string, result sendResult) bool {
	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	if ch, ok := a.pending[requestID]; ok {
		delete(a.pending, requestID)
		ch <- result
		return true
	}
	now := time.Now()
	for id, r := range a.early {
		if now.Sub(r.at) > earlyResultTTL {
			delete(a.early, id)
		}
	}
	if _, ok := a.early[requestID]; !ok {
		result.at = now
		a.early[requestID] = result
	}
	return false
}

// Send publishes a pre-encoded payload on contentTopic and returns its wire
// hash. The library reports the hash only with the first outcome of the send,
// usually its propagation, so Send waits for that.
//
// pubsubTopic and priority are not honoured: the Messaging API derives the
// shard from the content topic and has no send priority.
func (a *Adapter) Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error) {
	requestID, err := a.client.Send(ctx, contentTopic, payload, ephemeral)
	if err != nil {
		return nil, fmt.Errorf("delivery: send: %w", err)
	}

	ch := make(chan sendResult, 1)
	a.sendMu.Lock()
	if result, ok := a.early[requestID]; ok {
		delete(a.early, requestID)
		ch <- result
	} else {
		a.pending[requestID] = ch
	}
	a.sendMu.Unlock()

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sendHashTimeout)
		defer cancel()
	}

	select {
	case result := <-ch:
		if result.err != nil {
			return nil, fmt.Errorf("delivery: send %s: %w", requestID, result.err)
		}
		return result.hash, nil
	case <-ctx.Done():
		a.sendMu.Lock()
		delete(a.pending, requestID)
		a.sendMu.Unlock()
		return nil, fmt.Errorf("delivery: send %s: waiting for the message hash: %w", requestID, ctx.Err())
	}
}

// Subscribe subscribes the content topics. The pubsub topic is ignored: the
// transport counts subscriptions per content topic (see ContentTopicOnly).
func (a *Adapter) Subscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	for _, topic := range contentTopics {
		if err := a.client.Subscribe(topic.ContentTopic()); err != nil {
			return fmt.Errorf("delivery: subscribe %s: %w", topic.ContentTopic(), err)
		}
	}
	return nil
}

// Unsubscribe unsubscribes the content topics. The pubsub topic is ignored.
func (a *Adapter) Unsubscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	for _, topic := range contentTopics {
		if err := a.client.Unsubscribe(topic.ContentTopic()); err != nil {
			return fmt.Errorf("delivery: unsubscribe %s: %w", topic.ContentTopic(), err)
		}
	}
	return nil
}

// ContentTopicOnly marks the Adapter as subscribing and delivering by content
// topic alone.
func (a *Adapter) ContentTopicOnly() {}

// SubscribeEnvelopeEvents streams received messages and send outcomes.
func (a *Adapter) SubscribeEnvelopeEvents(events chan<- types.EnvelopeEvent) types.Subscription {
	return a.envelopeFeed.Subscribe(events)
}

func (a *Adapter) setConnectionState(state types.ConnectionState) {
	a.connMu.Lock()
	defer a.connMu.Unlock()
	a.connState = state
	status := types.ConnStatus{IsOnline: state.IsOnline(), State: state}
	for id, sub := range a.connSubscribers {
		if !sub.Send(status) {
			delete(a.connSubscribers, id)
		}
	}
}

// ConnectionState returns the last connection status the node reported.
func (a *Adapter) ConnectionState() types.ConnectionState {
	a.connMu.Lock()
	defer a.connMu.Unlock()
	return a.connState
}

// SubscribeToConnStatusChanges streams connection status changes.
func (a *Adapter) SubscribeToConnStatusChanges() (*types.ConnStatusSubscription, error) {
	a.connMu.Lock()
	defer a.connMu.Unlock()
	sub := types.NewConnStatusSubscription()
	a.connSubscribers[sub.ID] = sub
	return sub, nil
}

// OnHistoryReconcileNeeded never fires: the node recovers missed messages
// from store itself and reports them as received.
func (a *Adapter) OnHistoryReconcileNeeded() <-chan types.HistoryReconcileWindow {
	return a.historyReconcileNeeded
}

// MaxMessageSize returns the node's message size limit, read when it started.
func (a *Adapter) MaxMessageSize() uint32 {
	return a.maxMessageSize.Load()
}

// Pause is a no-op: the Messaging API cannot idle the node.
func (a *Adapter) Pause() error { return nil }

// Resume is a no-op, see Pause.
func (a *Adapter) Resume() error { return nil }

// ConnectionChanged is a no-op: the Messaging API takes no connectivity hint
// from the host.
func (a *Adapter) ConnectionChanged(connection.State) {}

// ConfirmMessageDelivered is a no-op: the node confirms its sends itself.
func (a *Adapter) ConfirmMessageDelivered([]common.Hash) {}

// Peers returns the node's connected peers, or none when it cannot tell.
func (a *Adapter) Peers() types.PeerStats {
	peers, err := a.client.ConnectedPeers()
	if err != nil {
		a.logger.Debug("failed to list connected peers", zap.Error(err))
		return types.PeerStats{}
	}
	return peers
}

// PeerID returns the node's peer ID, or an empty one when it cannot tell.
func (a *Adapter) PeerID() peer.ID {
	id, err := a.client.PeerID()
	if err != nil {
		a.logger.Debug("failed to read the peer ID", zap.Error(err))
		return ""
	}
	return id
}

// Metrics returns the node's metrics in the Prometheus text format.
func (a *Adapter) Metrics() string {
	metrics, err := a.client.Metrics()
	if err != nil {
		a.logger.Debug("failed to read metrics", zap.Error(err))
		return ""
	}
	return metrics
}

// StoreQuery always fails with ErrStoreQueryUnsupported.
func (a *Adapter) StoreQuery(
	ctx context.Context,
	batch types.MailserverBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return ErrStoreQueryUnsupported
}

func decodeHash(s string) []byte {
	b, err := hexutil.Decode(s)
	if err != nil {
		return nil
	}
	return b
}

func toConnectionState(s ConnectionStatus) types.ConnectionState {
	switch s {
	case Connected:
		return types.ConnectionStateConnected
	case PartiallyConnected:
		return types.ConnectionStatePartiallyConnected
	default:
		return types.ConnectionStateDisconnected
	}
}
