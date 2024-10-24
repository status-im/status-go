package wakuv2

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pprotocol "github.com/libp2p/go-libp2p/core/protocol"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	ethdnsdisc "github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/wakuv2/common"
)

var testStoreENRBootstrap = "enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@store.staging.status.nodes.status.im"
var testBootENRBootstrap = "enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.staging.status.nodes.status.im"

func setDefaultConfig(config *Config, lightMode bool) {
	config.ClusterID = 16

	if lightMode {
		config.EnablePeerExchangeClient = true
		config.LightClient = true
		config.EnableDiscV5 = false
	} else {
		config.EnableDiscV5 = true
		config.EnablePeerExchangeServer = true
		config.LightClient = false
		config.EnablePeerExchangeClient = false
	}
}

func TestDiscoveryV5(t *testing.T) {
	config := &Config{}
	setDefaultConfig(config, false)
	config.DiscV5BootstrapNodes = []string{testStoreENRBootstrap}
	config.DiscoveryLimit = 20
	w, err := New(nil, "shards.staging", config, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	require.NoError(t, w.Start())

	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) == 0 {
			return errors.New("no peers discovered")
		}
		return nil
	})

	require.NoError(t, err)

	require.NotEqual(t, 0, len(w.Peers()))
	require.NoError(t, w.Stop())
}

func TestRestartDiscoveryV5(t *testing.T) {
	config := &Config{}
	setDefaultConfig(config, false)
	// Use wrong discv5 bootstrap address, to simulate being offline
	config.DiscV5BootstrapNodes = []string{"enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@1.1.1.2"}
	config.DiscoveryLimit = 20
	config.UDPPort = 10002
	config.ClusterID = 16
	w, err := New(nil, "", config, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	require.NoError(t, w.Start())
	require.False(t, w.seededBootnodesForDiscV5)

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 2 * time.Second
	}

	// Sanity check, not great, but it's probably helpful
	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) == 0 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)

	require.Error(t, err)

	w.discV5BootstrapNodes = []string{testStoreENRBootstrap}

	options = func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 90 * time.Second
	}

	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) == 0 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)
	require.NoError(t, err)

	require.True(t, w.seededBootnodesForDiscV5)
	require.NotEqual(t, 0, len(w.Peers()))
	require.NoError(t, w.Stop())
}

