package node

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"

	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func createTestMsg(version uint32) *pb.WakuMessage {
	message := new(pb.WakuMessage)
	message.Payload = []byte{0, 1, 2}
	message.Version = proto.Uint32(version)
	message.Timestamp = proto.Int64(123456)
	message.ContentTopic = "abc"
	return message
}

func TestWakuNode2(t *testing.T) {
	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")

	key, err := tests.RandomHex(32)
	require.NoError(t, err)
	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	ctx := context.Background()

	wakuNode, err := New(
		WithPrivateKey(prvKey),
		WithHostAddress(hostAddr),
		WithWakuRelay(),
	)
	require.NoError(t, err)

	err = wakuNode.Start(ctx)
	require.NoError(t, err)

	_, err = wakuNode.Relay().Subscribe(ctx, protocol.NewContentFilter("waku/rs/1/1"))
	require.NoError(t, err)
	time.Sleep(time.Second * 1)

	err = wakuNode.Relay().Unsubscribe(ctx, protocol.NewContentFilter("waku/rs/1/1"))
	require.NoError(t, err)

	defer wakuNode.Stop()
}

func int2Bytes(i int) []byte {
	if i > 0 {
		return append(big.NewInt(int64(i)).Bytes(), byte(1))
	}
	return append(big.NewInt(int64(i)).Bytes(), byte(0))
}

func TestUpAndDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	hostAddr1, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key1, _ := tests.RandomHex(32)
	prvKey1, _ := crypto.HexToECDSA(key1)

	nodes, err := dnsdisc.RetrieveNodes(context.Background(), "enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im")
	require.NoError(t, err)

	var bootnodes []*enode.Node
	for _, n := range nodes {
		if n.ENR != nil {
			bootnodes = append(bootnodes, n.ENR)
		}
	}

	wakuNode1, err := New(
		WithPrivateKey(prvKey1),
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithDiscoveryV5(0, bootnodes, true),
	)

	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		utils.Logger().Info("Starting...", zap.Int("iteration", i))
		err = wakuNode1.Start(ctx)
		require.NoError(t, err)
		err = wakuNode1.DiscV5().Start(ctx)
		require.NoError(t, err)
		utils.Logger().Info("Started!", zap.Int("iteration", i))
		time.Sleep(3 * time.Second)
		utils.Logger().Info("Stopping...", zap.Int("iteration", i))
		wakuNode1.Stop()
		utils.Logger().Info("Stopped!", zap.Int("iteration", i))
	}
}

