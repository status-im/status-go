package peers

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
)

type ProxyPoolSimulationSuite struct {
	suite.Suite

	topic discv5.Topic
	port  uint16

	bootnodeEthereum   *p2p.Server
	bootnodeEthereumV5 *discv5.Node
	bootnodeStatus     *p2p.Server
	bootnodeStatusV5   *discv5.Node
}

func TestProxyPoolSimulationSuite(t *testing.T) {
	s := ProxyPoolSimulationSuite{
		topic: discv5.Topic("some-topic"),
		port:  33730,
	}
	suite.Run(t, &s)
}

func (s *ProxyPoolSimulationSuite) nextPort() uint16 {
	s.port++
	return s.port
}

func (s *ProxyPoolSimulationSuite) createAndStartBootnode(aName string, version int) (*p2p.Server, uint16, error) {
	port := s.nextPort()
	name := common.MakeName(aName, "1.0")
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, 0, err
	}

	bootnode := &p2p.Server{
		Config: p2p.Config{
			MaxPeers:           10,
			Name:               name,
			ListenAddr:         fmt.Sprintf("0.0.0.0:%d", port),
			PrivateKey:         key,
			DiscoveryV5:        true,
			DiscoveryV5Network: version,
			NoDiscovery:        true,
		},
	}

	return bootnode, port, bootnode.Start()
}

func (s *ProxyPoolSimulationSuite) createAndStartPeerAndDiscV5(aName string, version int, bootnodes []*discv5.Node) (*p2p.Server, *DiscV5, error) {
	port := s.nextPort()
	name := common.MakeName(aName, "1.0")
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	peer := &p2p.Server{
		Config: p2p.Config{
			MaxPeers:    10,
			Name:        name,
			ListenAddr:  fmt.Sprintf("0.0.0.0:%d", port),
			PrivateKey:  key,
			NoDiscovery: true,
		},
	}
	if err := peer.Start(); err != nil {
		return nil, nil, err
	}

	discV5 := NewDiscV5(key, peer.ListenAddr, version, bootnodes)

	return peer, discV5, discV5.Start()
}

func (s *ProxyPoolSimulationSuite) SetupTest() {
	var (
		port uint16
		err  error
	)

	s.bootnodeEthereum, port, err = s.createAndStartBootnode("bootstrapEthereum", discv5.Version)
	s.Require().NoError(err)
	s.bootnodeEthereumV5 = discv5.NewNode(
		s.bootnodeEthereum.DiscV5.Self().ID,
		net.ParseIP("127.0.0.1"),
		port,
		port,
	)

	s.bootnodeStatus, port, err = s.createAndStartBootnode("bootstrapStatus", discv5.StatusVersion)
	s.Require().NoError(err)
	s.bootnodeStatusV5 = discv5.NewNode(
		s.bootnodeStatus.DiscV5.Self().ID,
		net.ParseIP("127.0.0.1"),
		port,
		port,
	)
}

func (s *ProxyPoolSimulationSuite) TearDown() {
	s.bootnodeEthereum.Stop()
	s.bootnodeStatus.Stop()
}

func (s *ProxyPoolSimulationSuite) TestDiscoveryWithBootnode() {
	registerPeer, registerDiscV5, err := s.createAndStartPeerAndDiscV5("register", discv5.Version, []*discv5.Node{s.bootnodeEthereumV5})
	s.Require().NoError(err)

	register := NewRegister(registerDiscV5, s.topic)
	s.Require().NoError(register.Start())

	seekerPeer, seekerDiscV5, err := s.createAndStartPeerAndDiscV5("seeker", discv5.Version, []*discv5.Node{s.bootnodeEthereumV5})
	s.Require().NoError(err)

	events := make(chan *p2p.PeerEvent, 20)
	subscription := seekerPeer.SubscribeEvents(events)
	defer subscription.Unsubscribe()

	seekerCache, err := newInMemoryCache()
	s.Require().NoError(err)
	seekerPool := NewPeerPool(
		seekerDiscV5,
		map[discv5.Topic]params.Limits{s.topic: {Min: 1, Max: 1}},
		seekerCache,
		&Options{100 * time.Millisecond, 100 * time.Millisecond, 0, true, 100 * time.Millisecond, nil, nil},
	)
	s.Require().NoError(seekerPool.Start(seekerPeer))

	connectedPeer := s.getPeerFromEvent(events, p2p.PeerEventTypeAdd)
	s.Equal(registerPeer.Self().ID, connectedPeer)
}

// TestDiscoveryProxy tests a scenario where a peer registers its topic in the network A
// and is discovered in the network B through a proxy pool.
func (s *ProxyPoolSimulationSuite) TestDiscoveryProxy() {
	// Register topic in the network A (Ethereum network in this example).
	registerPeer, registerDiscV5, err := s.createAndStartPeerAndDiscV5("register", discv5.Version, []*discv5.Node{s.bootnodeEthereumV5})
	s.Require().NoError(err)

	register := NewRegister(registerDiscV5, s.topic)
	s.Require().NoError(register.Start())

	// Create another peer that can act as a server node.
	// It is connected to the network B (Status network in this example),
	// but it also has a proxy Discovery connected to the network A.
	// Proxy pool together with proxy topic forward peers selected by topic
	// from the network A to the network B.
	serverPeer, serverDiscV5, err := s.createAndStartPeerAndDiscV5("server", discv5.StatusVersion, []*discv5.Node{s.bootnodeStatusV5})
	s.Require().NoError(err)

	proxyKey, _ := crypto.GenerateKey()
	proxyDiscV5 := NewDiscV5(proxyKey, fmt.Sprintf("0.0.0.0:%d", s.nextPort()), discv5.Version, []*discv5.Node{s.bootnodeEthereumV5})
	s.Require().NoError(proxyDiscV5.Start())

	proxyCache, err := newInMemoryCache()
	s.Require().NoError(err)
	proxyPool := NewPeerPool(
		proxyDiscV5,
		nil,
		proxyCache,
		&Options{100 * time.Millisecond, 100 * time.Millisecond, 0, true, 100 * time.Millisecond, []discv5.Topic{s.topic}, serverDiscV5},
	)
	s.Require().NoError(proxyPool.Start(serverPeer))

	// Create an equivalent of mobile node that connects to the network B (Status network).
	// It is a regular peer that seeks for peers with a given topic.
	seekerPeer, seekerDiscV5, err := s.createAndStartPeerAndDiscV5("mobile", discv5.StatusVersion, []*discv5.Node{s.bootnodeStatusV5})
	s.Require().NoError(err)

	events := make(chan *p2p.PeerEvent, 20)
	subscription := seekerPeer.SubscribeEvents(events)
	defer subscription.Unsubscribe()

	seekerCache, err := newInMemoryCache()
	s.Require().NoError(err)
	seekerPool := NewPeerPool(
		seekerDiscV5,
		map[discv5.Topic]params.Limits{s.topic: {Min: 1, Max: 1}},
		seekerCache,
		&Options{100 * time.Millisecond, 100 * time.Millisecond, 0, true, 100 * time.Millisecond, nil, nil},
	)
	s.Require().NoError(seekerPool.Start(seekerPeer))

	connectedPeer := s.getPeerFromEvent(events, p2p.PeerEventTypeAdd)
	s.Equal(registerPeer.Self().ID, connectedPeer)
}

func (s *ProxyPoolSimulationSuite) getPeerFromEvent(events <-chan *p2p.PeerEvent, etype p2p.PeerEventType) (nodeID discover.NodeID) {
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