func TestRelayPeers(t *testing.T) {
	config := &Config{
		EnableMissingMessageVerification: true,
	}
	setDefaultConfig(config, false)
	w, err := New(nil, "", config, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	_, err = w.RelayPeersByTopic(config.DefaultShardPubsubTopic)
	require.NoError(t, err)

	// Ensure function returns an error for lightclient
	config = &Config{}
	config.ClusterID = 16
	config.LightClient = true
	w, err = New(nil, "", config, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	_, err = w.RelayPeersByTopic(config.DefaultShardPubsubTopic)
	require.Error(t, err)
}

func parseNodes(rec []string) []*enode.Node {
	var ns []*enode.Node
	for _, r := range rec {
		var n enode.Node
		if err := n.UnmarshalText([]byte(r)); err != nil {
			panic(err)
		}
		ns = append(ns, &n)
	}
	return ns
}

// In order to run these tests, you must run an nwaku node
//
// Using Docker:
//
//	IP_ADDRESS=$(hostname -I | awk '{print $1}');
//	docker run \
//	 -p 60000:60000/tcp -p 9000:9000/udp -p 8645:8645/tcp harbor.status.im/wakuorg/nwaku:v0.31.0 \
//	 --tcp-port=60000 --discv5-discovery=true --cluster-id=16 --pubsub-topic=/waku/2/rs/16/32 --pubsub-topic=/waku/2/rs/16/64 \
//	 --nat=extip:${IP_ADDRESS} --discv5-discovery --discv5-udp-port=9000 --rest-address=0.0.0.0 --store

func TestBasicWakuV2(t *testing.T) {
	nwakuInfo, err := GetNwakuInfo(nil, nil)
	require.NoError(t, err)

	// Creating a fake DNS Discovery ENRTree
	tree, url := makeTestTree("n", parseNodes([]string{nwakuInfo.EnrUri}), nil)
	enrTreeAddress := url
	envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
	if envEnrTreeAddress != "" {
		enrTreeAddress = envEnrTreeAddress
	}

	config := &Config{}
	setDefaultConfig(config, false)
	config.Port = 0
	config.Resolver = mapResolver(tree.ToTXT("n"))
	config.DiscV5BootstrapNodes = []string{enrTreeAddress}
	config.DiscoveryLimit = 20
	config.WakuNodes = []string{enrTreeAddress}
	w, err := New(nil, "", config, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w.Start())

	enr, err := w.ENR()
	require.NoError(t, err)
	require.NotNil(t, enr)

	// DNSDiscovery
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	discoveredNodes, err := dnsdisc.RetrieveNodes(ctx, enrTreeAddress, dnsdisc.WithResolver(config.Resolver))
	require.NoError(t, err)

	// Peer used for retrieving history
	r, err := rand.Int(rand.Reader, big.NewInt(int64(len(discoveredNodes))))
	require.NoError(t, err)

	storeNode := discoveredNodes[int(r.Int64())]

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	// Sanity check, not great, but it's probably helpful
	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) < 1 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)
	require.NoError(t, err)

	// Dropping Peer
	err = w.DropPeer(storeNode.PeerID)
	require.NoError(t, err)

	// Dialing with peerID
	err = w.DialPeerByID(storeNode.PeerID)
	require.NoError(t, err)

	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) < 1 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)
	require.NoError(t, err)

	filter := &common.Filter{
		PubsubTopic:   config.DefaultShardPubsubTopic,
		Messages:      common.NewMemoryMessageStore(),
		ContentTopics: common.NewTopicSetFromBytes([][]byte{{1, 2, 3, 4}}),
	}

	_, err = w.Subscribe(filter)
	require.NoError(t, err)

	msgTimestamp := w.timestamp()
	contentTopic := maps.Keys(filter.ContentTopics)[0]

	time.Sleep(2 * time.Second)

	_, err = w.Send(config.DefaultShardPubsubTopic, &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	}, nil)

	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	timestampInSeconds := msgTimestamp / int64(time.Second)
	marginInSeconds := 20

	options = func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 60 * time.Second
		b.InitialInterval = 500 * time.Millisecond
	}
	err = tt.RetryWithBackOff(func() error {
		_, envelopeCount, err := w.Query(
			context.Background(),
			storeNode.PeerID,
			store.FilterCriteria{
				ContentFilter: protocol.NewContentFilter(config.DefaultShardPubsubTopic, contentTopic.ContentTopic()),
				TimeStart:     proto.Int64((timestampInSeconds - int64(marginInSeconds)) * int64(time.Second)),
				TimeEnd:       proto.Int64((timestampInSeconds + int64(marginInSeconds)) * int64(time.Second)),
			},
			nil,
			nil,
			false,
		)
		if err != nil || envelopeCount == 0 {
			// in case of failure extend timestamp margin up to 40secs
			if marginInSeconds < 40 {
				marginInSeconds += 5
			}
			return errors.New("no messages received from store node")
		}
		return nil
	}, options)
	require.NoError(t, err)

	require.NoError(t, w.Stop())
}

type mapResolver map[string]string

func (mr mapResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	if record, ok := mr[name]; ok {
		return []string{record}, nil
	}
	return nil, errors.New("not found")
}

var signingKeyForTesting, _ = crypto.ToECDSA(hexutil.MustDecode("0xdc599867fc513f8f5e2c2c9c489cde5e71362d1d9ec6e693e0de063236ed1240"))

