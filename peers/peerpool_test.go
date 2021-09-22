package peers

import (
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/rendezvous/server"

	"github.com/status-im/status-go/discovery"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
)

type PeerPoolSimulationSuite struct {
	suite.Suite

	bootnode         *p2p.Server
	peers            []*p2p.Server
	discovery        []discovery.Discovery
	rendezvousServer *server.Server
}

func TestPeerPoolSimulationSuite(t *testing.T) {
	s := &PeerPoolSimulationSuite{}
	suite.Run(t, s)
}

func (s *PeerPoolSimulationSuite) SetupTest() {
	key, _ := crypto.GenerateKey()
	name := common.MakeName("bootnode", "1.0")
	// 127.0.0.1 is invalidated by discovery v5
	s.bootnode = &p2p.Server{
		Config: p2p.Config{
			MaxPeers:    10,
			Name:        name,
			ListenAddr:  ":0",
			PrivateKey:  key,
			DiscoveryV5: true,
			NoDiscovery: true,
		},
	}
	s.Require().NoError(s.bootnode.Start())
	bootnodePort := uint16(s.bootnode.NodeInfo().Ports.Listener)
	bootnodeV5 := discv5.NewNode(s.bootnode.DiscV5.Self().ID, net.ParseIP("127.0.0.1"), bootnodePort, bootnodePort)

	// 1 peer to initiate connection, 1 peer as a first candidate, 1 peer - for failover
	s.peers = make([]*p2p.Server, 3)
	s.discovery = make([]discovery.Discovery, 3)
	for i := range s.peers {
		key, _ := crypto.GenerateKey()
		peer := &p2p.Server{
			Config: p2p.Config{
				MaxPeers:         10,
				Name:             common.MakeName("peer-"+strconv.Itoa(i), "1.0"),
				ListenAddr:       ":0",
				PrivateKey:       key,
				NoDiscovery:      true,
				BootstrapNodesV5: []*discv5.Node{bootnodeV5},
			},
		}
		s.NoError(peer.Start())
		s.peers[i] = peer
	}
}

func (s *PeerPoolSimulationSuite) setupEthV5() {
	for i := range s.peers {
		peer := s.peers[i]
		d := discovery.NewDiscV5(peer.PrivateKey, peer.ListenAddr, peer.BootstrapNodesV5)
		s.NoError(d.Start())
		s.discovery[i] = d
	}
}

func (s *PeerPoolSimulationSuite) TearDown() {
	s.bootnode.Stop()
	for i := range s.peers {
		s.peers[i].Stop()
		s.NoError(s.discovery[i].Stop())
	}
	if s.rendezvousServer != nil {
		s.rendezvousServer.Stop()
	}
}

func (s *PeerPoolSimulationSuite) getPeerFromEvent(events <-chan *p2p.PeerEvent, etype p2p.PeerEventType) (nodeID enode.ID) {
	select {
	case ev := <-events:
		if ev.Type == etype {
			return ev.Peer
		}
		s.Failf("invalid event", "expected %s but got %s for peer %s", etype, ev.Type, ev.Peer)
	case <-time.After(10 * time.Second):
		s.Fail("timed out waiting for a peer")
		return
	}
	return
}

func (s *PeerPoolSimulationSuite) getPoolEvent(events <-chan string) string {
	select {
	case ev := <-events:
		return ev
	case <-time.After(10 * time.Second):
		s.FailNow("timed out waiting a pool event")
		return ""
	}
}

func (s *PeerPoolSimulationSuite) TestPeerPoolCacheEthV5() {
	s.setupEthV5()
	var err error

	topic := discv5.Topic("cap=test")
	config := map[discv5.Topic]params.Limits{
		topic: params.NewLimits(1, 1),
	}
	peerPoolOpts := &Options{100 * time.Millisecond, 100 * time.Millisecond, 0, true, 100 * time.Millisecond, nil}
	cache, err := newInMemoryCache()
	s.Require().NoError(err)
	peerPool := NewPeerPool(s.discovery[1], config, cache, peerPoolOpts)

	// start peer pool
	s.Require().NoError(peerPool.Start(s.peers[1]))
	defer peerPool.Stop()

	// check if cache is passed to topic pools
	for _, topicPool := range peerPool.topics {
		tp := topicPool.(*TopicPool)
		s.Equal(cache, tp.cache)
	}
}

