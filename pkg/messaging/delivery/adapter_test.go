package delivery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

var (
	_ transport.MessagingAPI     = (*Adapter)(nil)
	_ transport.ContentTopicOnly = (*Adapter)(nil)
)

const testHash = "0x0102030405060708091011121314151617181920212223242526272829303132"

type fakeClient struct {
	mu         sync.Mutex
	events     chan Event
	subscribed map[string]int
	sendID     string
	peers      types.PeerStats
}

func newFakeClient() *fakeClient {
	return &fakeClient{events: make(chan Event, 16), subscribed: make(map[string]int), sendID: "0a1b2c3d4e5f60718293"}
}

func (f *fakeClient) Start() error         { return nil }
func (f *fakeClient) Stop() error          { return nil }
func (f *fakeClient) Close() error         { close(f.events); return nil }
func (f *fakeClient) Events() <-chan Event { return f.events }

func (f *fakeClient) Subscribe(contentTopic string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subscribed[contentTopic]++
	return nil
}

func (f *fakeClient) Unsubscribe(contentTopic string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subscribed[contentTopic]--
	return nil
}

func (f *fakeClient) Send(ctx context.Context, contentTopic string, payload []byte, ephemeral bool) (string, error) {
	return f.sendID, nil
}

func (f *fakeClient) ConnectedPeers() (types.PeerStats, error) { return f.peers, nil }
func (f *fakeClient) PeerID() (peer.ID, error)                 { return "16Uiu2HAmTest", nil }
func (f *fakeClient) Metrics() (string, error)                 { return "metric 1", nil }
func (f *fakeClient) MaxMessageSize() (uint64, error)          { return 150 * 1024, nil }

func (f *fakeClient) subscriptions(contentTopic string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.subscribed[contentTopic]
}

func startAdapter(t *testing.T) (*Adapter, *fakeClient) {
	client := newFakeClient()
	a := NewAdapter(client, zap.NewNop())
	require.NoError(t, a.Start())
	t.Cleanup(func() { _ = a.Stop() })
	return a, client
}

func nextEvent(t *testing.T, ch <-chan types.EnvelopeEvent) types.EnvelopeEvent {
	select {
	case ev := <-ch:
		return ev
	case <-time.After(time.Second):
		t.Fatal("no envelope event")
		return types.EnvelopeEvent{}
	}
}

func TestSendReturnsWithoutWaitingForAnOutcome(t *testing.T) {
	a, _ := startAdapter(t)

	hash, err := a.Send(context.Background(), "", "/waku/1/0x01020304/rfc26", []byte("hi"), false, nil)
	require.NoError(t, err)
	require.Equal(t, requestKey("0a1b2c3d4e5f60718293"), hash)
}

func TestRequestKeyIsThePaddedRequestID(t *testing.T) {
	require.Equal(t, append(make([]byte, 22), 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f, 0x60, 0x71, 0x82, 0x93), requestKey("0a1b2c3d4e5f60718293"))
	require.Nil(t, requestKey("not-hex"))
}

func TestSendOutcomesBecomeEnvelopeEvents(t *testing.T) {
	a, client := startAdapter(t)
	events := make(chan types.EnvelopeEvent, 4)
	sub := a.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	client.events <- SentEvent{RequestID: "0a1b2c3d4e5f60718293", MessageHash: testHash}
	client.events <- PropagatedEvent{RequestID: "0a1b2c3d4e5f60718293", MessageHash: testHash}
	ev := nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeSent, ev.Event)
	require.Equal(t, requestKey("0a1b2c3d4e5f60718293"), ev.Hash.Bytes())
	require.Equal(t, cryptotypes.BytesToHash(decodeHash(testHash)), ev.Data)

	client.events <- ErrorEvent{RequestID: "93827160f5e4d3c2b1a0", MessageHash: testHash, Err: "retry window elapsed"}
	ev = nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeExpired, ev.Event)
	require.Equal(t, requestKey("93827160f5e4d3c2b1a0"), ev.Hash.Bytes())
}

func TestReceivedMessageIsReportedWithoutPubsubTopic(t *testing.T) {
	a, client := startAdapter(t)
	topic := types.BytesToTopic([]byte{1, 2, 3, 4})
	require.NoError(t, a.Subscribe(context.Background(), "/waku/2/rs/16/32", []types.TopicType{topic}))

	events := make(chan types.EnvelopeEvent, 4)
	sub := a.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	client.events <- ReceivedEvent{MessageHash: testHash, ContentTopic: topic.ContentTopic(), Payload: []byte("hi"), Timestamp: 7}

	ev := nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeAvailable, ev.Event)
	msg := ev.Data.(*types.ReceivedMessage)
	require.Equal(t, []byte("hi"), msg.Payload)
	require.Equal(t, decodeHash(testHash), msg.Hash)
	require.Empty(t, msg.PubsubTopic)

	select {
	case ev := <-events:
		t.Fatalf("unexpected second event: %+v", ev)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSubscriptionsGoStraightToTheLibrary(t *testing.T) {
	a, client := startAdapter(t)
	topic := types.BytesToTopic([]byte{1, 2, 3, 4})
	ctx := context.Background()

	require.NoError(t, a.Subscribe(ctx, "", []types.TopicType{topic}))
	require.Equal(t, 1, client.subscriptions(topic.ContentTopic()))

	require.NoError(t, a.Unsubscribe(ctx, "", []types.TopicType{topic}))
	require.Equal(t, 0, client.subscriptions(topic.ContentTopic()))
}

func TestStopIsIdempotent(t *testing.T) {
	a, _ := startAdapter(t)
	require.NoError(t, a.Stop())
	require.NoError(t, a.Stop())
}

func TestNodeInfoComesFromTheClient(t *testing.T) {
	a, client := startAdapter(t)
	client.peers = types.PeerStats{"16Uiu2HAmPeer": types.WakuV2Peer{}}

	require.Equal(t, client.peers, a.Peers())
	require.Equal(t, peer.ID("16Uiu2HAmTest"), a.PeerID())
	require.Equal(t, "metric 1", a.Metrics())
	require.Equal(t, uint32(150*1024), a.MaxMessageSize())
}

func TestConnectionStatusIsTracked(t *testing.T) {
	a, client := startAdapter(t)
	sub, err := a.SubscribeToConnStatusChanges()
	require.NoError(t, err)

	client.events <- ConnectionEvent{Status: PartiallyConnected}

	select {
	case status := <-sub.C:
		require.Equal(t, types.ConnectionStatePartiallyConnected, status.State)
		require.True(t, status.IsOnline)
	case <-time.After(time.Second):
		t.Fatal("no connection status")
	}
	require.Equal(t, types.ConnectionStatePartiallyConnected, a.ConnectionState())
}