func makeTestTree(domain string, nodes []*enode.Node, links []string) (*ethdnsdisc.Tree, string) {
	tree, err := ethdnsdisc.MakeTree(1, nodes, links)
	if err != nil {
		panic(err)
	}
	url, err := tree.Sign(signingKeyForTesting, domain)
	if err != nil {
		panic(err)
	}
	return tree, url
}

func TestPeerExchange(t *testing.T) {
	logger := tt.MustCreateTestLogger()
	// start node which serve as PeerExchange server
	config := &Config{}
	config.ClusterID = 16
	config.EnableDiscV5 = true
	config.EnablePeerExchangeServer = true
	config.EnablePeerExchangeClient = false
	pxServerNode, err := New(nil, "", config, logger.Named("pxServerNode"), nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, pxServerNode.Start())

	time.Sleep(1 * time.Second)

	// start node that will be discovered by PeerExchange
	config = &Config{}
	config.ClusterID = 16
	config.EnableDiscV5 = true
	config.EnablePeerExchangeServer = false
	config.EnablePeerExchangeClient = false
	config.DiscV5BootstrapNodes = []string{pxServerNode.node.ENR().String()}
	discV5Node, err := New(nil, "", config, logger.Named("discV5Node"), nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, discV5Node.Start())

	time.Sleep(1 * time.Second)

	// start light node which use PeerExchange to discover peers
	enrNodes := []*enode.Node{pxServerNode.node.ENR()}
	tree, url := makeTestTree("n", enrNodes, nil)
	resolver := mapResolver(tree.ToTXT("n"))

	config = &Config{}
	config.ClusterID = 16
	config.EnablePeerExchangeServer = false
	config.EnablePeerExchangeClient = true
	config.LightClient = true
	config.Resolver = resolver

	config.WakuNodes = []string{url}
	lightNode, err := New(nil, "", config, logger.Named("lightNode"), nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, lightNode.Start())

	// Sanity check, not great, but it's probably helpful
	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}
	err = tt.RetryWithBackOff(func() error {
		// we should not use lightNode.Peers() here as it only indicates peers that are connected right now,
		// in light client mode,the peer will be closed via `w.node.Host().Network().ClosePeer(peerInfo.ID)`
		// after invoking identifyAndConnect, instead, we should check the peerStore, peers from peerStore
		// won't get deleted especially if they are statically added.
		if len(lightNode.node.Host().Peerstore().Peers()) == 2 {
			return nil
		}
		return errors.New("no peers discovered")
	}, options)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, discV5Node.node.PeerExchange().Request(ctx, 1))
	require.Error(t, discV5Node.node.PeerExchange().Request(ctx, 1)) //should fail due to rate limit

	require.NoError(t, lightNode.Stop())
	require.NoError(t, pxServerNode.Stop())
	require.NoError(t, discV5Node.Stop())
}

