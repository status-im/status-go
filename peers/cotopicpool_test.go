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
	topicPool *cacheOnlyTopicPool
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
	t := newTopicPool(nil, MailServerDiscoveryTopic, limits, 100*time.Millisecond, 200*time.Millisecond, cache)
	s.topicPool = newCacheOnlyTopicPool(t)
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
	s.True(s.topicPool.MaxReached())

	// When we stop searching for peers (when Max limit is reached)
	s.topicPool.StopSearch(s.peer)
	s.True(s.topicPool.MaxReached())
	s.Equal(s.topicPool.limits.Max, 0)
	s.Equal(s.topicPool.limits.Min, 0)

	// Then we should drop all connected peers
	s.Equal(len(s.topicPool.connectedPeers), 0)

	// And cached peers should remain
	cachedPeers := s.topicPool.cache.GetPeersRange(s.topicPool.topic, s.topicPool.maxCachedPeers)
	s.Equal(len(cachedPeers), 1)
}

func (s *CacheOnlyTopicPoolSuite) TestConfirmAddedSignals() {
	sentNodeID := ""
	sentTopic := ""
	sendEnodeDiscovered = func(enode, topic string) {
		sentNodeID = enode
		sentTopic = topic
	}

	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.Equal((discv5.NodeID{1}).String(), sentNodeID)
	s.Equal(MailServerDiscoveryTopic, sentTopic)
}
