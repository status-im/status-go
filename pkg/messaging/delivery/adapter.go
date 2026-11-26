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

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/internal/connection"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// ErrStoreQueryUnsupported is returned by StoreQuery: the Messaging API
// recovers history itself and exposes no store query.
var ErrStoreQueryUnsupported = errors.New("store queries are not exposed by the Messaging API")

// Adapter is the messaging backend over the logos-delivery Messaging API.
// It satisfies types.Waku and transport.MessagingAPI.
type Adapter struct {
	mc     *messaging.MessagingClient
	logger *zap.Logger

	envelopeFeed event.Feed

	connMu          sync.Mutex
	connState       types.ConnectionState
	connSubscribers map[string]*types.ConnStatusSubscription

	historyReconcileNeeded chan types.HistoryReconcileWindow

	maxMessageSize atomic.Uint32

	// topics are the subscribed content topics, kept from before the node
	// starts, which Start then subscribes.
	topicsMu sync.Mutex
	started  bool
	topics   map[string]struct{}

	// calls is held shared by every call into the node and exclusively by
	// Stop, which must not release the node while one is in flight.
	calls  sync.RWMutex
	closed bool

	quit     chan struct{}
	stopOnce sync.Once
	stopErr  error
	wg       sync.WaitGroup
}

// NewAdapter creates a logos-delivery node from cfg and returns an Adapter
// driving it. The node is not started.
func NewAdapter(cfg messaging.Config, logger *zap.Logger) (*Adapter, error) {
	mc, err := messaging.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create node: %w", err)
	}
	return newAdapter(mc, logger), nil
}

func newAdapter(mc *messaging.MessagingClient, logger *zap.Logger) *Adapter {
	return &Adapter{
		mc:                     mc,
		logger:                 logger.Named("delivery"),
		connSubscribers:        make(map[string]*types.ConnStatusSubscription),
		topics:                 make(map[string]struct{}),
		historyReconcileNeeded: make(chan types.HistoryReconcileWindow, 16),
		quit:                   make(chan struct{}),
	}
}

var _ types.Waku = (*Adapter)(nil)

// Start starts the node and begins translating its events.
func (a *Adapter) Start() error {
	a.wg.Add(2)
	go a.eventLoop()
	go a.historyReconcileLoop()
	if err := a.mc.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := a.readNodeInfo(); err != nil {
		return errors.Join(err, a.mc.Stop())
	}
	a.topicsMu.Lock()
	defer a.topicsMu.Unlock()
	a.started = true
	for topic := range a.topics {
		if err := a.mc.Subscribe(topic); err != nil {
			return errors.Join(fmt.Errorf("subscribe %s: %w", topic, err), a.mc.Stop())
		}
	}
	return nil
}

func (a *Adapter) readNodeInfo() error {
	size, err := a.mc.Node().Debug().MaxMessageSize()
	if err != nil {
		return fmt.Errorf("max message size: %w", err)
	}
	a.maxMessageSize.Store(uint32(size))
	return nil
}

// rLockCalls read-locks calls unless the Adapter is stopped, in which case it
// returns false and holds no lock.
func (a *Adapter) rLockCalls() bool {
	a.calls.RLock()
	if a.closed {
		a.calls.RUnlock()
		return false
	}
	return true
}

func (a *Adapter) rUnlockCalls() { a.calls.RUnlock() }