func TestWakuV2Filter(t *testing.T) {
	t.Skip("flaky test")

	enrTreeAddress := testBootENRBootstrap
	envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
	if envEnrTreeAddress != "" {
		enrTreeAddress = envEnrTreeAddress
	}
	config := &Config{}
	setDefaultConfig(config, true)
	config.EnablePeerExchangeClient = false
	config.Port = 0
	config.MinPeersForFilter = 2

	config.DiscV5BootstrapNodes = []string{enrTreeAddress}
	config.DiscoveryLimit = 20
	config.WakuNodes = []string{enrTreeAddress}
	w, err := New(nil, "", config, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w.Start())

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 10 * time.Second
	}
	time.Sleep(10 * time.Second) //TODO: Check if we can remove this sleep.

	// Sanity check, not great, but it's probably helpful
	err = tt.RetryWithBackOff(func() error {
		peers, err := w.node.PeerManager().FilterPeersByProto(nil, nil, filter.FilterSubscribeID_v20beta1)
		if err != nil {
			return err
		}
		if len(peers) < 2 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)
	require.NoError(t, err)
	testPubsubTopic := "/waku/2/rs/16/32"
	contentTopicBytes := make([]byte, 4)
	_, err = rand.Read(contentTopicBytes)
	require.NoError(t, err)
	filter := &common.Filter{
		Messages:      common.NewMemoryMessageStore(),
		PubsubTopic:   testPubsubTopic,
		ContentTopics: common.NewTopicSetFromBytes([][]byte{contentTopicBytes}),
	}

	fID, err := w.Subscribe(filter)
	require.NoError(t, err)

	msgTimestamp := w.timestamp()
	contentTopic := maps.Keys(filter.ContentTopics)[0]

	_, err = w.Send(testPubsubTopic, &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	}, nil)
	require.NoError(t, err)
	time.Sleep(5 * time.Second)

	// Ensure there is at least 1 active filter subscription
	subscriptions := w.node.FilterLightnode().Subscriptions()
	require.Greater(t, len(subscriptions), 0)

	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	// Mock peers going down
	_, err = w.node.FilterLightnode().UnsubscribeWithSubscription(w.ctx, subscriptions[0])
	require.NoError(t, err)

	time.Sleep(10 * time.Second)

	// Ensure there is at least 1 active filter subscription
	subscriptions = w.node.FilterLightnode().Subscriptions()
	require.Greater(t, len(subscriptions), 0)

	// Ensure that messages are retrieved with a fresh sub
	_, err = w.Send(testPubsubTopic, &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5, 6},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	}, nil)
	require.NoError(t, err)
	time.Sleep(10 * time.Second)

	messages = filter.Retrieve()
	require.Len(t, messages, 1)
	err = w.Unsubscribe(context.Background(), fID)
	require.NoError(t, err)
	require.NoError(t, w.Stop())
}

func TestWakuV2Store(t *testing.T) {
	t.Skip("deprecated. Storenode must use nwaku")

	// Configuration for the first Waku node
	config1 := &Config{
		Port:                             0,
		ClusterID:                        16,
		EnableDiscV5:                     false,
		DiscoveryLimit:                   20,
		EnableStore:                      false,
		StoreCapacity:                    100,
		StoreSeconds:                     3600,
		EnableMissingMessageVerification: true,
	}
	w1PeersCh := make(chan peer.IDSlice, 100) // buffered not to block on the send side

	// Start the first Waku node
	w1, err := New(nil, "", config1, nil, nil, nil, nil, func(cs types.ConnStatus) {
		w1PeersCh <- maps.Keys(cs.Peers)
	})
	require.NoError(t, err)
	require.NoError(t, w1.Start())
	defer func() {
		require.NoError(t, w1.Stop())
		close(w1PeersCh)
	}()

	// Configuration for the second Waku node
	sql2, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	config2 := &Config{
		Port:           0,
		ClusterID:      16,
		EnableDiscV5:   false,
		DiscoveryLimit: 20,
		EnableStore:    true,
		StoreCapacity:  100,
		StoreSeconds:   3600,
	}

	// Start the second Waku node
	w2, err := New(nil, "", config2, nil, sql2, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w2.Start())
	w2EnvelopeCh := make(chan common.EnvelopeEvent, 100)
	w2.SubscribeEnvelopeEvents(w2EnvelopeCh)
	defer func() {
		require.NoError(t, w2.Stop())
		close(w2EnvelopeCh)
	}()

	// Connect the two nodes directly
	peer2Addr := w2.node.ListenAddresses()[0].String()
	err = w1.node.DialPeer(context.Background(), peer2Addr)
	require.NoError(t, err)

	waitForPeerConnection(t, w2.node.Host().ID(), w1PeersCh)

	// Create a filter for the second node to catch messages
	filter := &common.Filter{
		Messages:      common.NewMemoryMessageStore(),
		PubsubTopic:   config2.DefaultShardPubsubTopic,
		ContentTopics: common.NewTopicSetFromBytes([][]byte{{1, 2, 3, 4}}),
	}

	_, err = w2.Subscribe(filter)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Send a message from the first node
	msgTimestamp := w1.CurrentTime().UnixNano()
	contentTopic := maps.Keys(filter.ContentTopics)[0]
	_, err = w1.Send(config1.DefaultShardPubsubTopic, &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	}, nil)
	require.NoError(t, err)

	waitForEnvelope(t, contentTopic.ContentTopic(), w2EnvelopeCh)

	// Retrieve the message from the second node's filter
	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	timestampInSeconds := msgTimestamp / int64(time.Second)
	marginInSeconds := 5
	// Query the second node's store for the message
	_, envelopeCount, err := w1.Query(
		context.Background(),
		w2.node.Host().ID(),
		store.FilterCriteria{
			TimeStart:     proto.Int64((timestampInSeconds - int64(marginInSeconds)) * int64(time.Second)),
			TimeEnd:       proto.Int64((timestampInSeconds + int64(marginInSeconds)) * int64(time.Second)),
			ContentFilter: protocol.NewContentFilter(config1.DefaultShardPubsubTopic, contentTopic.ContentTopic()),
		},
		nil,
		nil,
		false,
	)
	require.NoError(t, err)
	require.True(t, envelopeCount > 0, "no messages received from store node")
}

