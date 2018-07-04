package peers

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/suite"
)

type CacheOnlyTopicPoolSuite struct {
	suite.Suite

	peer      *p2p.Server
	topicPool *CacheOnlyTopicPool
}

func TestCacheOnlyTopicPoolSuite(t *testing.T) {
	suite.Run(t, new(CacheOnlyTopicPoolSuite))
}

func (s *CacheOnlyTopicPoolSuite) SetupTest() {
	maxCachedPeersMultiplier = 1
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
	limits := params.NewLimits(1, 2)
	cache, err := newInMemoryCache()
	s.Require().NoError(err)
	s.topicPool = newCacheOnlyTopicPool(&DiscV5{}, MailServerDiscoveryTopic, limits, 100*time.Millisecond, 200*time.Millisecond, cache)
	s.topicPool.running = 1
	// This is a buffered channel to simplify testing.
	// If your test generates more than 10 mode changes,
	// override this `period` field or consume from it
	// using `AssertConsumed()`.
	s.topicPool.period = make(chan time.Duration, 10)
}

func (s *CacheOnlyTopicPoolSuite) TearDown() {
	s.peer.Stop()
}

func (s *CacheOnlyTopicPoolSuite) TestReplacementPeerIsCounted() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.topicPool.connectedPeers = map[discv5.NodeID]*peerInfo{}
	s.True(s.topicPool.MaxReached())
}

func (s *CacheOnlyTopicPoolSuite) TestConfirmAddedSignals() {
	sentNodeID := ""
	sentTopic := ""
	sendEnodeDiscoveryCompleted = func(enode, topic string) {
		sentNodeID = enode
		sentTopic = topic
	}

	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.Equal("01000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", sentNodeID)
	s.Equal(MailServerDiscoveryTopic, sentTopic)
}