// TestPeerPoolMaxPeersOverflow verifies that following scenario will not occur:
// - found peer A and B in the same kademlia cycle
// - process peer A
// - max limit is reached -> closed discv5 and set it to nil
// - process peer B
// - panic because discv5 is nil!!!
func TestPeerPoolMaxPeersOverflow(t *testing.T) {
	maxCachedPeersMultiplier = 0
	signals := make(chan string, 10)
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope struct {
			Type string
		}
		require.NoError(t, json.Unmarshal([]byte(jsonEvent), &envelope))
		signals <- envelope.Type
	})
	defer signal.ResetDefaultNodeNotificationHandler()

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	peer := &p2p.Server{
		Config: p2p.Config{
			PrivateKey:  key,
			NoDiscovery: true,
		},
	}
	require.NoError(t, peer.Start())
	defer peer.Stop()
	discovery := discovery.NewDiscV5(key, peer.ListenAddr, nil)
	require.NoError(t, discovery.Start())
	defer func() { assert.NoError(t, discovery.Stop()) }()
	require.True(t, discovery.Running())

	poolOpts := &Options{DefaultFastSync, DefaultSlowSync, 0, true, 100 * time.Millisecond, nil}
	pool := NewPeerPool(discovery, nil, nil, poolOpts)
	require.NoError(t, pool.Start(peer))
	require.Equal(t, signal.EventDiscoveryStarted, <-signals)
	// without config, it will stop the discovery because all topic pools are satisfied
	pool.events <- &p2p.PeerEvent{Type: p2p.PeerEventTypeAdd}
	require.Equal(t, signal.EventDiscoverySummary, <-signals)
	require.Equal(t, signal.EventDiscoveryStopped, <-signals)
	require.False(t, discovery.Running())
	// another peer added after discovery is stopped should not panic
	pool.events <- &p2p.PeerEvent{Type: p2p.PeerEventTypeAdd}
}

func TestPeerPoolDiscV5Timeout(t *testing.T) {
	signals := make(chan string)
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope struct {
			Type  string
			Event json.RawMessage
		}
		require.NoError(t, json.Unmarshal([]byte(jsonEvent), &envelope))
		// Send signal asynchronously to avoid blocking.
		// It's better than sending to a buffered channel because
		// it won't ever block, for example, if two events were expected
		// but received more.
		// In this case, a strange PeerEventTypeDrop event was emitted.
		go func() {
			switch typ := envelope.Type; typ {
			case signal.EventDiscoveryStarted, signal.EventDiscoveryStopped:
				signals <- envelope.Type
			}
		}()
	})
	defer signal.ResetDefaultNodeNotificationHandler()

	// start server
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	server := &p2p.Server{
		Config: p2p.Config{
			PrivateKey:  key,
			NoDiscovery: true,
		},
	}
	require.NoError(t, server.Start())
	defer server.Stop()

	discovery := discovery.NewDiscV5(key, server.ListenAddr, nil)
	require.NoError(t, discovery.Start())
	defer func() { assert.NoError(t, discovery.Stop()) }()
	require.True(t, discovery.Running())

	// start PeerPool
	poolOpts := &Options{DefaultFastSync, DefaultSlowSync, time.Millisecond * 100, true, 100 * time.Millisecond, nil}
	pool := NewPeerPool(discovery, nil, nil, poolOpts)
	require.NoError(t, pool.Start(server))
	require.Equal(t, signal.EventDiscoveryStarted, <-signals)

	// timeout after finding no peers
	select {
	case sig := <-signals:
		require.Equal(t, signal.EventDiscoveryStopped, sig)
	case <-time.After(pool.opts.DiscServerTimeout * 2):
		t.Fatal("timed out")
	}
	require.False(t, discovery.Running())

	// timeout after discovery restart
	require.NoError(t, pool.restartDiscovery(server))
	require.Equal(t, signal.EventDiscoveryStarted, <-signals)
	require.True(t, discovery.Running())
	pool.events <- &p2p.PeerEvent{Type: p2p.PeerEventTypeDrop} // required to turn the loop and pick up new timeout
	select {
	case sig := <-signals:
		require.Equal(t, signal.EventDiscoveryStopped, sig)
	case <-time.After(pool.opts.DiscServerTimeout * 2):
		t.Fatal("timed out")
	}
	require.False(t, discovery.Running())
}

func TestPeerPoolNotAllowedStopping(t *testing.T) {
	// create and start server
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	server := &p2p.Server{
		Config: p2p.Config{
			PrivateKey:  key,
			NoDiscovery: true,
		},
	}
	require.NoError(t, server.Start())
	defer server.Stop()

	discovery := discovery.NewDiscV5(key, server.ListenAddr, nil)
	require.NoError(t, discovery.Start())
	defer func() { assert.NoError(t, discovery.Stop()) }()
	require.True(t, discovery.Running())

	// start PeerPool
	poolOpts := &Options{DefaultFastSync, DefaultSlowSync, time.Millisecond * 100, false, 100 * time.Millisecond, nil}
	pool := NewPeerPool(discovery, nil, nil, poolOpts)
	require.NoError(t, pool.Start(server))

	// wait 2x timeout duration
	<-time.After(pool.opts.DiscServerTimeout * 2)
	require.True(t, discovery.Running())
}