func waitForPeerConnection(t *testing.T, peerID peer.ID, peerCh chan peer.IDSlice) {
	waitForPeerConnectionWithTimeout(t, peerID, peerCh, 3*time.Second)
}

func waitForPeerConnectionWithTimeout(t *testing.T, peerID peer.ID, peerCh chan peer.IDSlice, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case peers := <-peerCh:
			for _, p := range peers {
				if p == peerID {
					return
				}
			}
		case <-ctx.Done():
			require.Fail(t, "timed out waiting for peer "+peerID.String())
			return
		}
	}
}

func waitForEnvelope(t *testing.T, contentTopic string, envCh chan common.EnvelopeEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		select {
		case env := <-envCh:
			if env.Topic.ContentTopic() == contentTopic {
				return
			}
		case <-ctx.Done():
			require.Fail(t, "timed out waiting for envelope's topic "+contentTopic)
			return
		}
	}
}

func TestOnlineChecker(t *testing.T) {
	w, err := New(nil, "shards.staging", nil, nil, nil, nil, nil, nil)
	require.NoError(t, w.Start())

	require.NoError(t, err)
	require.False(t, w.onlineChecker.IsOnline())

	w.ConnectionChanged(connection.State{Offline: false})
	require.True(t, w.onlineChecker.IsOnline())

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-w.goingOnline
		require.True(t, true)
	}()

	time.Sleep(100 * time.Millisecond)

	w.ConnectionChanged(connection.State{Offline: true})
	require.False(t, w.onlineChecker.IsOnline())

	// Test lightnode online checker
	config := &Config{}
	config.ClusterID = 16
	config.LightClient = true
	lightNode, err := New(nil, "shards.staging", config, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	err = lightNode.Start()
	require.NoError(t, err)

	require.False(t, lightNode.onlineChecker.IsOnline())
	f := &common.Filter{}
	lightNode.filterManager.SubscribeFilter("test", protocol.NewContentFilter(f.PubsubTopic, f.ContentTopics.ContentTopics()...))

}

