package wakuv2

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/cenkalti/backoff/v3"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	ethdnsdisc "github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/wakuv2/common"
)

var testENRBootstrap = "enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@boot.staging.shards.nodes.status.im"

func setDefaultConfig(config *Config, lightMode bool) {
	config.ClusterID = 16
	config.UseShardAsDefaultTopic = true

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
	config.DiscV5BootstrapNodes = []string{testENRBootstrap}
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
	config.UDPPort = 9002
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

	w.discV5BootstrapNodes = []string{testENRBootstrap}

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

func TestBasicWakuV2(t *testing.T) {
	enrTreeAddress := testENRBootstrap //"enrtree://AL65EKLJAUXKKPG43HVTML5EFFWEZ7L4LOKTLZCLJASG4DSESQZEC@prod.status.nodes.status.im"
	envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
	if envEnrTreeAddress != "" {
		enrTreeAddress = envEnrTreeAddress
	}

	config := &Config{}
	setDefaultConfig(config, false)
	config.Port = 0
	config.DiscV5BootstrapNodes = []string{enrTreeAddress}
	config.DiscoveryLimit = 20
	config.WakuNodes = []string{enrTreeAddress}
	w, err := New(nil, "", config, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w.Start())

	// DNSDiscovery
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	discoveredNodes, err := dnsdisc.RetrieveNodes(ctx, enrTreeAddress)
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
		if len(w.Peers()) < 2 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)

	require.NoError(t, err)

	filter := &common.Filter{
		Messages:      common.NewMemoryMessageStore(),
		ContentTopics: common.NewTopicSetFromBytes([][]byte{[]byte{1, 2, 3, 4}}),
	}

	_, err = w.Subscribe(filter)
	require.NoError(t, err)

	msgTimestamp := w.timestamp()
	contentTopic := maps.Keys(filter.ContentTopics)[0]

	_, err = w.Send(relay.DefaultWakuTopic, &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	})

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
		storeResult, err := w.query(context.Background(), storeNode.PeerID, relay.DefaultWakuTopic, []common.TopicType{contentTopic}, uint64(timestampInSeconds-int64(marginInSeconds)), uint64(timestampInSeconds+int64(marginInSeconds)), []byte{}, []legacy_store.HistoryRequestOption{})
		if err != nil || len(storeResult.Messages) == 0 {
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
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	// start node which serve as PeerExchange server
	config := &Config{}
	config.EnableDiscV5 = true
	config.EnablePeerExchangeServer = true
	config.EnablePeerExchangeClient = false
	pxServerNode, err := New(nil, "", config, logger.Named("pxServerNode"), nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, pxServerNode.Start())

	time.Sleep(1 * time.Second)

	// start node that will be discovered by PeerExchange
	config = &Config{}
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

	require.NoError(t, lightNode.Stop())
	require.NoError(t, pxServerNode.Stop())
	require.NoError(t, discV5Node.Stop())
}

func TestWakuV2Filter(t *testing.T) {
	enrTreeAddress := testENRBootstrap
	envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
	if envEnrTreeAddress != "" {
		enrTreeAddress = envEnrTreeAddress
	}

	config := &Config{}
	setDefaultConfig(config, true)
	config.Port = 0
	config.KeepAliveInterval = 0
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

	filter := &common.Filter{
		Messages:      common.NewMemoryMessageStore(),
		PubsubTopic:   "/waku/2/rs/16/1",
		ContentTopics: common.NewTopicSetFromBytes([][]byte{[]byte{1, 2, 3, 4}}),
	}

	fmt.Println("### Subscribe")
	_, err = w.Subscribe(filter)
	require.NoError(t, err)

	msgTimestamp := w.timestamp()
	contentTopic := maps.Keys(filter.ContentTopics)[0]

	_, err = w.Send("", &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	})
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	fmt.Println("### Check")
	// Ensure there is at least 1 active filter subscription
	subscriptions := w.node.FilterLightnode().Subscriptions()
	require.Greater(t, len(subscriptions), 0)

	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	// Mock peers going down
	subscriptions[0].Close()

	time.Sleep(10 * time.Second)

	// Ensure there is at least 1 active filter subscription
	subscriptions = w.node.FilterLightnode().Subscriptions()
	require.Greater(t, len(subscriptions), 0)

	// Ensure that messages are retrieved with a fresh sub
	_, err = w.Send("", &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5, 6},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	})
	require.NoError(t, err)
	time.Sleep(10 * time.Second)

	messages = filter.Retrieve()
	require.Len(t, messages, 1)

	require.NoError(t, w.Stop())
}

func TestWakuV2Store(t *testing.T) {
	// Configuration for the first Waku node
	config1 := &Config{
		Port:              0,
		EnableDiscV5:      false,
		DiscoveryLimit:    20,
		EnableStore:       false,
		StoreCapacity:     100,
		StoreSeconds:      3600,
		KeepAliveInterval: 10,
	}
	w1PeersCh := make(chan []string, 100) // buffered not to block on the send side

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
		Port:              0,
		EnableDiscV5:      false,
		DiscoveryLimit:    20,
		EnableStore:       true,
		StoreCapacity:     100,
		StoreSeconds:      3600,
		KeepAliveInterval: 10,
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

	waitForPeerConnection(t, w2.node.ID(), w1PeersCh)

	// Create a filter for the second node to catch messages
	filter := &common.Filter{
		Messages:      common.NewMemoryMessageStore(),
		ContentTopics: common.NewTopicSetFromBytes([][]byte{{1, 2, 3, 4}}),
	}

	_, err = w2.Subscribe(filter)
	require.NoError(t, err)

	// Send a message from the first node
	msgTimestamp := w1.CurrentTime().UnixNano()
	contentTopic := maps.Keys(filter.ContentTopics)[0]
	_, err = w1.Send(relay.DefaultWakuTopic, &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	})
	require.NoError(t, err)

	waitForEnvelope(t, contentTopic.ContentTopic(), w2EnvelopeCh)

	// Retrieve the message from the second node's filter
	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	timestampInSeconds := msgTimestamp / int64(time.Second)
	marginInSeconds := 5

	// Query the second node's store for the message
	storeResult, err := w1.query(context.Background(), w2.node.Host().ID(), relay.DefaultWakuTopic, []common.TopicType{contentTopic}, uint64(timestampInSeconds-int64(marginInSeconds)), uint64(timestampInSeconds+int64(marginInSeconds)), []byte{}, []legacy_store.HistoryRequestOption{})
	require.NoError(t, err)
	require.True(t, len(storeResult.Messages) > 0, "no messages received from store node")
}

func waitForPeerConnection(t *testing.T, peerID string, peerCh chan []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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
			require.Fail(t, "timed out waiting for peer "+peerID)
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