func (s *PeerPoolSimulationSuite) TestUpdateTopicLimits() {
	s.setupEthV5()
	var err error

	topic := discv5.Topic("cap=test")
	config := map[discv5.Topic]params.Limits{
		topic: params.NewLimits(1, 1),
	}
	peerPoolOpts := &Options{100 * time.Millisecond, 100 * time.Millisecond, 0, true, 100 * time.Millisecond, nil}
	cache, err := newInMemoryCache()
	s.Require().NoError(err)
	peerPool := NewPeerPool(s.discovery[1], config, cache, peerPoolOpts)

	// start peer pool
	s.Require().NoError(peerPool.Start(s.peers[1]))
	defer peerPool.Stop()

	for _, topicPool := range peerPool.topics {
		tp := topicPool.(*TopicPool)
		s.Equal(1, tp.limits.Max)
		s.Equal(1, tp.limits.Min)
	}

	// Updating TopicPool's limits
	err = peerPool.UpdateTopic("cap=test", params.NewLimits(5, 10))
	s.Require().NoError(err)
	time.Sleep(1 * time.Millisecond)
	for _, topicPool := range peerPool.topics {
		tp := topicPool.(*TopicPool)
		tp.mu.RLock()
		defer tp.mu.RUnlock()

		s.Equal(10, tp.limits.Max)
		s.Equal(5, tp.limits.Min)
	}
}

func (s *PeerPoolSimulationSuite) TestMailServerPeersDiscovery() {
	s.setupEthV5()

	// eliminate peer we won't use
	s.peers[2].Stop()

	// Buffered channels must be used because we expect the events
	// to be in the same order. Use a buffer length greater than
	// the expected number of events to avoid deadlock.
	poolEvents := make(chan string, 10)
	summaries := make(chan []*p2p.PeerInfo, 10)
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope struct {
			Type  string
			Event json.RawMessage
		}
		s.NoError(json.Unmarshal([]byte(jsonEvent), &envelope))

		switch typ := envelope.Type; typ {
		case signal.EventDiscoverySummary:
			poolEvents <- envelope.Type
			var summary []*p2p.PeerInfo
			s.NoError(json.Unmarshal(envelope.Event, &summary))
			if len(summary) != 0 {
				summaries <- summary
			}
		}
	})
	defer signal.ResetDefaultNodeNotificationHandler()

	// subscribe for peer events before starting the peer pool
	events := make(chan *p2p.PeerEvent, 20)
	subscription := s.peers[1].SubscribeEvents(events)
	defer subscription.Unsubscribe()

	// create and start topic registry
	register := NewRegister(s.discovery[0], MailServerDiscoveryTopic)
	s.Require().NoError(register.Start())

	// create and start peer pool
	config := map[discv5.Topic]params.Limits{
		MailServerDiscoveryTopic: params.NewLimits(1, 1),
	}
	cache, err := newInMemoryCache()
	s.Require().NoError(err)
	peerPoolOpts := &Options{
		100 * time.Millisecond,
		100 * time.Millisecond,
		0,
		true,
		100 * time.Millisecond,
		[]enode.ID{s.peers[0].Self().ID()},
	}
	peerPool := NewPeerPool(s.discovery[1], config, cache, peerPoolOpts)
	s.Require().NoError(peerPool.Start(s.peers[1]))
	defer peerPool.Stop()

	// wait for and verify the mail server peer
	connectedPeer := s.getPeerFromEvent(events, p2p.PeerEventTypeAdd)
	s.Equal(s.peers[0].Self().ID().String(), connectedPeer.String())

	// wait for a summary event to be sure that ConfirmAdded() was called
	s.Equal(signal.EventDiscoverySummary, s.getPoolEvent(poolEvents))
	summary := (<-summaries)
	s.Require().Len(summary, 1)
	s.Equal(s.peers[0].Self().ID().String(), summary[0].ID)

	// check cache
	cachedPeers := peerPool.cache.GetPeersRange(MailServerDiscoveryTopic, 5)
	s.Require().Len(cachedPeers, 1)
	s.Equal(discv5.PubkeyID(s.peers[0].Self().Pubkey()), cachedPeers[0].ID)

	// wait for another event as the peer should be removed
	disconnectedPeer := s.getPeerFromEvent(events, p2p.PeerEventTypeDrop)
	s.Equal(s.peers[0].Self().ID().String(), disconnectedPeer.String())
	s.Equal(signal.EventDiscoverySummary, s.getPoolEvent(poolEvents))
}