func TestLightpushRateLimit(t *testing.T) {
	logger := tt.MustCreateTestLogger()

	config0 := &Config{}
	setDefaultConfig(config0, false)
	w0PeersCh := make(chan peer.IDSlice, 5) // buffered not to block on the send side

	// Start the relayu node
	w0, err := New(nil, "", config0, logger.Named("relayNode"), nil, nil, nil, func(cs types.ConnStatus) {
		w0PeersCh <- maps.Keys(cs.Peers)
	})
	require.NoError(t, err)
	require.NoError(t, w0.Start())
	defer func() {
		require.NoError(t, w0.Stop())
		close(w0PeersCh)
	}()

	contentTopics := common.NewTopicSetFromBytes([][]byte{{1, 2, 3, 4}})
	filter := &common.Filter{
		PubsubTopic:   config0.DefaultShardPubsubTopic,
		Messages:      common.NewMemoryMessageStore(),
		ContentTopics: contentTopics,
	}

	_, err = w0.Subscribe(filter)
	require.NoError(t, err)

	config1 := &Config{}
	setDefaultConfig(config1, false)
	w1PeersCh := make(chan peer.IDSlice, 5) // buffered not to block on the send side

	// Start the full node
	w1, err := New(nil, "", config1, logger.Named("fullNode"), nil, nil, nil, func(cs types.ConnStatus) {
		w1PeersCh <- maps.Keys(cs.Peers)
	})
	require.NoError(t, err)
	require.NoError(t, w1.Start())
	defer func() {
		require.NoError(t, w1.Stop())
		close(w1PeersCh)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//Connect the relay peer and full node
	err = w1.node.DialPeer(ctx, w0.node.ListenAddresses()[0].String())
	require.NoError(t, err)

	err = tt.RetryWithBackOff(func() error {
		if len(w1.Peers()) == 0 {
			return errors.New("no peers discovered")
		}
		return nil
	})
	require.NoError(t, err)

	config2 := &Config{}
	setDefaultConfig(config2, true)
	w2PeersCh := make(chan peer.IDSlice, 5) // buffered not to block on the send side

	// Start the light node
	w2, err := New(nil, "", config2, logger.Named("lightNode"), nil, nil, nil, func(cs types.ConnStatus) {
		w2PeersCh <- maps.Keys(cs.Peers)
	})
	require.NoError(t, err)
	require.NoError(t, w2.Start())
	defer func() {
		require.NoError(t, w2.Stop())
		close(w2PeersCh)
	}()

	//Use this instead of DialPeer to make sure the peer is added to PeerStore and can be selected for Lighpush
	w2.node.AddDiscoveredPeer(w1.PeerID(), w1.node.ListenAddresses(), wps.Static, w1.cfg.DefaultShardedPubsubTopics, w1.node.ENR(), true)

	waitForPeerConnectionWithTimeout(t, w2.node.Host().ID(), w1PeersCh, 5*time.Second)

	event := make(chan common.EnvelopeEvent, 10)
	w2.SubscribeEnvelopeEvents(event)

	for i := range [4]int{} {
		msgTimestamp := w2.timestamp()
		_, err := w2.Send(config2.DefaultShardPubsubTopic, &pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5, 6, byte(i)},
			ContentTopic: maps.Keys(contentTopics)[0].ContentTopic(),
			Version:      proto.Uint32(0),
			Timestamp:    &msgTimestamp,
		}, nil)

		require.NoError(t, err)

		time.Sleep(550 * time.Millisecond)

	}

	messages := filter.Retrieve()
	require.Len(t, messages, 2)

}

func TestTelemetryFormat(t *testing.T) {
	tc := NewBandwidthTelemetryClient(tt.MustCreateTestLogger(), "#")

	s := metrics.Stats{
		TotalIn:  10,
		TotalOut: 20,
		RateIn:   30,
		RateOut:  40,
	}

	m := make(map[libp2pprotocol.ID]metrics.Stats)
	m[relay.WakuRelayID_v200] = s
	m[filter.FilterPushID_v20beta1] = s
	m[filter.FilterSubscribeID_v20beta1] = s
	m[legacy_store.StoreID_v20beta4] = s
	m[lightpush.LightPushID_v20beta1] = s

	requestBody := tc.getTelemetryRequestBody(m)
	_, err := json.Marshal(requestBody)
	require.NoError(t, err)
}