func Test500(t *testing.T) {
	maxMsgs := 500
	maxMsgBytes := int2Bytes(maxMsgs)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hostAddr1, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key1, _ := tests.RandomHex(32)
	prvKey1, _ := crypto.HexToECDSA(key1)

	hostAddr2, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key2, _ := tests.RandomHex(32)
	prvKey2, _ := crypto.HexToECDSA(key2)

	wakuNode1, err := New(
		WithPrivateKey(prvKey1),
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = wakuNode1.Start(ctx)
	require.NoError(t, err)
	defer wakuNode1.Stop()

	wakuNode2, err := New(
		WithPrivateKey(prvKey2),
		WithHostAddress(hostAddr2),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = wakuNode2.Start(ctx)
	require.NoError(t, err)
	defer wakuNode2.Stop()

	err = wakuNode2.DialPeer(ctx, wakuNode1.ListenAddresses()[0].String())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	sub1, err := wakuNode1.Relay().Subscribe(ctx, protocol.NewContentFilter(relay.DefaultWakuTopic))
	require.NoError(t, err)
	sub2, err := wakuNode2.Relay().Subscribe(ctx, protocol.NewContentFilter(relay.DefaultWakuTopic))
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()

		ticker := time.NewTimer(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				require.Fail(t, "Timeout Sub1")
			case msg := <-sub1[0].Ch:
				if msg == nil {
					return
				}
				if bytes.Equal(msg.Message().Payload, maxMsgBytes) {
					return
				}
			}
		}
	}()

	go func() {
		defer wg.Done()

		ticker := time.NewTimer(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				require.Fail(t, "Timeout Sub2")
			case msg := <-sub2[0].Ch:
				if msg == nil {
					return
				}
				if bytes.Equal(msg.Message().Payload, maxMsgBytes) {
					return
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 1; i <= maxMsgs; i++ {
			msg := createTestMsg(0)
			msg.Payload = int2Bytes(i)
			msg.Timestamp = proto.Int64(int64(i))
			if _, err := wakuNode2.Relay().Publish(ctx, msg, relay.WithDefaultPubsubTopic()); err != nil {
				require.Fail(t, "Could not publish all messages")
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()

}

func TestDecoupledStoreFromRelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// NODE1: Relay Node + Filter Server
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode1, err := New(
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithWakuFilterFullNode(),
	)
	require.NoError(t, err)
	err = wakuNode1.Start(ctx)
	require.NoError(t, err)
	defer wakuNode1.Stop()

	subs, err := wakuNode1.Relay().Subscribe(ctx, protocol.NewContentFilter(relay.DefaultWakuTopic))
	require.NoError(t, err)
	defer subs[0].Unsubscribe()

	// NODE2: Filter Client/Store
	db, err := sqlite.NewDB(":memory:", utils.Logger())
	require.NoError(t, err)
	dbStore, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(db), persistence.WithMigrations(sqlite.Migrations))
	require.NoError(t, err)

	hostAddr2, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode2, err := New(
		WithHostAddress(hostAddr2),
		WithWakuFilterLightNode(),
		WithWakuStore(),
		WithMessageProvider(dbStore),
	)
	require.NoError(t, err)
	err = wakuNode2.Start(ctx)
	require.NoError(t, err)
	defer wakuNode2.Stop()

	peerID, err := wakuNode2.AddPeer(wakuNode1.ListenAddresses(), peerstore.Static, []string{relay.DefaultWakuTopic}, filter.FilterSubscribeID_v20beta1)
	require.NoError(t, err)

	subscription, err := wakuNode2.FilterLightnode().Subscribe(ctx, protocol.ContentFilter{
		PubsubTopic:   relay.DefaultWakuTopic,
		ContentTopics: protocol.NewContentTopicSet("abc"),
	}, filter.WithPeer(peerID))
	require.NoError(t, err)

	// Sleep to make sure the filter is subscribed
	time.Sleep(1 * time.Second)

	// Send MSG1 on NODE1
	msg := createTestMsg(0)
	msg.Payload = []byte{1, 2, 3, 4, 5}
	msg.Timestamp = utils.GetUnixEpoch()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// MSG1 should be pushed in NODE2 via filter
		defer wg.Done()
		env, ok := <-subscription[0].C
		if !ok {
			require.Fail(t, "no message")
		}
		require.Equal(t, msg.Timestamp, env.Message().Timestamp)
	}()

	time.Sleep(500 * time.Millisecond)

	if _, err := wakuNode1.Relay().Publish(ctx, msg, relay.WithDefaultPubsubTopic()); err != nil {
		require.Fail(t, "Could not publish all messages")
	}

	wg.Wait()

	// NODE3: Query from NODE2
	hostAddr3, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode3, err := New(
		WithHostAddress(hostAddr3),
		WithWakuFilterLightNode(),
	)
	require.NoError(t, err)
	err = wakuNode3.Start(ctx)
	require.NoError(t, err)
	defer wakuNode3.Stop()

	_, err = wakuNode3.AddPeer(wakuNode2.ListenAddresses(), peerstore.Static, []string{relay.DefaultWakuTopic}, legacy_store.StoreID_v20beta4)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)
	// NODE2 should have returned the message received via filter
	result, err := wakuNode3.LegacyStore().Query(ctx, legacy_store.Query{})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
	require.Equal(t, msg.Timestamp, result.Messages[0].Timestamp)
}

func TestStaticShardingMultipleTopics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testClusterID := uint16(20)

	// Node1 with Relay
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode1, err := New(
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithClusterID(testClusterID),
	)
	require.NoError(t, err)
	err = wakuNode1.Start(ctx)
	require.NoError(t, err)
	defer wakuNode1.Stop()

	pubSubTopic1 := protocol.NewStaticShardingPubsubTopic(testClusterID, uint16(0))
	pubSubTopic1Str := pubSubTopic1.String()
	contentTopic1 := "/test/2/my-app/sharded"

	pubSubTopic2 := protocol.NewStaticShardingPubsubTopic(testClusterID, uint16(10))
	pubSubTopic2Str := pubSubTopic2.String()
	contentTopic2 := "/test/3/my-app/sharded"

	require.Equal(t, testClusterID, wakuNode1.ClusterID())

	r := wakuNode1.Relay()

	subs1, err := r.Subscribe(ctx, protocol.NewContentFilter(pubSubTopic1Str, contentTopic1))
	require.NoError(t, err)

	subs2, err := r.Subscribe(ctx, protocol.NewContentFilter(pubSubTopic2Str, contentTopic2))
	require.NoError(t, err)

	require.NotEqual(t, subs1[0].ID, subs2[0].ID)

	require.True(t, r.IsSubscribed(pubSubTopic1Str))
	require.True(t, r.IsSubscribed(pubSubTopic2Str))

	s1, err := r.GetSubscriptionWithPubsubTopic(pubSubTopic1Str, contentTopic1)
	require.NoError(t, err)
	s2, err := r.GetSubscriptionWithPubsubTopic(pubSubTopic2Str, contentTopic2)
	require.NoError(t, err)
	require.Equal(t, s1.ID, subs1[0].ID)
	require.Equal(t, s2.ID, subs2[0].ID)

	// Wait for subscriptions
	time.Sleep(1 * time.Second)

	// Send message to subscribed topic
	msg := tests.CreateWakuMessage(contentTopic1, utils.GetUnixEpoch(), "test message")

	_, err = r.Publish(ctx, msg, relay.WithPubSubTopic(pubSubTopic1Str))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(1)
	// Message msg could be retrieved
	go func() {
		defer wg.Done()
		env, ok := <-subs1[0].Ch
		require.True(t, ok, "no message retrieved")
		require.Equal(t, msg.Timestamp, env.Message().Timestamp)
	}()

	wg.Wait()

	// Send another message to non-subscribed pubsub topic, but subscribed content topic
	msg2 := tests.CreateWakuMessage(contentTopic1, utils.GetUnixEpoch(), "test message 2")
	pubSubTopic3 := protocol.NewStaticShardingPubsubTopic(testClusterID, uint16(321))
	pubSubTopic3Str := pubSubTopic3.String()
	_, err = r.Publish(ctx, msg2, relay.WithPubSubTopic(pubSubTopic3Str))
	require.Error(t, err)

	time.Sleep(100 * time.Millisecond)

	// No message could be retrieved
	tests.WaitForTimeout(t, ctx, 1*time.Second, &wg, subs1[0].Ch)

	// Send another message to subscribed pubsub topic, but not subscribed content topic - mix it up
	msg3 := tests.CreateWakuMessage(contentTopic2, utils.GetUnixEpoch(), "test message 3")

	_, err = r.Publish(ctx, msg3, relay.WithPubSubTopic(pubSubTopic1Str))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// No message could be retrieved
	tests.WaitForTimeout(t, ctx, 1*time.Second, &wg, subs1[0].Ch)

}

