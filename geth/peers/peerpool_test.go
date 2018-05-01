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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/signal"
)

type PeerPoolSimulationSuite struct {
	suite.Suite

	bootnode *p2p.Server
	peers    []*p2p.Server
}

func TestPeerPoolSimulationSuite(t *testing.T) {
	suite.Run(t, new(PeerPoolSimulationSuite))
}

func (s *PeerPoolSimulationSuite) SetupTest() {
	port := 33731
	key, _ := crypto.GenerateKey()
	name := common.MakeName("bootnode", "1.0")
	// 127.0.0.1 is invalidated by discovery v5
	s.bootnode = &p2p.Server{
		Config: p2p.Config{
			MaxPeers:    10,
			Name:        name,
			ListenAddr:  fmt.Sprintf("0.0.0.0:%d", 33731),
			PrivateKey:  key,
			DiscoveryV5: true,
			NoDiscovery: true,
		},
	}
	port++
	s.Require().NoError(s.bootnode.Start())
	bootnodeV5 := discv5.NewNode(s.bootnode.DiscV5.Self().ID, net.ParseIP("127.0.0.1"), uint16(port), uint16(port))

	// 1 peer to initiate connection, 1 peer as a first candidate, 1 peer - for failover
	s.peers = make([]*p2p.Server, 3)
	for i := range s.peers {
		key, _ := crypto.GenerateKey()
		whisper := whisperv6.New(nil)
		peer := &p2p.Server{
			Config: p2p.Config{
				MaxPeers:         10,
				Name:             common.MakeName("peer-"+strconv.Itoa(i), "1.0"),
				ListenAddr:       fmt.Sprintf("0.0.0.0:%d", port),
				PrivateKey:       key,
				DiscoveryV5:      true,
				NoDiscovery:      true,
				BootstrapNodesV5: []*discv5.Node{bootnodeV5},
				Protocols:        whisper.Protocols(),
			},
		}
		port++
		s.NoError(peer.Start())
		s.peers[i] = peer
	}
}

func (s *PeerPoolSimulationSuite) getPeerFromEvent(events <-chan *p2p.PeerEvent, etype p2p.PeerEventType) (nodeID discover.NodeID) {
	select {
	case ev := <-events:
		if ev.Type == etype {
			return ev.Peer
		}
	case <-time.After(5 * time.Second):
		s.Fail("timed out waiting for a peer")
		return
	}
	return
}

func (s *PeerPoolSimulationSuite) getPoolEvent(events <-chan string) string {
	select {
	case ev := <-events:
		return ev
	case <-time.After(time.Second):
		s.Fail("timed out waiting for a peer")
		return ""
	}
}

func (s *PeerPoolSimulationSuite) TestSingleTopicDiscoveryWithFailover() {
	poolEvents := make(chan string, 1)
	summaries := make(chan map[string]int, 1)
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		fmt.Println(jsonEvent)
		var envelope struct {
			Type  string
			Event json.RawMessage
		}
		s.NoError(json.Unmarshal([]byte(jsonEvent), &envelope))
		switch envelope.Type {
		case signal.EventDiscoveryStarted:
			poolEvents <- envelope.Type
		case signal.EventDiscoveryStopped:
			poolEvents <- envelope.Type
		case signal.EventDiscoverySummary:
			poolEvents <- envelope.Type
			var summary map[string]int
			s.NoError(json.Unmarshal(envelope.Event, &summary))
			summaries <- summary
		}

	})
	topic := discv5.Topic("cap=test")
	// simulation should only rely on fast sync
	config := map[discv5.Topic]params.Limits{
		topic: params.NewLimits(1, 1), // limits are chosen for simplicity of the simulation
	}
	peerPool := NewPeerPool(config, 100*time.Millisecond, 100*time.Millisecond, nil, true)
	register := NewRegister(topic)
	s.Require().NoError(register.Start(s.peers[0]))
	defer register.Stop()
	// need to wait for topic to get registered, discv5 can query same node
	// for a topic only once a minute
	events := make(chan *p2p.PeerEvent, 20)
	subscription := s.peers[1].SubscribeEvents(events)
	defer subscription.Unsubscribe()
	s.NoError(peerPool.Start(s.peers[1]))
	defer peerPool.Stop()
	s.Equal(signal.EventDiscoveryStarted, s.getPoolEvent(poolEvents))
	connected := s.getPeerFromEvent(events, p2p.PeerEventTypeAdd)
	s.Equal(s.peers[0].Self().ID, connected)
	s.Equal(signal.EventDiscoveryStopped, s.getPoolEvent(poolEvents))
	s.Require().Nil(s.peers[1].DiscV5)

	s.Require().Equal(signal.EventDiscoverySummary, s.getPoolEvent(poolEvents))
	summary := <-summaries
	s.Len(summary, 1)
	s.Contains(summary, "shh/6")
	s.Equal(summary["shh/6"], 1)

	s.peers[0].Stop()
	disconnected := s.getPeerFromEvent(events, p2p.PeerEventTypeDrop)
	s.Equal(connected, disconnected)

	s.Require().Equal(signal.EventDiscoverySummary, s.getPoolEvent(poolEvents))
	summary = <-summaries
	s.Len(summary, 0)

	s.Equal(signal.EventDiscoveryStarted, s.getPoolEvent(poolEvents))
	s.Require().NotNil(s.peers[1].DiscV5)
	register = NewRegister(topic)
	s.Require().NoError(register.Start(s.peers[2]))
	defer register.Stop()
	s.Equal(s.peers[2].Self().ID, s.getPeerFromEvent(events, p2p.PeerEventTypeAdd))

	s.Equal(signal.EventDiscoveryStopped, s.getPoolEvent(poolEvents))
	s.Require().Equal(signal.EventDiscoverySummary, s.getPoolEvent(poolEvents))
	summary = <-summaries
	s.Len(summary, 1)
	s.Contains(summary, "shh/6")
	s.Equal(summary["shh/6"], 1)
}

func (s *PeerPoolSimulationSuite) TearDown() {
	s.bootnode.Stop()
	for _, p := range s.peers {
		p.Stop()
	}
}

// TestMaxPeersOverflow verifies that following scenario will not occur:
// - found peer A and B in the same kademlia cycle
// - process peer A
// - max limit is reached -> closed discv5 and set it to nil
// - process peer B
// - panic because discv5 is nil!!!
func TestMaxPeersOverflow(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	peer := &p2p.Server{
		Config: p2p.Config{
			PrivateKey:  key,
			DiscoveryV5: true,
			NoDiscovery: true,
		},
	}
	require.NoError(t, peer.Start())
	defer peer.Stop()
	require.NotNil(t, peer.DiscV5)
	pool := NewPeerPool(nil, DefaultFastSync, DefaultSlowSync, nil, true)
	pool.handleAddedEvent(peer, &p2p.PeerEvent{})
	require.Nil(t, peer.DiscV5)
	pool.handleAddedEvent(peer, &p2p.PeerEvent{})
}
