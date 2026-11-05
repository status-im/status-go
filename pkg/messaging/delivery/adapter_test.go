package delivery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"
	wakupb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	"github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

var (
	_ transport.MessagingAPI     = (*Adapter)(nil)
	_ transport.ContentTopicOnly = (*Adapter)(nil)
)

const testHash = "0x0102030405060708091011121314151617181920212223242526272829303132"

func subscribeEvents(t *testing.T, a *Adapter) <-chan types.EnvelopeEvent {
	events := make(chan types.EnvelopeEvent, 4)
	sub := a.SubscribeEnvelopeEvents(events)
	t.Cleanup(sub.Unsubscribe)
	return events
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

func TestSendOutcomesBecomeEnvelopeEvents(t *testing.T) {
	a := newAdapter(nil, nil, zap.NewNop())
	events := subscribeEvents(t, a)

	a.handleEvent(messaging.MessageSentEvent{RequestID: "0a1b2c3d4e5f60718293", MessageHash: testHash})
	a.handleEvent(messaging.MessagePropagatedEvent{RequestID: "0a1b2c3d4e5f60718293", MessageHash: testHash})
	ev := nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeSent, ev.Event)
	require.Equal(t, types.RequestID("0a1b2c3d4e5f60718293"), ev.RequestID)
	require.Equal(t, cryptotypes.BytesToHash(decodeHash(testHash)), ev.Hash)

	a.handleEvent(messaging.MessageErrorEvent{RequestID: "93827160f5e4d3c2b1a0", MessageHash: testHash, Err: "retry window elapsed"})
	ev = nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeExpired, ev.Event)
	require.Equal(t, types.RequestID("93827160f5e4d3c2b1a0"), ev.RequestID)
}

func TestReceivedMessageIsReportedWithoutPubsubTopic(t *testing.T) {
	a := newAdapter(nil, nil, zap.NewNop())
	events := subscribeEvents(t, a)
	topic := types.BytesToTopic([]byte{1, 2, 3, 4})

	a.handleEvent(messaging.MessageReceivedEvent{
		MessageHash: testHash,
		Message:     messaging.Message{ContentTopic: topic.ContentTopic(), Payload: []byte("hi"), Timestamp: 7},
	})

	ev := nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeAvailable, ev.Event)
	msg := ev.Data.(*types.ReceivedMessage)
	require.Equal(t, []byte("hi"), msg.Payload)
	require.Equal(t, decodeHash(testHash), msg.Hash)
	require.Empty(t, msg.PubsubTopic)
	require.Empty(t, events)
}

func TestConnectionStatusIsTracked(t *testing.T) {
	a := newAdapter(nil, nil, zap.NewNop())
	sub, err := a.SubscribeToConnStatusChanges()
	require.NoError(t, err)

	go a.handleEvent(messaging.ConnectionStatusEvent{Status: messaging.PartiallyConnected})

	select {
	case status := <-sub.C:
		require.Equal(t, types.ConnectionStatePartiallyConnected, status.State)
		require.True(t, status.IsOnline)
	case <-time.After(time.Second):
		t.Fatal("no connection status")
	}
	require.Eventually(t, func() bool {
		return a.ConnectionState() == types.ConnectionStatePartiallyConnected
	}, time.Second, time.Millisecond)
}

// fakeStoreRequestor serves pages in order, each with a cursor to the next.
type fakeStoreRequestor struct {
	mu       sync.Mutex
	pages    [][]*storepb.WakuMessageKeyValue
	requests []*storepb.StoreQueryRequest
}

func (f *fakeStoreRequestor) Query(_ context.Context, _ peer.AddrInfo, request *storepb.StoreQueryRequest) ([]*storepb.WakuMessageKeyValue, []byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, proto.Clone(request).(*storepb.StoreQueryRequest))
	page := len(f.requests) - 1
	if page >= len(f.pages) {
		return nil, nil, nil
	}
	var cursor []byte
	if page+1 < len(f.pages) {
		cursor = []byte{byte(page + 1)}
	}
	return f.pages[page], cursor, nil
}

func storedMessage(hash byte, payload string) *storepb.WakuMessageKeyValue {
	return &storepb.WakuMessageKeyValue{
		MessageHash: []byte{hash},
		PubsubTopic: proto.String("/waku/2/rs/16/32"),
		Message:     &wakupb.WakuMessage{ContentTopic: "/waku/1/0x01020304/rfc26", Payload: []byte(payload), Timestamp: proto.Int64(7)},
	}
}

func TestFetchMessagesByHashesFollowsCursorsAndReportsMessages(t *testing.T) {
	requestor := &fakeStoreRequestor{pages: [][]*storepb.WakuMessageKeyValue{{storedMessage(1, "a")}, {storedMessage(2, "b")}}}
	a := newAdapter(nil, requestor, zap.NewNop())
	a.storeClient.SetStorenodes([]peer.AddrInfo{{ID: "16Uiu2HAmStore"}})
	events := subscribeEvents(t, a)

	require.NoError(t, a.FetchMessagesByHashes(context.Background(), []string{"0x01", "0x02", "not-hex"}))

	for _, want := range []string{"a", "b"} {
		ev := nextEvent(t, events)
		require.Equal(t, types.EventEnvelopeAvailable, ev.Event)
		msg := ev.Data.(*types.ReceivedMessage)
		require.Equal(t, []byte(want), msg.Payload)
		require.Equal(t, "/waku/2/rs/16/32", msg.PubsubTopic)
	}
	require.Len(t, requestor.requests, 2)
	require.Equal(t, [][]byte{{1}, {2}}, requestor.requests[0].MessageHashes)
	require.Nil(t, requestor.requests[0].PaginationCursor)
	require.Equal(t, []byte{1}, requestor.requests[1].PaginationCursor)
}

func TestStoreQueryWithoutStorenodesFails(t *testing.T) {
	a := newAdapter(nil, &fakeStoreRequestor{}, zap.NewNop())
	err := a.StoreQuery(context.Background(), types.MailserverBatch{}, 10, nil, false)
	require.ErrorIs(t, err, waku.ErrNoStorenodesReachable)
}
