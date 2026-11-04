package lightpush

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"golang.org/x/time/rate"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func makeWakuRelay(t *testing.T, pusubTopic string) (*relay.WakuRelay, *relay.Subscription, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	b := relay.NewBroadcaster(10)
	require.NoError(t, b.Start(context.Background()))
	relay := relay.NewWakuRelay(b, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	require.NoError(t, err)
	err = relay.Start(context.Background())
	require.NoError(t, err)

	sub, err := relay.Subscribe(context.Background(), protocol.NewContentFilter(pusubTopic))
	require.NoError(t, err)

	return relay, sub[0], host
}

// Node1: Relay
// Node2: Relay+Lightpush
// Node3: Relay+Lightpush

// Client that will lightpush a message
//
// Node1 and Node 2 are peers
// Node1 and Node 3 are peers
// Client and Node 2 are peers
// Client and Node 3 are peers
// Client will use lightpush request, sending the message to Node2 and Node3
//
// Client send a successful message using lightpush
// Node2, Node3 receive the message and broadcast it
// Node1 receive the messages
func TestWakuLightPush(t *testing.T) {
	testTopic := "/waku/2/go/lightpush/test"
	node1, sub1, host1 := makeWakuRelay(t, testTopic)
	defer node1.Stop()
	defer sub1.Unsubscribe()

	node2, sub2, host2 := makeWakuRelay(t, testTopic)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	ctx := context.Background()
	lightPushNode2 := NewWakuLightPush(node2, nil, prometheus.DefaultRegisterer, utils.Logger())
	lightPushNode2.SetHost(host2)
	err := lightPushNode2.Start(ctx)
	require.NoError(t, err)
	defer lightPushNode2.Stop()

	node3, sub3, host3 := makeWakuRelay(t, testTopic)
	defer node3.Stop()
	defer sub3.Unsubscribe()

	lightPushNode3 := NewWakuLightPush(node3, nil, prometheus.DefaultRegisterer, utils.Logger())
	lightPushNode3.SetHost(host3)
	err = lightPushNode3.Start(ctx)
	require.NoError(t, err)
	defer lightPushNode3.Stop()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	clientHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)

	err = host2.Connect(ctx, host2.Peerstore().PeerInfo(host1.ID()))
	require.NoError(t, err)

	host3.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host3.Peerstore().AddProtocols(host1.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)

	err = host3.Connect(ctx, host3.Peerstore().PeerInfo(host1.ID()))
	require.NoError(t, err)

	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host2.ID(), LightPushID_v20beta1)
	require.NoError(t, err)

	clientHost.Peerstore().AddAddr(host3.ID(), tests.GetHostAddress(host3), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host3.ID(), LightPushID_v20beta1)
	require.NoError(t, err)

	msg2 := tests.CreateWakuMessage("test2", utils.GetUnixEpoch())

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub1.Ch
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub2.Ch
	}()

	var lpOptions []RequestOption
	lpOptions = append(lpOptions, WithPubSubTopic(testTopic))
	lpOptions = append(lpOptions, WithPeer(host2.ID()))
	lpOptions = append(lpOptions, WithMaxPeers(2))

	// Checking that msg hash is correct
	hash, err := client.Publish(ctx, msg2, lpOptions...)
	require.NoError(t, err)
	require.Equal(t, protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), string(testTopic)).Hash(), hash)
	wg.Wait()
}

func TestWakuLightPushStartWithoutRelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientHost, err := tests.MakeHost(context.Background(), 0, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)
	err = client.Start(ctx)

	require.Errorf(t, err, "relay is required")
}

func TestWakuLightPushNoPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testTopic := "abc"

	clientHost, err := tests.MakeHost(context.Background(), 0, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)
	var lpOptions []RequestOption
	lpOptions = append(lpOptions, WithPubSubTopic(testTopic))

	_, err = client.Publish(ctx, tests.CreateWakuMessage("test", utils.GetUnixEpoch()), lpOptions...)
	require.Errorf(t, err, "no suitable remote peers")
}

// Node1: Relay
// Node2: Relay+Lightpush
// Client that will lightpush a message
//
// Node1 and Node 2 are peers
// Client and Node 2 are peers
// Client will use lightpush request, sending the message to Node2
//
// Client send a successful message using lightpush
// Node2 receive the message and broadcast it
// Node1 receive the message

func TestWakuLightPushAutoSharding(t *testing.T) {
	contentTopic := "0/test/1/testTopic/proto"
	cTopic1, err := protocol.StringToContentTopic(contentTopic)
	require.NoError(t, err)
	//Computing pubSubTopic only for filterFullNode.
	pubSubTopicInst := protocol.GetShardFromContentTopic(cTopic1, protocol.GenerationZeroShardsCount)
	pubSubTopic := pubSubTopicInst.String()
	node1, sub1, host1 := makeWakuRelay(t, pubSubTopic)
	defer node1.Stop()
	defer sub1.Unsubscribe()

	node2, sub2, host2 := makeWakuRelay(t, pubSubTopic)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	ctx := context.Background()
	lightPushNode2 := NewWakuLightPush(node2, nil, prometheus.DefaultRegisterer, utils.Logger())
	lightPushNode2.SetHost(host2)
	err = lightPushNode2.Start(ctx)
	require.NoError(t, err)
	defer lightPushNode2.Stop()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	clientHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)

	err = host2.Connect(ctx, host2.Peerstore().PeerInfo(host1.ID()))
	require.NoError(t, err)

	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host2.ID(), LightPushID_v20beta1)
	require.NoError(t, err)

	msg1 := tests.CreateWakuMessage(contentTopic, utils.GetUnixEpoch())

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub1.Ch
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub2.Ch

	}()
	var lpOptions []RequestOption
	lpOptions = append(lpOptions, WithPeer(host2.ID()))
	// Verifying successful request
	hash1, err := client.Publish(ctx, msg1, lpOptions...)
	require.NoError(t, err)
	require.Equal(t, protocol.NewEnvelope(msg1, *utils.GetUnixEpoch(), string(pubSubTopic)).Hash(), hash1)

	wg.Wait()

}

