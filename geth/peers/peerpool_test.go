package peers

import (
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/geth/params"
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

	s.peers = make([]*p2p.Server, 2)
	for i := range s.peers {
		key, _ := crypto.GenerateKey()
		peer := &p2p.Server{
			Config: p2p.Config{
				MaxPeers:         10,
				Name:             common.MakeName("peer-"+strconv.Itoa(i), "1.0"),
				ListenAddr:       fmt.Sprintf("0.0.0.0:%d", port),
				PrivateKey:       key,
				DiscoveryV5:      true,
				NoDiscovery:      true,
				BootstrapNodesV5: []*discv5.Node{bootnodeV5},
			},
		}
		port++
		s.NoError(peer.Start())
		s.peers[i] = peer
	}
}

func (s *PeerPoolSimulationSuite) TestSingleTopicDiscovery() {
	topic := discv5.Topic("cap=test")
	expectedConnections := 1
	// simulation should only rely on fast sync
	config := map[discv5.Topic]params.Limits{
		topic: {expectedConnections, expectedConnections},
	}
	peerPool := NewPeerPool(config, 100*time.Millisecond, 100*time.Millisecond, nil, false)
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
	connected := 0
	for {
		select {
		case ev := <-events:
			if ev.Type == p2p.PeerEventTypeAdd {
				connected++
			}
		case <-time.After(5 * time.Second):
			s.Require().FailNowf("waiting for peers timed out", strconv.Itoa(connected))
		}
		if connected == expectedConnections {
			break
		}
	}
}

func (s *PeerPoolSimulationSuite) TearDown() {
	s.bootnode.Stop()
	for _, p := range s.peers {
		p.Stop()
	}
}
