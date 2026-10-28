package delivery

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/internal/connection"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// ErrStoreQueryUnsupported is returned by StoreQuery: the Messaging API
// recovers history itself and exposes no store query.
var ErrStoreQueryUnsupported = errors.New("delivery: store queries are not exposed by the Messaging API")

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
		return nil, fmt.Errorf("delivery: create node: %w", err)
	}
	return newAdapter(mc, logger), nil
}

func newAdapter(mc *messaging.MessagingClient, logger *zap.Logger) *Adapter {
	return &Adapter{
		mc:                     mc,
		logger:                 logger.Named("delivery"),
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
	if err := a.mc.Start(); err != nil {
		return fmt.Errorf("delivery: start: %w", err)
	}
	size, err := a.mc.Node().Debug().MaxMessageSize()
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

func (a *Adapter) handleEvent(ev messaging.Event) {
	switch e := ev.(type) {
	case messaging.MessageReceivedEvent:
		a.handleReceived(e)
	case messaging.MessagePropagatedEvent:
		a.envelopeFeed.Send(types.EnvelopeEvent{
			Event: types.EventEnvelopeSent,
			Hash:  cryptotypes.BytesToHash(requestKey(string(e.RequestID))),
			Data:  cryptotypes.BytesToHash(decodeHash(e.MessageHash)),
		})
	case messaging.MessageErrorEvent:
		a.logger.Debug("send failed", zap.String("requestID", string(e.RequestID)), zap.String("error", e.Err))
		a.envelopeFeed.Send(types.EnvelopeEvent{
			Event: types.EventEnvelopeExpired,
			Hash:  cryptotypes.BytesToHash(requestKey(string(e.RequestID))),
		})
	case messaging.ConnectionStatusEvent:
		a.setConnectionState(toConnectionState(e.Status))
	}
}

// handleReceived reports a received message. It carries no pubsub topic: the
// Messaging API delivers by content topic.
func (a *Adapter) handleReceived(e messaging.MessageReceivedEvent) {
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
}

// Send queues a pre-encoded payload on contentTopic and returns a key derived
// from the library's request ID, without waiting for the message to go out.
// The envelope events of the send carry the same key; the sent event also
// carries the wire hash in Data.
//
// pubsubTopic and priority are not honoured: the Messaging API derives the
// shard from the content topic and has no send priority.
func (a *Adapter) Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error) {
	requestID, err := a.mc.Send(ctx, contentTopic, payload, ephemeral)
	if err != nil {
		return nil, fmt.Errorf("delivery: send: %w", err)
	}
	key := requestKey(string(requestID))
	if key == nil {
		return nil, fmt.Errorf("delivery: send: unexpected request ID %q", requestID)
	}
	return key, nil
}

// requestKey is the envelope key of a send: its hex request ID, left-padded
// to 32 bytes, so it matches the library's logs.
func requestKey(requestID string) []byte {
	id, err := hex.DecodeString(requestID)
	if err != nil || len(id) > cryptotypes.HashLength {
		return nil
	}
	return cryptotypes.BytesToHash(id).Bytes()
}

// Subscribe subscribes the content topics. The pubsub topic is ignored: the
// transport counts subscriptions per content topic (see ContentTopicOnly).
func (a *Adapter) Subscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	for _, topic := range contentTopics {
		if err := a.mc.Subscribe(topic.ContentTopic()); err != nil {
			return fmt.Errorf("delivery: subscribe %s: %w", topic.ContentTopic(), err)
		}
	}
	return nil
}

// Unsubscribe unsubscribes the content topics. The pubsub topic is ignored.
func (a *Adapter) Unsubscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	for _, topic := range contentTopics {
		if err := a.mc.Unsubscribe(topic.ContentTopic()); err != nil {
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
	id, err := a.mc.Node().Debug().PeerID()
	if err != nil {
		a.logger.Debug("failed to read the peer ID", zap.Error(err))
		return ""
	}
	return id
}

// Metrics returns the node's metrics in the Prometheus text format.
func (a *Adapter) Metrics() string {
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