func TestWakuLightPushCornerCases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testTopic := "/waku/2/go/lightpush/test"
	testContentTopic := "/test/10/my-lp-app/proto"

	// Prepare peer manager instance to include in test
	pm := peermanager.NewPeerManager(10, 10, nil, nil, true, utils.Logger())

	node1, sub1, host1 := makeWakuRelay(t, testTopic)
	defer node1.Stop()
	defer sub1.Unsubscribe()

	node2, sub2, host2 := makeWakuRelay(t, testTopic)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	lightPushNode2 := NewWakuLightPush(node2, pm, prometheus.DefaultRegisterer, utils.Logger(), WithRateLimiter(rate.Inf, 0))
	lightPushNode2.SetHost(host2)
	err := lightPushNode2.Start(ctx)
	require.NoError(t, err)
	defer lightPushNode2.Stop()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	clientHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)

	err = host2.Connect(ctx, host2.Peerstore().PeerInfo(host1.ID()))
	require.NoError(t, err)

	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host2.ID(), LightPushID_v20beta1)
	require.NoError(t, err)

	msg2 := tests.CreateWakuMessage(testContentTopic, utils.GetUnixEpoch())

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)

	var wg sync.WaitGroup

	var lpOptions []RequestOption
	lpOptions = append(lpOptions, WithPubSubTopic(testTopic))
	lpOptions = append(lpOptions, WithPeer(host2.ID()))

	// Check that msg publish has passed for nominal case
	_, err = client.Publish(ctx, msg2, lpOptions...)
	require.NoError(t, err)

	// Wait for the nominal case message at node1
	tests.WaitForMsg(t, 2*time.Second, &wg, sub1.Ch)

	// Test error case with nil message
	_, err = client.Publish(ctx, nil, lpOptions...)
	require.Error(t, err)

	// Create new "dummy" host - not related to any node
	host3, err := tests.MakeHost(context.Background(), 12345, rand.Reader)
	require.NoError(t, err)

	var lpOptions2 []RequestOption

	// Test error case with empty options
	_, err = client.Publish(ctx, msg2, lpOptions2...)
	require.Error(t, err)

	// Test error case with unrelated host
	_, err = client.Publish(ctx, msg2, WithPubSubTopic(testTopic), WithPeer(host3.ID()))
	require.Error(t, err)

	// Test corner case with default pubSub topic
	_, err = client.Publish(ctx, msg2, WithDefaultPubsubTopic(), WithPeer(host2.ID()))
	require.Error(t, err)
	require.Equal(t, "lightpush errorCould not publish message: cannot publish to unsubscribed topic", err.Error())

	// Test situation when cancel func is nil
	lightPushNode2.cancel = nil
}

func TestWakuLightPushWithStaticSharding(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare pubsub topic for static sharding
	pubSubTopic := protocol.NewStaticShardingPubsubTopic(uint16(25), uint16(0)).String()
	testContentTopic := "/test/10/my-lp-app/proto"

	// Node topology: clientNode (lightpush client) <-> node2(relay+lightpush server) <-> node3(relay)
	// ClientNode
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	clientHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger(), WithRateLimiter(rate.Inf, 0))
	client.SetHost(clientHost)

	// Node2
	node2, sub2, host2 := makeWakuRelay(t, pubSubTopic)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	lightPushNode2 := NewWakuLightPush(node2, nil, prometheus.DefaultRegisterer, utils.Logger(), WithRateLimiter(rate.Inf, 0))
	lightPushNode2.SetHost(host2)
	err = lightPushNode2.Start(ctx)
	require.NoError(t, err)
	defer lightPushNode2.Stop()

	// Node3
	node3, sub3, host3 := makeWakuRelay(t, pubSubTopic)
	defer node3.Stop()
	defer sub3.Unsubscribe()

	// Add path clientNode (lightpush client) -> node2(relay+lightpush server)
	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host2.ID(), LightPushID_v20beta1)
	require.NoError(t, err)

	// Add path node2(relay+lightpush server) -> node3(relay)
	host2.Peerstore().AddAddr(host3.ID(), tests.GetHostAddress(host3), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host3.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)

	err = host2.Connect(ctx, host2.Peerstore().PeerInfo(host3.ID()))
	require.NoError(t, err)

	// Create messages
	msg := tests.CreateWakuMessage(testContentTopic, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage(testContentTopic, utils.GetUnixEpoch())

	// Wait for the mesh connection to happen between nodes
	time.Sleep(2 * time.Second)

	var wg sync.WaitGroup

	// Check that msg publish has led to message deliver for existing topic
	_, err = client.Publish(ctx, msg, WithPubSubTopic(pubSubTopic), WithPeer(host2.ID()))
	require.NoError(t, err)
	tests.WaitForMsg(t, 2*time.Second, &wg, sub3.Ch)

	// Check that msg2 publish finished without message delivery for unconfigured topic
	_, err = client.Publish(ctx, msg2, WithPubSubTopic("/waku/2/rsv/25/0"), WithPeer(host2.ID()))
	require.Error(t, err)
	require.Equal(t, "lightpush errorCould not publish message: cannot publish to unsubscribed topic", err.Error())
	tests.WaitForTimeout(t, ctx, 1*time.Second, &wg, sub3.Ch)
}