// Stop stops the node and releases it, once the calls in flight return.
// Later calls return the first result. The Adapter cannot be restarted.
func (a *Adapter) Stop() error {
	a.stopOnce.Do(func() {
		a.calls.Lock()
		a.closed = true
		a.calls.Unlock()

		stopErr := a.mc.Stop()
		closeErr := a.mc.Close()
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

	events := a.mc.Events()
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

// historyReconcileLoop gives the Messenger the windows to fetch from store
// while the node is not Connected, as the go-waku backend does.
func (a *Adapter) historyReconcileLoop() {
	defer panics.LogOnPanic()
	defer a.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	reconciler := waku.NewHistoryReconciler(a.reliablyConnected(), time.Now())
	reconciler.Observe(a.reliablyConnected(), time.Now())
	for {
		var out chan<- types.HistoryReconcileWindow
		next, ok := reconciler.Next()
		if ok {
			out = a.historyReconcileNeeded
		}
		select {
		case <-a.quit:
			return
		case out <- next:
			reconciler.Done()
		case <-ticker.C:
			reconciler.Observe(a.reliablyConnected(), time.Now())
		}
	}
}

func (a *Adapter) reliablyConnected() bool {
	return a.ConnectionState() == types.ConnectionStateConnected
}

// handleEvent reports a node event. A received message carries no pubsub
// topic: the Messaging API delivers by content topic.
func (a *Adapter) handleEvent(ev messaging.Event) {
	switch e := ev.(type) {
	case messaging.MessageReceivedEvent:
		hash := decodeHash(e.MessageHash)
		a.envelopeFeed.Send(types.EnvelopeEvent{
			Event: types.EventEnvelopeAvailable,
			Hash:  cryptotypes.BytesToHash(hash),
			Data: &types.ReceivedMessage{
				Hash:         hash,
				ContentTopic: e.Message.ContentTopic,
				Payload:      e.Message.Payload,
				Ephemeral:    e.Message.Ephemeral,
				Meta:         e.Message.Meta,
				Version:      e.Message.Version,
				Timestamp:    e.Message.Timestamp,
			},
		})
	case messaging.MessagePropagatedEvent:
		a.envelopeFeed.Send(types.EnvelopeEvent{
			Event: types.EventEnvelopeSent,
			Hash:  cryptotypes.BytesToHash(decodeHash(e.MessageHash)),
		})
	case messaging.MessageErrorEvent:
		a.logger.Debug("send failed", zap.String("hash", e.MessageHash), zap.String("error", e.Err))
		a.envelopeFeed.Send(types.EnvelopeEvent{
			Event: types.EventEnvelopeExpired,
			Hash:  cryptotypes.BytesToHash(decodeHash(e.MessageHash)),
		})
	case messaging.ConnectionStatusEvent:
		a.setConnectionState(toConnectionState(e.Status))
	}
}

// Send queues a pre-encoded payload on contentTopic and returns its wire hash,
// without waiting for the message to go out. Like the go-waku backend, it
// reports the message as available right away: the node does not hand its own
// messages to its subscribers.
//
// A message the node refuses (*messaging.SendError) fails the send: the node
// queues nothing for it.
//
// pubsubTopic and priority are not honoured: the Messaging API derives the
// shard from the content topic and has no send priority.
func (a *Adapter) Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error) {
	if !a.rLockCalls() {
		return nil, fmt.Errorf("send: %w", messaging.ErrClosed)
	}
	messageHash, err := a.mc.Send(ctx, contentTopic, payload, ephemeral)
	a.rUnlockCalls()
	if err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}
	hash := decodeHash(messageHash)
	if hash == nil {
		return nil, fmt.Errorf("send: unexpected message hash %q", messageHash)
	}
	a.envelopeFeed.Send(types.EnvelopeEvent{
		Event: types.EventEnvelopeAvailable,
		Hash:  cryptotypes.BytesToHash(hash),
		Data: &types.ReceivedMessage{
			Hash:         hash,
			ContentTopic: contentTopic,
			Payload:      payload,
			Ephemeral:    ephemeral,
			Timestamp:    time.Now().UnixNano(),
		},
	})
	return hash, nil
}

// Subscribe subscribes the content topics, or, before Start, has Start
// subscribe them, as the go-waku backend does. The pubsub topic is ignored: the
// transport counts subscriptions per content topic (see ContentTopicOnly).
func (a *Adapter) Subscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	if !a.rLockCalls() {
		return fmt.Errorf("subscribe: %w", messaging.ErrClosed)
	}
	defer a.rUnlockCalls()
	a.topicsMu.Lock()
	defer a.topicsMu.Unlock()
	for _, topic := range contentTopics {
		if a.started {
			if err := a.mc.Subscribe(topic.ContentTopic()); err != nil {
				return fmt.Errorf("subscribe %s: %w", topic.ContentTopic(), err)
			}
		}
		a.topics[topic.ContentTopic()] = struct{}{}
	}
	return nil
}

