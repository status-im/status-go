package peers

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"

	// to access logs in the test with `-log` flag
	_ "github.com/status-im/status-go/t/utils"
)

type PeerPoolSimulationSuite struct {
	suite.Suite

	bootnode  *p2p.Server
	peers     []*p2p.Server
	discovery []Discovery
	port      uint16
}

func TestPeerPoolSimulationSuite(t *testing.T) {
	s := new(PeerPoolSimulationSuite)
	s.port = 33731
	suite.Run(t, s)
}

func (s *PeerPoolSimulationSuite) nextPort() uint16 {
	s.port++
	return s.port
}

func (s *PeerPoolSimulationSuite) SetupTest() {
	bootnodePort := s.nextPort()
	key, _ := crypto.GenerateKey()
	name := common.MakeName("bootnode", "1.0")
	// 127.0.0.1 is invalidated by discovery v5
	s.bootnode = &p2p.Server{
		Config: p2p.Config{
			MaxPeers:    10,
			Name:        name,
			ListenAddr:  fmt.Sprintf("0.0.0.0:%d", bootnodePort),
			PrivateKey:  key,
			DiscoveryV5: true,
			NoDiscovery: true,
		},
	}
	s.Require().NoError(s.bootnode.Start())
	bootnodeV5 := discv5.NewNode(s.bootnode.DiscV5.Self().ID, net.ParseIP("127.0.0.1"), bootnodePort, bootnodePort)

	// 1 peer to initiate connection, 1 peer as a first candidate, 1 peer - for failover
	s.peers = make([]*p2p.Server, 3)
	s.discovery = make([]Discovery, 3)
	for i := range s.peers {
		key, _ := crypto.GenerateKey()
		whisper := whisperv6.New(nil)
		peer := &p2p.Server{
			Config: p2p.Config{
				MaxPeers:         10,
				Name:             common.MakeName("peer-"+strconv.Itoa(i), "1.0"),
				ListenAddr:       fmt.Sprintf("0.0.0.0:%d", s.nextPort()),
				PrivateKey:       key,
				NoDiscovery:      true,
				BootstrapNodesV5: []*discv5.Node{bootnodeV5},
				Protocols:        whisper.Protocols(),
			},
		}
		s.NoError(peer.Start())
		s.peers[i] = peer
		d := NewDiscV5(key, peer.ListenAddr, peer.BootstrapNodesV5)
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
}

func (s *PeerPoolSimulationSuite) getPeerFromEvent(events <-chan *p2p.PeerEvent, etype p2p.PeerEventType) (nodeID discover.NodeID) {
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

func (s *PeerPoolSimulationSuite) TestPeerPoolCache() {
	var err error

	topic := discv5.Topic("cap=test")
	config := map[discv5.Topic]params.Limits{
		topic: params.NewLimits(1, 1),
	}
	peerPoolOpts := &Options{100 * time.Millisecond, 100 * time.Millisecond, 0, true, 100 * time.Millisecond}
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

func (s *PeerPoolSimulationSuite) TestSingleTopicDiscoveryWithFailover() {
	s.T().Skip("Skipping due to being flaky")

	var err error

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
		case signal.EventDiscoveryStarted, signal.EventDiscoveryStopped:
			poolEvents <- envelope.Type
		case signal.EventDiscoverySummary:
			poolEvents <- envelope.Type
			var summary []*p2p.PeerInfo
			s.NoError(json.Unmarshal(envelope.Event, &summary))
			summaries <- summary
		}
	})
	defer signal.ResetDefaultNodeNotificationHandler()

	topic := discv5.Topic("cap=test")
	// simulation should only rely on fast sync
	config := map[discv5.Topic]params.Limits{
		topic: params.NewLimits(1, 1), // limits are chosen for simplicity of the simulation
	}
	peerPoolOpts := &Options{100 * time.Millisecond, 100 * time.Millisecond, 0, true, 0}
	cache, err := newInMemoryCache()
	s.Require().NoError(err)
	peerPool := NewPeerPool(s.discovery[1], config, cache, peerPoolOpts)

	// create and start topic registry
	register := NewRegister(s.discovery[0], topic)
	s.Require().NoError(register.Start())

	// subscribe for peer events before starting the peer pool
	events := make(chan *p2p.PeerEvent, 20)
	subscription := s.peers[1].SubscribeEvents(events)
	defer subscription.Unsubscribe()

	// start the peer pool
	s.Require().NoError(peerPool.Start(s.peers[1]))
	defer peerPool.Stop()
	s.Equal(signal.EventDiscoveryStarted, s.getPoolEvent(poolEvents))

	// wait for the peer to be found and connected
	connectedPeer := s.getPeerFromEvent(events, p2p.PeerEventTypeAdd)
	s.Equal(s.peers[0].Self().ID, connectedPeer)
	// as the upper limit was reached, Discovery should be stoped
	s.Equal(signal.EventDiscoverySummary, s.getPoolEvent(poolEvents))
	s.Equal(signal.EventDiscoveryStopped, s.getPoolEvent(poolEvents))
	s.Len(<-summaries, 1)

	// stop topic register and the connected peer
	register.Stop()
	s.peers[0].Stop()
	disconnectedPeer := s.getPeerFromEvent(events, p2p.PeerEventTypeDrop)
	s.Equal(connectedPeer, disconnectedPeer)
	s.Equal(signal.EventDiscoverySummary, s.getPoolEvent(poolEvents))
	s.Len(<-summaries, 0)
	// Discovery should be restarted because the number of peers dropped
	// below the lower limit.
	s.Equal(signal.EventDiscoveryStarted, s.getPoolEvent(poolEvents))

	// register the second peer
	register = NewRegister(s.discovery[2], topic)
	s.Require().NoError(register.Start())
	defer register.Stop()
	s.Equal(s.peers[2].Self().ID, s.getPeerFromEvent(events, p2p.PeerEventTypeAdd))
	// Discovery can be stopped again.
	s.Require().Equal(signal.EventDiscoverySummary, s.getPoolEvent(poolEvents))
	s.Equal(signal.EventDiscoveryStopped, s.getPoolEvent(poolEvents))
	s.Len(<-summaries, 1)
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
	discovery := NewDiscV5(key, peer.ListenAddr, nil)
	require.NoError(t, discovery.Start())
	defer func() { assert.NoError(t, discovery.Stop()) }()
	require.True(t, discovery.Running())

	poolOpts := &Options{DefaultFastSync, DefaultSlowSync, 0, true, 100 * time.Millisecond}
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

	discovery := NewDiscV5(key, server.ListenAddr, nil)
	require.NoError(t, discovery.Start())
	defer func() { assert.NoError(t, discovery.Stop()) }()
	require.True(t, discovery.Running())

	// start PeerPool
	poolOpts := &Options{DefaultFastSync, DefaultSlowSync, time.Millisecond * 100, true, 100 * time.Millisecond}
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

	discovery := NewDiscV5(key, server.ListenAddr, nil)
	require.NoError(t, discovery.Start())
	defer func() { assert.NoError(t, discovery.Stop()) }()
	require.True(t, discovery.Running())

	// start PeerPool
	poolOpts := &Options{DefaultFastSync, DefaultSlowSync, time.Millisecond * 100, false, 100 * time.Millisecond}
	pool := NewPeerPool(discovery, nil, nil, poolOpts)
	require.NoError(t, pool.Start(server))

	// wait 2x timeout duration
	<-time.After(pool.opts.DiscServerTimeout * 2)
	require.True(t, discovery.Running())
}
