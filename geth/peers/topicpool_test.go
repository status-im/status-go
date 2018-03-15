package peers

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/suite"
)

type TopicPoolSuite struct {
	suite.Suite

	peer      *p2p.Server
	topicPool *TopicPool
}

func TestTopicPoolSuite(t *testing.T) {
	suite.Run(t, new(TopicPoolSuite))
}

func (s *TopicPoolSuite) SetupTest() {
	key, _ := crypto.GenerateKey()
	name := common.MakeName("peer", "1.0")
	s.peer = &p2p.Server{
		Config: p2p.Config{
			MaxPeers:    10,
			Name:        name,
			ListenAddr:  "0.0.0.0:0",
			PrivateKey:  key,
			NoDiscovery: true,
		},
	}
	s.Require().NoError(s.peer.Start())
	topic := discv5.Topic("cap=cap1")
	limits := params.Limits{1, 2}
	s.topicPool = NewTopicPool(topic, limits, 100*time.Millisecond, 200*time.Millisecond)
	s.topicPool.period = make(chan time.Duration, 2)
	s.topicPool.running = 1
}

func (s *TopicPoolSuite) TearDown() {
	s.peer.Stop()
}

func (s *TopicPoolSuite) AssertConsumed(channel chan time.Duration, expected time.Duration, timeout time.Duration) {
	select {
	case received := <-channel:
		s.Equal(expected, received)
	case <-time.After(timeout):
		s.FailNow("timed out waiting")
	}
}

func (s *TopicPoolSuite) TestSyncSwitches() {
	testPeer := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, testPeer)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(testPeer.ID))
	s.AssertConsumed(s.topicPool.period, s.topicPool.slowSync, time.Second)
	s.True(s.topicPool.peers[testPeer.ID].connected)
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(testPeer.ID), p2p.DiscProtocolError.Error())
	s.AssertConsumed(s.topicPool.period, s.topicPool.fastSync, time.Second)
}

func (s *TopicPoolSuite) TestNewPeerSelectedOnDrop() {
	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	peer3 := discv5.NewNode(discv5.NodeID{3}, s.peer.Self().IP, 32311, 32311)
	// add 3 nodes and confirm connection for 1 and 2
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.processFoundNode(s.peer, peer3)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.True(s.topicPool.peers[peer1.ID].connected)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.True(s.topicPool.peers[peer2.ID].connected)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer3.ID))
	s.False(s.topicPool.peers[peer3.ID].connected)

	s.True(s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer1.ID), p2p.DiscNetworkError.Error()))
}

func (s *TopicPoolSuite) TestRequestedDoesntRemove() {
	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer1.ID), p2p.DiscRequested.Error())
	s.Contains(s.topicPool.peers, peer1.ID)
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer1.ID), p2p.DiscProtocolError.Error())
	s.NotContains(s.topicPool.peers, peer1.ID)
}
