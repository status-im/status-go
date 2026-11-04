package peermanager

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func makeWakuRelay(t *testing.T, log *zap.Logger) (*relay.WakuRelay, host.Host, relay.Broadcaster) {

	broadcaster := relay.NewBroadcaster(10)
	require.NoError(t, broadcaster.Start(context.Background()))

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	h, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	broadcaster.RegisterForAll()

	r := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(),
		prometheus.DefaultRegisterer, log)

	r.SetHost(h)

	return r, h, broadcaster
}

func makePeerManagerWithEventBus(t *testing.T, r *relay.WakuRelay, h *host.Host) (*PeerManager, event.Bus) {
	// Host 1 used by peer manager
	pm := NewPeerManager(10, 20, nil, r, true, utils.Logger())
	pm.SetHost(*h)

	// Create a new relay event bus
	relayEvtBus := r.Events()

	// Subscribe to EventBus
	err := pm.SubscribeToRelayEvtBus(relayEvtBus)
	require.NoError(t, err)

	// Register necessary protocols
	pm.RegisterWakuProtocol(relay.WakuRelayID_v200, relay.WakuRelayENRField)
	return pm, relayEvtBus
}

func emitTopicEvent(pubSubTopic string, peerID peer.ID, emitter event.Emitter, state relay.PeerTopicState) error {

	peerEvt := relay.EvtPeerTopic{
		PubsubTopic: pubSubTopic,
		PeerID:      peerID,
		State:       state,
	}

	return emitter.Emit(peerEvt)
}

func TestSubscribeToRelayEvtBus(t *testing.T) {
	log := utils.Logger()

	// Host 1
	r, h1, _ := makeWakuRelay(t, log)

	// Host 1 used by peer manager
	pm := NewPeerManager(10, 20, nil, nil, true, utils.Logger())
	pm.SetHost(h1)

	// Create a new relay event bus
	relayEvtBus := r.Events()

	// Subscribe to EventBus
	err := pm.SubscribeToRelayEvtBus(relayEvtBus)
	require.NoError(t, err)

}

func TestHandleRelayTopicSubscription(t *testing.T) {
	log := utils.Logger()
	pubSubTopic := "/waku/2/go/pm/test"
	ctx := context.Background()

	// Relay and Host
	r, h1, _ := makeWakuRelay(t, log)
	err := r.Start(ctx)
	require.NoError(t, err)

	// Peer manager with event bus
	pm, _ := makePeerManagerWithEventBus(t, r, &h1)
	pm.ctx = ctx

	// Start event loop to listen to events
	ctxEventLoop, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pm.peerEventLoop(ctxEventLoop)

	// Subscribe to Pubsub topic
	_, err = r.Subscribe(ctx, protocol.NewContentFilter(pubSubTopic))
	require.NoError(t, err)

	// Wait for event loop to call handler
	time.Sleep(200 * time.Millisecond)

	// Check Peer Manager knows about the topic
	pm.topicMutex.RLock()
	_, ok := pm.subRelayTopics[pubSubTopic]
	require.True(t, ok)
	pm.topicMutex.RUnlock()

	// UnSubscribe from Pubsub topic
	err = r.Unsubscribe(ctx, protocol.NewContentFilter(pubSubTopic))
	require.NoError(t, err)

	// Wait for event loop to call handler
	time.Sleep(200 * time.Millisecond)

	// Check the original topic was removed from Peer Manager
	pm.topicMutex.RLock()
	_, ok = pm.subRelayTopics[pubSubTopic]
	require.False(t, ok)
	pm.topicMutex.RUnlock()

	r.Stop()
}

func TestHandlePeerTopicEvent(t *testing.T) {
	log := utils.Logger()
	pubSubTopic := "/waku/2/go/pm/test"
	ctx := context.Background()

	hosts := make([]host.Host, 5)
	relays := make([]*relay.WakuRelay, 5)

	for i := 0; i < 5; i++ {
		relays[i], hosts[i], _ = makeWakuRelay(t, log)
		err := relays[i].Start(ctx)
		require.NoError(t, err)
	}

	// Create peer manager instance with the first hosts
	pm, eventBus := makePeerManagerWithEventBus(t, relays[0], &hosts[0])
	pm.ctx = ctx
	pm.RegisterWakuProtocol(relay.WakuRelayID_v200, relay.WakuRelayENRField)

	// Connect host[0] with all other hosts to reach 4 connections
	for i := 1; i < 5; i++ {
		pm.host.Peerstore().AddAddrs(hosts[i].ID(), hosts[i].Addrs(), peerstore.PermanentAddrTTL)
		err := pm.host.Connect(ctx, hosts[i].Peerstore().PeerInfo(hosts[i].ID()))
		require.NoError(t, err)
		err = pm.host.Peerstore().(wps.WakuPeerstore).SetDirection(hosts[i].ID(), network.DirOutbound)
		require.NoError(t, err)

	}

	// Wait for connections to settle
	time.Sleep(2 * time.Second)

	if len(pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic)) == 0 {
		log.Info("No peers for the topic yet")
	}

	// Subscribe to Pubsub topic on first host only
	_, err := relays[0].Subscribe(ctx, protocol.NewContentFilter(pubSubTopic))
	require.NoError(t, err)

	// Start event loop to listen to events
	ctxEventLoop := context.Background()
	go pm.peerEventLoop(ctxEventLoop)

	// Prepare emitter
	emitter, err := eventBus.Emitter(new(relay.EvtPeerTopic))
	require.NoError(t, err)

	// Send PEER_JOINED events for hosts 2-5
	for i := 1; i < 5; i++ {
		err = emitTopicEvent(pubSubTopic, hosts[i].ID(), emitter, relay.PEER_JOINED)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// Check four hosts have joined the topic
	require.Equal(t, 4, len(pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic)))

	// Check all hosts have been connected
	for _, peer := range pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic) {
		require.Equal(t, network.Connected, pm.host.Network().Connectedness(peer))
	}

	// Send PEER_LEFT events for hosts 2-5
	for i := 1; i < 5; i++ {
		err = emitTopicEvent(pubSubTopic, hosts[i].ID(), emitter, relay.PEER_LEFT)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// Check all hosts have left the topic
	require.Equal(t, 0, len(pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic)))

}
