package delivery

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// privateCluster is a two-node network with no preset: one auto-sharded shard,
// no discovery, and no store confirmation.
func privateCluster(extra map[string]any) map[string]any {
	overrides := map[string]any{
		"cluster-id":            16,
		"num-shards-in-network": 1,
		"listen-address":        "127.0.0.1",
		"discv5-discovery":      false,
		"reliability":           false,
		"nat":                   "none",
	}
	for k, v := range extra {
		overrides[k] = v
	}
	return overrides
}

func newTestAdapter(t *testing.T, cfg Config) *Adapter {
	client, err := NewClient(cfg)
	require.NoError(t, err)
	a := NewAdapter(client, zap.NewNop())
	require.NoError(t, a.Start())
	t.Cleanup(func() { require.NoError(t, a.Stop()) })
	return a
}

func waitFor(t *testing.T, ch <-chan types.EnvelopeEvent, want types.EventType) types.EnvelopeEvent {
	timeout := time.After(60 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Event == want {
				return ev
			}
		case <-timeout:
			t.Fatalf("no %s event", want)
		}
	}
}

func TestAdapterExchangesMessagesOverLiblogosdelivery(t *testing.T) {
	keyA, err := crypto.GenerateKey()
	require.NoError(t, err)
	pubA, err := libp2pcrypto.UnmarshalSecp256k1PublicKey(crypto.CompressPubkey(&keyA.PublicKey))
	require.NoError(t, err)
	idA, err := peer.IDFromPublicKey(pubA)
	require.NoError(t, err)
	portA := freePort(t)

	a := newTestAdapter(t, Config{
		Mode:      ModeCore,
		DataDir:   t.TempDir(),
		NodeKey:   keyA,
		TCPPort:   portA,
		Overrides: privateCluster(nil),
	})
	b := newTestAdapter(t, Config{
		Mode:    ModeCore,
		DataDir: t.TempDir(),
		Overrides: privateCluster(map[string]any{
			"tcp-port":   freePort(t),
			"entry-node": []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", portA, idA)},
		}),
	})

	topic := types.BytesToTopic([]byte{0xde, 0xad, 0xbe, 0xef})
	pubsubTopic := "/waku/2/rs/16/32"

	eventsA := make(chan types.EnvelopeEvent, 100)
	subA := a.SubscribeEnvelopeEvents(eventsA)
	defer subA.Unsubscribe()
	eventsB := make(chan types.EnvelopeEvent, 100)
	subB := b.SubscribeEnvelopeEvents(eventsB)
	defer subB.Unsubscribe()

	require.NoError(t, a.Subscribe(context.Background(), pubsubTopic, []types.TopicType{topic}))
	require.NoError(t, b.Subscribe(context.Background(), pubsubTopic, []types.TopicType{topic}))

	require.Eventually(t, func() bool {
		return a.ConnectionState().IsOnline() && b.ConnectionState().IsOnline()
	}, 60*time.Second, 200*time.Millisecond, "nodes did not connect")

	// A fresh relay mesh needs a heartbeat or two before it carries messages.
	keys := map[cryptotypes.Hash]bool{}
	var received types.EnvelopeEvent
	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		key, err := b.Send(ctx, pubsubTopic, topic.ContentTopic(), []byte("hello from b"), false, nil)
		require.NoError(t, err)
		keys[cryptotypes.BytesToHash(key)] = true
		select {
		case received = <-eventsA:
			return received.Event == types.EventEnvelopeAvailable
		case <-time.After(5 * time.Second):
			return false
		}
	}, 90*time.Second, time.Second, "a never received b's message")

	msg := received.Data.(*types.ReceivedMessage)
	require.Equal(t, []byte("hello from b"), msg.Payload)
	require.Equal(t, topic.ContentTopic(), msg.ContentTopic)
	require.Empty(t, msg.PubsubTopic)

	wireHash := cryptotypes.BytesToHash(msg.Hash)
	for {
		sent := waitFor(t, eventsB, types.EventEnvelopeSent)
		if sent.Data == wireHash {
			require.True(t, keys[sent.Hash], "the sent event must carry a key Send returned")
			break
		}
	}
}
