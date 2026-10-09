package delivery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

var _ transport.MessagingAPI = (*Adapter)(nil)

const testHash = "0x0102030405060708091011121314151617181920212223242526272829303132"

type fakeClient struct {
	mu           sync.Mutex
	events       chan Event
	subscribed   map[string]int
	sendID       string
	beforeReturn func()
}

func newFakeClient() *fakeClient {
	return &fakeClient{events: make(chan Event, 16), subscribed: make(map[string]int), sendID: "req-1"}
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
	if f.beforeReturn != nil {
		f.beforeReturn()
	}
	return f.sendID, nil
}

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

func TestSendReturnsHashFromSentEvent(t *testing.T) {
	a, client := startAdapter(t)

	go func() {
		time.Sleep(20 * time.Millisecond)
		client.events <- SentEvent{RequestID: "req-1", MessageHash: testHash}
	}()

	hash, err := a.Send(context.Background(), "", "/waku/1/0x01020304/rfc26", []byte("hi"), false, nil)
	require.NoError(t, err)
	require.Equal(t, decodeHash(testHash), hash)
}

func TestSendReturnsHashFromPropagatedEvent(t *testing.T) {
	a, client := startAdapter(t)

	go func() {
		time.Sleep(20 * time.Millisecond)
		client.events <- PropagatedEvent{RequestID: "req-1", MessageHash: testHash}
	}()

	hash, err := a.Send(context.Background(), "", "/waku/1/0x01020304/rfc26", []byte("hi"), false, nil)
	require.NoError(t, err)
	require.Equal(t, decodeHash(testHash), hash)
}

func TestSendReturnsHashWhenEventBeatsRequestID(t *testing.T) {
	a, client := startAdapter(t)

	client.beforeReturn = func() {
		client.events <- SentEvent{RequestID: "req-1", MessageHash: testHash}
		require.Eventually(t, func() bool {
			a.sendMu.Lock()
			defer a.sendMu.Unlock()
			_, ok := a.early["req-1"]
			return ok
		}, time.Second, time.Millisecond)
	}

	hash, err := a.Send(context.Background(), "", "/waku/1/0x01020304/rfc26", []byte("hi"), false, nil)
	require.NoError(t, err)
	require.Equal(t, decodeHash(testHash), hash)
}

func TestSendFailsOnErrorEvent(t *testing.T) {
	a, client := startAdapter(t)

	go func() {
		time.Sleep(20 * time.Millisecond)
		client.events <- ErrorEvent{RequestID: "req-1", MessageHash: testHash, Err: "send queue full"}
	}()

	_, err := a.Send(context.Background(), "", "/waku/1/0x01020304/rfc26", []byte("hi"), false, nil)
	require.ErrorContains(t, err, "send queue full")
}

func TestSendTimesOutWithoutEvent(t *testing.T) {
	a, _ := startAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := a.Send(ctx, "", "/waku/1/0x01020304/rfc26", []byte("hi"), false, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSendOutcomesBecomeEnvelopeEvents(t *testing.T) {
	a, client := startAdapter(t)
	events := make(chan types.EnvelopeEvent, 4)
	sub := a.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	client.events <- PropagatedEvent{RequestID: "req-1", MessageHash: testHash}
	ev := nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeSent, ev.Event)
	require.Equal(t, decodeHash(testHash), ev.Hash.Bytes())

	client.events <- ErrorEvent{RequestID: "req-2", MessageHash: testHash, Err: "retry window elapsed"}
	ev = nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeExpired, ev.Event)
}

func TestReceivedMessageIsReportedOnEverySubscribedPubsubTopic(t *testing.T) {
	a, client := startAdapter(t)
	topic := types.BytesToTopic([]byte{1, 2, 3, 4})

	require.NoError(t, a.Subscribe(context.Background(), "/waku/2/rs/16/32", []types.TopicType{topic}))
	require.NoError(t, a.Subscribe(context.Background(), "/waku/2/rs/16/64", []types.TopicType{topic}))
	require.Equal(t, 1, client.subscriptions(topic.ContentTopic()))

	events := make(chan types.EnvelopeEvent, 4)
	sub := a.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	client.events <- ReceivedEvent{MessageHash: testHash, ContentTopic: topic.ContentTopic(), Payload: []byte("hi"), Timestamp: 7}

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		ev := nextEvent(t, events)
		require.Equal(t, types.EventEnvelopeAvailable, ev.Event)
		msg := ev.Data.(*types.ReceivedMessage)
		require.Equal(t, []byte("hi"), msg.Payload)
		require.Equal(t, decodeHash(testHash), msg.Hash)
		got[msg.PubsubTopic] = true
	}
	require.Equal(t, map[string]bool{"/waku/2/rs/16/32": true, "/waku/2/rs/16/64": true}, got)
}

func TestUnsubscribeKeepsTopicWhileAnotherPubsubTopicWantsIt(t *testing.T) {
	a, client := startAdapter(t)
	topic := types.BytesToTopic([]byte{1, 2, 3, 4})
	ctx := context.Background()

	require.NoError(t, a.Subscribe(ctx, "/waku/2/rs/16/32", []types.TopicType{topic}))
	require.NoError(t, a.Subscribe(ctx, "/waku/2/rs/16/64", []types.TopicType{topic}))

	require.NoError(t, a.Unsubscribe(ctx, "/waku/2/rs/16/64", []types.TopicType{topic}))
	require.Equal(t, 1, client.subscriptions(topic.ContentTopic()))

	require.NoError(t, a.Unsubscribe(ctx, "/waku/2/rs/16/32", []types.TopicType{topic}))
	require.Equal(t, 0, client.subscriptions(topic.ContentTopic()))
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