// Unsubscribe unsubscribes the content topics. The pubsub topic is ignored.
func (a *Adapter) Unsubscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	if !a.rLockCalls() {
		return fmt.Errorf("unsubscribe: %w", messaging.ErrClosed)
	}
	defer a.rUnlockCalls()
	a.topicsMu.Lock()
	defer a.topicsMu.Unlock()
	for _, topic := range contentTopics {
		if a.started {
			if err := a.mc.Unsubscribe(topic.ContentTopic()); err != nil {
				return fmt.Errorf("unsubscribe %s: %w", topic.ContentTopic(), err)
			}
		}
		delete(a.topics, topic.ContentTopic())
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

// OnHistoryReconcileNeeded returns the windows to fetch from store: while the
// node is not Connected, and once more when it is again. Temporary, until
// logos-delivery owns the history fetch (https://github.com/status-im/status-go/issues/8104).
func (a *Adapter) OnHistoryReconcileNeeded() <-chan types.HistoryReconcileWindow {
	return a.historyReconcileNeeded
}

// MaxMessageSize returns the node's message size limit, read when it started.
func (a *Adapter) MaxMessageSize() uint32 {
	return a.maxMessageSize.Load()
}

// TrafficReport is empty: the Messaging API does not report traffic.
func (a *Adapter) TrafficReport() types.TrafficReport { return types.TrafficReport{} }

// ResetTrafficStats is a no-op, see TrafficReport.
func (a *Adapter) ResetTrafficStats() {}

// Pause is a no-op: the Messaging API cannot idle the node
// (https://github.com/logos-messaging/logos-delivery/issues/4495).
func (a *Adapter) Pause() error { return nil }

// Resume is a no-op, see Pause.
func (a *Adapter) Resume() error { return nil }

// ConnectionChanged is a no-op: the Messaging API takes no connectivity hint
// from the host.
func (a *Adapter) ConnectionChanged(connection.State) {}

// ConfirmMessageDelivered is a no-op: the node confirms its sends itself. To
// be removed with the other no-ops (https://github.com/status-im/status-go/issues/8103).
func (a *Adapter) ConfirmMessageDelivered([]common.Hash) {}

// Peers returns the node's connected peers, or none when it cannot tell.
func (a *Adapter) Peers() types.PeerStats {
	if !a.rLockCalls() {
		return types.PeerStats{}
	}
	defer a.rUnlockCalls()
	peers, err := a.mc.Node().Peers().ConnectedInfo()
	if err != nil {
		a.logger.Debug("failed to list connected peers", zap.Error(err))
		return types.PeerStats{}
	}
	stats := make(types.PeerStats, len(peers))
	for id, info := range peers {
		stats[id] = types.WakuV2Peer{Protocols: info.Protocols, Addresses: info.Addresses}
	}
	return stats
}

// PeerID returns the node's peer ID, or an empty one when it cannot tell.
func (a *Adapter) PeerID() peer.ID {
	if !a.rLockCalls() {
		return ""
	}
	defer a.rUnlockCalls()
	id, err := a.mc.Node().Debug().PeerID()
	if err != nil {
		a.logger.Debug("failed to read the peer ID", zap.Error(err))
		return ""
	}
	return id
}

// Metrics returns the node's metrics in the Prometheus text format.
func (a *Adapter) Metrics() string {
	if !a.rLockCalls() {
		return ""
	}
	defer a.rUnlockCalls()
	metrics, err := a.mc.Node().Debug().Metrics()
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

func toConnectionState(s messaging.ConnectionStatus) types.ConnectionState {
	switch s {
	case messaging.Connected:
		return types.ConnectionStateConnected
	case messaging.PartiallyConnected:
		return types.ConnectionStatePartiallyConnected
	default:
		return types.ConnectionStateDisconnected
	}
}
