package messaging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/delivery"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// deliveryTestBackend is a logos-delivery node joined to a private two-node
// cluster: relay needs a peer to publish to, so a second node serves as one.
//
// Its cores share the node, so it counts their subscriptions: one core
// dropping a topic must not unsubscribe the others.
type deliveryTestBackend struct {
	*delivery.Adapter
	peer    *delivery.Adapter
	dataDir string
	skip    atomic.Bool

	subscriptionsMu sync.Mutex
	subscriptions   map[types.TopicSubscription]int

	// failedSends carries the expiry of sends dropped while skip is set.
	failedSends event.Feed
}

func newLogosDeliveryTestBackend() (testBackend, error) {
	return newDeliveryTestBackend()
}

func newDeliveryTestBackend() (*deliveryTestBackend, error) {
	peerKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	peerPub, err := libp2pcrypto.UnmarshalSecp256k1PublicKey(crypto.CompressPubkey(&peerKey.PublicKey))
	if err != nil {
		return nil, err
	}
	peerID, err := peer.IDFromPublicKey(peerPub)
	if err != nil {
		return nil, err
	}
	peerPort, err := freeTCPPort()
	if err != nil {
		return nil, err
	}

	dataDir, err := os.MkdirTemp("", "logos-delivery-test-")
	if err != nil {
		return nil, err
	}
	peerNode, err := newTestDeliveryAdapter(bindings.Overrides{
		"nodekey":            hex.EncodeToString(crypto.FromECDSA(peerKey)),
		"tcp-port":           peerPort,
		"local-storage-path": filepath.Join(dataDir, "peer"),
	})
	if err != nil {
		return nil, err
	}
	nodePort, err := freeTCPPort()
	if err != nil {
		return nil, err
	}
	node, err := newTestDeliveryAdapter(bindings.Overrides{
		"tcp-port":           nodePort,
		"local-storage-path": filepath.Join(dataDir, "node"),
		"entry-node":         []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", peerPort, peerID)},
	})
	if err != nil {
		return nil, err
	}
	return &deliveryTestBackend{
		Adapter:       node,
		peer:          peerNode,
		dataDir:       dataDir,
		subscriptions: make(map[types.TopicSubscription]int),
	}, nil
}

func newTestDeliveryAdapter(overrides bindings.Overrides) (*delivery.Adapter, error) {
	cfg := bindings.Config{
		Mode: bindings.ModeCore,
		MessagingOverrides: bindings.Overrides{
			"cluster-id":            16,
			"num-shards-in-network": 1,
			"max-msg-size":          "1024KiB",
			"listen-address":        "127.0.0.1",
			"discv5-discovery":      false,
			"reliability":           false,
			"backfill-enabled":      false,
			"nat":                   "none",
			"log-level":             "ERROR",
		},
	}
	for k, v := range overrides {
		cfg.MessagingOverrides[k] = v
	}
	return delivery.NewAdapter(cfg, zap.NewNop())
}

func freeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// Start starts both nodes and waits until they are connected.
func (b *deliveryTestBackend) Start() error {
	if err := b.peer.Start(); err != nil {
		return err
	}
	if err := b.Adapter.Start(); err != nil {
		return err
	}
	deadline := time.Now().Add(60 * time.Second)
	for !b.Adapter.ConnectionState().IsOnline() {
		if time.Now().After(deadline) {
			return errors.New("logos-delivery test node did not connect to its peer")
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (b *deliveryTestBackend) Stop() error {
	return errors.Join(b.Adapter.Stop(), b.peer.Stop(), os.RemoveAll(b.dataDir))
}

func (b *deliveryTestBackend) SkipPublishToTopic(value bool) {
	b.skip.Store(value)
}

// Send drops the message while skip is set, the way the go-waku backend does:
// it returns a request ID, then reports the send expired.
func (b *deliveryTestBackend) Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) (types.RequestID, error) {
	if !b.skip.Load() {
		return b.Adapter.Send(ctx, pubsubTopic, contentTopic, payload, ephemeral, priority)
	}
	id := make([]byte, 10)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	requestID := types.RequestID(hex.EncodeToString(id))
	go func() {
		defer panics.LogOnPanic()
		b.failedSends.Send(types.EnvelopeEvent{Event: types.EventEnvelopeExpired, RequestID: requestID})
	}()
	return requestID, nil
}

func (b *deliveryTestBackend) SubscribeEnvelopeEvents(events chan<- types.EnvelopeEvent) types.Subscription {
	return event.JoinSubscriptions(
		b.Adapter.SubscribeEnvelopeEvents(events),
		b.failedSends.Subscribe(events),
	)
}

func (b *deliveryTestBackend) Subscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	b.subscriptionsMu.Lock()
	defer b.subscriptionsMu.Unlock()
	var first []types.TopicType
	for _, topic := range contentTopics {
		key := types.TopicSubscription{PubsubTopic: pubsubTopic, ContentTopic: topic}
		b.subscriptions[key]++
		if b.subscriptions[key] == 1 {
			first = append(first, topic)
		}
	}
	if len(first) == 0 {
		return nil
	}
	return b.Adapter.Subscribe(ctx, pubsubTopic, first)
}

func (b *deliveryTestBackend) Unsubscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	b.subscriptionsMu.Lock()
	defer b.subscriptionsMu.Unlock()
	var last []types.TopicType
	for _, topic := range contentTopics {
		key := types.TopicSubscription{PubsubTopic: pubsubTopic, ContentTopic: topic}
		if b.subscriptions[key] == 0 {
			continue
		}
		b.subscriptions[key]--
		if b.subscriptions[key] == 0 {
			delete(b.subscriptions, key)
			last = append(last, topic)
		}
	}
	if len(last) == 0 {
		return nil
	}
	return b.Adapter.Unsubscribe(ctx, pubsubTopic, last)
}