func TestStaticShardingLimits(t *testing.T) {

	log := utils.Logger()

	if os.Getenv("RUN_FLAKY_TESTS") != "true" {

		log.Info("Skipping", zap.String("test", t.Name()),
			zap.String("reason", "RUN_FLAKY_TESTS environment variable is not set to true"))
		t.SkipNow()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	testClusterID := uint16(21)

	// Node1 with Relay
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	discv5UDPPort1, err := tests.FindFreeUDPPort(t, "0.0.0.0", 3)
	require.NoError(t, err)
	wakuNode1, err := New(
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithClusterID(testClusterID),
		WithDiscoveryV5(uint(discv5UDPPort1), nil, true),
	)
	require.NoError(t, err)
	err = wakuNode1.Start(ctx)
	require.NoError(t, err)
	defer wakuNode1.Stop()

	// Node2 with Relay
	hostAddr2, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	discv5UDPPort2, err := tests.FindFreeUDPPort(t, "0.0.0.0", 3)
	require.NoError(t, err)
	wakuNode2, err := New(
		WithHostAddress(hostAddr2),
		WithWakuRelay(),
		WithClusterID(testClusterID),
		WithDiscoveryV5(uint(discv5UDPPort2), []*enode.Node{wakuNode1.localNode.Node()}, true),
	)
	require.NoError(t, err)
	err = wakuNode2.Start(ctx)
	require.NoError(t, err)
	defer wakuNode2.Stop()

	err = wakuNode1.DiscV5().Start(ctx)
	require.NoError(t, err)
	err = wakuNode2.DiscV5().Start(ctx)
	require.NoError(t, err)

	// Wait for discovery
	time.Sleep(3 * time.Second)

	contentTopic1 := "/test/2/my-app/sharded"

	r1 := wakuNode1.Relay()
	r2 := wakuNode2.Relay()

	var shardedPubSubTopics []string

	// Subscribe topics related to static sharding
	for i := 0; i < 1024; i++ {
		shardedPubSubTopics = append(shardedPubSubTopics, fmt.Sprintf("/waku/2/rs/%d/%d", testClusterID, i))
		_, err = r1.Subscribe(ctx, protocol.NewContentFilter(shardedPubSubTopics[i], contentTopic1))
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Let ENR updates to finish
	time.Sleep(3 * time.Second)

	// Subscribe topics related to static sharding
	for i := 0; i < 1024; i++ {
		_, err = r2.Subscribe(ctx, protocol.NewContentFilter(shardedPubSubTopics[i], contentTopic1))
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Let ENR updates to finish
	time.Sleep(3 * time.Second)

	// Check ENR value after 1024 subscriptions
	shardsENR, err := wenr.RelaySharding(wakuNode1.ENR().Record())
	require.NoError(t, err)
	require.Equal(t, testClusterID, shardsENR.ClusterID)
	require.Equal(t, 1, len(shardsENR.ShardIDs))

	// Prepare message
	msg1 := tests.CreateWakuMessage(contentTopic1, utils.GetUnixEpoch(), "test message")

	// Select shard to publish
	randomShard := rand.Intn(1024)

	// Check both nodes are subscribed
	require.True(t, r1.IsSubscribed(shardedPubSubTopics[randomShard]))
	require.True(t, r2.IsSubscribed(shardedPubSubTopics[randomShard]))

	time.Sleep(1 * time.Second)

	// Publish on node1
	_, err = r1.Publish(ctx, msg1, relay.WithPubSubTopic(shardedPubSubTopics[randomShard]))
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	s2, err := r2.GetSubscriptionWithPubsubTopic(shardedPubSubTopics[randomShard], contentTopic1)
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Retrieve on node2
	tests.WaitForMsg(t, 2*time.Second, &wg, s2.Ch)

}

func TestPeerExchangeRatelimit(t *testing.T) {
	log := utils.Logger()

	if os.Getenv("RUN_FLAKY_TESTS") != "true" {

		log.Info("Skipping", zap.String("test", t.Name()),
			zap.String("reason", "RUN_FLAKY_TESTS environment variable is not set to true"))
		t.SkipNow()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	testClusterID := uint16(21)

	// Node1 with Relay
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode1, err := New(
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithClusterID(testClusterID),
		WithPeerExchange(peer_exchange.WithRateLimiter(1, 1)),
	)
	require.NoError(t, err)
	err = wakuNode1.Start(ctx)
	require.NoError(t, err)
	defer wakuNode1.Stop()

	// Node2 with Relay
	hostAddr2, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode2, err := New(
		WithHostAddress(hostAddr2),
		WithWakuRelay(),
		WithClusterID(testClusterID),
		WithPeerExchange(peer_exchange.WithRateLimiter(1, 1)),
	)
	require.NoError(t, err)
	err = wakuNode2.Start(ctx)
	require.NoError(t, err)
	defer wakuNode2.Stop()

	err = wakuNode2.DialPeer(ctx, wakuNode1.ListenAddresses()[0].String())
	require.NoError(t, err)

	//time.Sleep(1 * time.Second)

	err = wakuNode1.PeerExchange().Request(ctx, 1)
	require.NoError(t, err)

	err = wakuNode1.PeerExchange().Request(ctx, 1)
	require.Error(t, err)

	time.Sleep(1 * time.Second)
	err = wakuNode1.PeerExchange().Request(ctx, 1)
	require.NoError(t, err)
}
