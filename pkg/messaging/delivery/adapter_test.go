package delivery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
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

func TestRequestKeyIsThePaddedRequestID(t *testing.T) {
	require.Equal(t, append(make([]byte, 22), 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f, 0x60, 0x71, 0x82, 0x93), requestKey("0a1b2c3d4e5f60718293"))
	require.Nil(t, requestKey("not-hex"))
}

func TestSendOutcomesBecomeEnvelopeEvents(t *testing.T) {
	a := newAdapter(nil, zap.NewNop())
	events := subscribeEvents(t, a)

	a.handleEvent(messaging.MessageSentEvent{RequestID: "0a1b2c3d4e5f60718293", MessageHash: testHash})
	a.handleEvent(messaging.MessagePropagatedEvent{RequestID: "0a1b2c3d4e5f60718293", MessageHash: testHash})
	ev := nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeSent, ev.Event)
	require.Equal(t, requestKey("0a1b2c3d4e5f60718293"), ev.Hash.Bytes())
	require.Equal(t, cryptotypes.BytesToHash(decodeHash(testHash)), ev.Data)

	a.handleEvent(messaging.MessageErrorEvent{RequestID: "93827160f5e4d3c2b1a0", MessageHash: testHash, Err: "retry window elapsed"})
	ev = nextEvent(t, events)
	require.Equal(t, types.EventEnvelopeExpired, ev.Event)
	require.Equal(t, requestKey("93827160f5e4d3c2b1a0"), ev.Hash.Bytes())
}

func TestReceivedMessageIsReportedWithoutPubsubTopic(t *testing.T) {
	a := newAdapter(nil, zap.NewNop())
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
	a := newAdapter(nil, zap.NewNop())
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
