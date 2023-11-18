package wakuv2

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/require"

	"golang.org/x/exp/maps"

	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/wakuv2/common"
)

var testENRBootstrap = "enrtree://AL65EKLJAUXKKPG43HVTML5EFFWEZ7L4LOKTLZCLJASG4DSESQZEC@prod.status.nodes.status.im"

func TestDiscoveryV5(t *testing.T) {
	config := &Config{}
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = []string{testENRBootstrap}
	config.DiscoveryLimit = 20
	config.UDPPort = 9001
	w, err := New("", "", config, nil, nil, nil, nil, nil)
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
	config.EnableDiscV5 = true
	// Use wrong discv5 bootstrap address, to simulate being offline
	config.DiscV5BootstrapNodes = []string{"enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@1.1.1.2"}
	config.DiscoveryLimit = 20
	config.UDPPort = 9002
	w, err := New("", "", config, nil, nil, nil, nil, nil)
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
	enrTreeAddress := testENRBootstrap
	envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
	if envEnrTreeAddress != "" {
		enrTreeAddress = envEnrTreeAddress
	}

	config := &Config{}
	config.Port = 0
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = []string{enrTreeAddress}
	config.DiscoveryLimit = 20
	config.WakuNodes = []string{enrTreeAddress}
	w, err := New("", "", config, nil, nil, nil, nil, nil)
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
		if len(w.Peers()) > 2 {
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
		Version:      0,
		Timestamp:    msgTimestamp,
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
		storeResult, err := w.query(context.Background(), storeNode.PeerID, relay.DefaultWakuTopic, []common.TopicType{contentTopic}, uint64(timestampInSeconds-int64(marginInSeconds)), uint64(timestampInSeconds+int64(marginInSeconds)), []store.HistoryRequestOption{})
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

func TestWakuV2Filter(t *testing.T) {
	enrTreeAddress := testENRBootstrap
	envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
	if envEnrTreeAddress != "" {
		enrTreeAddress = envEnrTreeAddress
	}

	config := &Config{}
	config.Port = 0
	config.LightClient = true
	config.KeepAliveInterval = 1
	config.MinPeersForFilter = 2
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = []string{enrTreeAddress}
	config.DiscoveryLimit = 20
	config.UDPPort = 9001
	config.WakuNodes = []string{enrTreeAddress}
	fleet := "status.test" // Need a name fleet so that LightClient is not set to false
	w, err := New("", fleet, config, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w.Start())

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 10 * time.Second
	}

	// Sanity check, not great, but it's probably helpful
	err = tt.RetryWithBackOff(func() error {
		if len(w.Peers()) > 2 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)
	require.NoError(t, err)

	filter := &common.Filter{
		Messages:      common.NewMemoryMessageStore(),
		ContentTopics: common.NewTopicSetFromBytes([][]byte{[]byte{1, 2, 3, 4}}),
	}

	filterID, err := w.Subscribe(filter)
	require.NoError(t, err)

	msgTimestamp := w.timestamp()
	contentTopic := maps.Keys(filter.ContentTopics)[0]

	_, err = w.Send("", &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      0,
		Timestamp:    msgTimestamp,
	})
	require.NoError(t, err)

	time.Sleep(15 * time.Second)

	// Ensure there is at least 1 active filter subscription
	subscriptions := w.node.FilterLightnode().Subscriptions()
	require.Greater(t, len(subscriptions), 0)

	// Ensure there are some active peers for this filter subscription
	stats := w.getFilterStats()
	require.Greater(t, len(stats[filterID]), 0)

	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	// Mock peers going down
	isFilterSubAliveBak := w.filterManager.isFilterSubAlive
	w.filterManager.settings.MinPeersForFilter = 0
	w.filterManager.isFilterSubAlive = func(sub *subscription.SubscriptionDetails) error {
		return errors.New("peer down")
	}

	time.Sleep(5 * time.Second)

	// Ensure there are 0 active peers now

	stats = w.getFilterStats()
	require.Len(t, stats[filterID], 0)

	// Reconnect
	w.filterManager.settings.MinPeersForFilter = 2
	w.filterManager.isFilterSubAlive = isFilterSubAliveBak
	time.Sleep(10 * time.Second)

	// Ensure there are some active peers now
	stats = w.getFilterStats()
	require.Greater(t, len(stats[filterID]), 0)

	require.NoError(t, w.Stop())
}

func TestWakuV2Store(t *testing.T) {
	// Configuration for the first Waku node
	sql1, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	config1 := &Config{
		Port:              0,
		EnableDiscV5:      false,
		DiscoveryLimit:    20,
		EnableStore:       true,
		StoreCapacity:     100,
		StoreSeconds:      3600,
		KeepAliveInterval: 10,
	}

	// Start the first Waku node
	w1, err := New("", "", config1, nil, sql1, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w1.Start())
	defer func() {
		require.NoError(t, w1.Stop())
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
	w2, err := New("", "", config2, nil, sql2, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, w2.Start())
	defer func() {
		require.NoError(t, w2.Stop())
	}()

	// Connect the two nodes directly
	peer2Addr := w2.node.ListenAddresses()[0].String()
	err = w1.node.DialPeer(context.Background(), peer2Addr)
	require.NoError(t, err)

	// Sanity check, not great, but it's probably helpful
	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}
	err = tt.RetryWithBackOff(func() error {
		if len(w1.Peers()) == 0 {
			return errors.New("no peers discovered")
		}
		return nil
	}, options)
	require.NoError(t, err)

	// Wait for the nodes to discover each other
	time.Sleep(1 * time.Second)

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
		Version:      0,
		Timestamp:    msgTimestamp,
	})
	require.NoError(t, err)

	// Wait for the message to be transferred
	time.Sleep(1 * time.Second)
	// Retrieve the message from the second node's filter
	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	timestampInSeconds := msgTimestamp / int64(time.Second)
	marginInSeconds := 5

	// Query the second node's store for the message
	storeResult, err := w1.query(context.Background(), w2.node.Host().ID(), relay.DefaultWakuTopic, []common.TopicType{contentTopic}, uint64(timestampInSeconds-int64(marginInSeconds)), uint64(timestampInSeconds+int64(marginInSeconds)), []store.HistoryRequestOption{
		store.WithLocalQuery(),
	})
	if err != nil || len(storeResult.Messages) == 0 {
		t.Fail()
	}

}
