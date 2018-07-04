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

type TopicPoolSuite struct {
	suite.Suite

	peer      *p2p.Server
	topicPool *TopicPool
}

func TestTopicPoolSuite(t *testing.T) {
	suite.Run(t, new(TopicPoolSuite))
}

func (s *TopicPoolSuite) SetupTest() {
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
	topic := discv5.Topic("cap=cap1")
	limits := params.NewLimits(1, 2)
	cache, err := newInMemoryCache()
	s.Require().NoError(err)
	s.topicPool = newTopicPool(&DiscV5{}, topic, limits, 100*time.Millisecond, 200*time.Millisecond, cache)
	s.topicPool.running = 1
	// This is a buffered channel to simplify testing.
	// If your test generates more than 10 mode changes,
	// override this `period` field or consume from it
	// using `AssertConsumed()`.
	s.topicPool.period = make(chan time.Duration, 10)
}

func (s *TopicPoolSuite) TearDown() {
	s.peer.Stop()
}

func (s *TopicPoolSuite) AssertConsumed(channel <-chan time.Duration, expected time.Duration, timeout time.Duration) {
	select {
	case received := <-channel:
		s.Equal(expected, received)
	case <-time.After(timeout):
		s.FailNow("timed out waiting")
	}
}

func (s *TopicPoolSuite) TestUsingCache() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.Equal([]*discv5.Node{peer1}, s.topicPool.cache.GetPeersRange(s.topicPool.topic, 10))

	// Add a new peer which exceeds the upper limit.
	// It should still be added to the cache and
	// not removed when dropped.
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.Equal([]*discv5.Node{peer1, peer2}, s.topicPool.cache.GetPeersRange(s.topicPool.topic, 10))
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer2.ID))
	s.Equal([]*discv5.Node{peer1, peer2}, s.topicPool.cache.GetPeersRange(s.topicPool.topic, 10))

	// A peer that drops by itself, should be removed from the cache.
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer1.ID))
	s.Equal([]*discv5.Node{peer2}, s.topicPool.cache.GetPeersRange(s.topicPool.topic, 10))
}

func (s *TopicPoolSuite) TestSyncSwitches() {
	testPeer := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, testPeer)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(testPeer.ID))
	s.AssertConsumed(s.topicPool.period, s.topicPool.slowMode, time.Second)
	s.NotNil(s.topicPool.connectedPeers[testPeer.ID])
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(testPeer.ID))
	s.AssertConsumed(s.topicPool.period, s.topicPool.fastMode, time.Second)
}

func (s *TopicPoolSuite) TestTimeoutFastMode() {
	s.topicPool.fastModeTimeout = time.Millisecond * 50

	// set fast mode
	s.topicPool.mu.Lock()
	s.topicPool.setSyncMode(s.topicPool.fastMode)
	s.topicPool.mu.Unlock()
	s.Equal(s.topicPool.fastMode, <-s.topicPool.period)

	// switch to slow mode after `fastModeTimeout`
	select {
	case mode := <-s.topicPool.period:
		s.Equal(s.topicPool.slowMode, mode)
	case <-time.After(s.topicPool.fastModeTimeout * 2):
		s.FailNow("timed out")
	}
}

func (s *TopicPoolSuite) TestSetSyncMode() {
	s.topicPool.fastModeTimeout = 0

	// set fast mode
	s.topicPool.setSyncMode(s.topicPool.fastMode)
	s.Equal(s.topicPool.fastMode, <-s.topicPool.period)
	s.Equal(s.topicPool.fastMode, s.topicPool.currentMode)

	// skip setting the same mode
	s.topicPool.setSyncMode(s.topicPool.fastMode)
	select {
	case <-s.topicPool.period:
		s.FailNow("should not have update the mode")
	default:
		// pass
	}

	// switch to slow mode
	cancel := make(chan struct{})
	s.topicPool.fastModeTimeoutCancel = cancel // should be set to nil
	s.topicPool.setSyncMode(s.topicPool.slowMode)
	s.Equal(s.topicPool.slowMode, <-s.topicPool.period)
	s.Equal(s.topicPool.slowMode, s.topicPool.currentMode)
	select {
	case <-cancel:
		s.Nil(s.topicPool.fastModeTimeoutCancel)
	default:
		s.FailNow("cancel should be closed")
	}
}

func (s *TopicPoolSuite) TestNewPeerSelectedOnDrop() {
	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	peer3 := discv5.NewNode(discv5.NodeID{3}, s.peer.Self().IP, 32311, 32311)
	// add 3 nodes and confirm connection for 1 and 2
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.processFoundNode(s.peer, peer3)
	s.Len(s.topicPool.pendingPeers, 3)
	s.Len(s.topicPool.discoveredPeersQueue, 0)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.Contains(s.topicPool.connectedPeers, peer1.ID)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.Contains(s.topicPool.connectedPeers, peer2.ID)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer3.ID))
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer3.ID))
	s.Contains(s.topicPool.pendingPeers, peer3.ID)
	s.Len(s.topicPool.pendingPeers, 1)
	s.Len(s.topicPool.discoveredPeersQueue, 1)
	// drop peer1
	s.True(s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer1.ID)))
	s.NotContains(s.topicPool.connectedPeers, peer1.ID)
	// add peer from the pool
	s.Equal(peer3.ID, s.topicPool.AddPeerFromTable(s.peer).ID)
	s.Len(s.topicPool.pendingPeers, 1)
	s.Len(s.topicPool.discoveredPeersQueue, 0)
}

func (s *TopicPoolSuite) TestRequestedDoesntRemove() {
	// max limit is 1 because we test that 2nd peer will stay in local table
	// when we request to drop it
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1
	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.False(s.topicPool.connectedPeers[peer1.ID].dismissed)
	s.True(s.topicPool.connectedPeers[peer2.ID].dismissed)
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer2.ID))
	s.Contains(s.topicPool.pendingPeers, peer2.ID)
	s.NotContains(s.topicPool.connectedPeers, peer2.ID)
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer1.ID))
	s.NotContains(s.topicPool.pendingPeers, peer1.ID)
	s.NotContains(s.topicPool.connectedPeers, peer1.ID)
}

func (s *TopicPoolSuite) TestTheMostRecentPeerIsSelected() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	peer3 := discv5.NewNode(discv5.NodeID{3}, s.peer.Self().IP, 32311, 32311)

	// after these operations, peer1 is confirmed and peer3 and peer2
	// was added to the pool; peer3 is the most recent one
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.processFoundNode(s.peer, peer3)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer3.ID))

	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer2.ID))
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer3.ID))
	// peer1 has dropped
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer1.ID))
	// and peer3 is take from the pool as the most recent
	s.True(s.topicPool.pendingPeers[peer2.ID].discoveredTime < s.topicPool.pendingPeers[peer3.ID].discoveredTime)
	s.Equal(peer3.ID, s.topicPool.AddPeerFromTable(s.peer).ID)
}

func (s *TopicPoolSuite) TestSelectPeerAfterMaxLimit() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	peer3 := discv5.NewNode(discv5.NodeID{3}, s.peer.Self().IP, 32311, 32311)

	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer2.ID))
	s.Len(s.topicPool.pendingPeers, 1)
	s.Contains(s.topicPool.pendingPeers, peer2.ID)
	s.topicPool.processFoundNode(s.peer, peer3)
	s.Len(s.topicPool.pendingPeers, 2)
	s.Contains(s.topicPool.pendingPeers, peer3.ID)
	s.Equal(peer3, s.topicPool.AddPeerFromTable(s.peer))
}

func (s *TopicPoolSuite) TestReplacementPeerIsCounted() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer2.ID))
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer1.ID))

	s.NotContains(s.topicPool.pendingPeers, peer1.ID)
	s.NotContains(s.topicPool.connectedPeers, peer1.ID)
	s.Contains(s.topicPool.pendingPeers, peer2.ID)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.True(s.topicPool.MaxReached())
}

func (s *TopicPoolSuite) TestPeerDontAddTwice() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	// peer2 already added to p2p server no reason to add it again
	s.Nil(s.topicPool.AddPeerFromTable(s.peer))
}

func (s *TopicPoolSuite) TestMaxCachedPeers() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 3
	peer1 := discv5.NewNode(discv5.NodeID{1}, s.peer.Self().IP, 32311, 32311)
	peer2 := discv5.NewNode(discv5.NodeID{2}, s.peer.Self().IP, 32311, 32311)
	peer3 := discv5.NewNode(discv5.NodeID{3}, s.peer.Self().IP, 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)
	s.topicPool.processFoundNode(s.peer, peer3)
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer1.ID))
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer2.ID))
	s.topicPool.ConfirmAdded(s.peer, discover.NodeID(peer3.ID))

	s.Equal(3, len(s.topicPool.connectedPeers))
	s.False(s.topicPool.connectedPeers[peer1.ID].dismissed)
	s.True(s.topicPool.connectedPeers[peer2.ID].dismissed)
	s.True(s.topicPool.connectedPeers[peer3.ID].dismissed)

	cached := s.topicPool.cache.GetPeersRange(s.topicPool.topic, 5)
	s.Equal(3, len(cached))

	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer2.ID))
	s.topicPool.ConfirmDropped(s.peer, discover.NodeID(peer3.ID))

	s.Equal(peer1.ID, cached[0].ID)
	s.Equal(peer2.ID, cached[1].ID)
	s.Equal(peer3.ID, cached[2].ID)
	s.Contains(s.topicPool.connectedPeers, peer1.ID)
	s.NotContains(s.topicPool.connectedPeers, peer2.ID)
	s.NotContains(s.topicPool.connectedPeers, peer3.ID)
	s.NotContains(s.topicPool.pendingPeers, peer1.ID)
	s.Contains(s.topicPool.pendingPeers, peer2.ID)
	s.Contains(s.topicPool.pendingPeers, peer3.ID)

	s.True(s.topicPool.maxCachedPeersReached())
	cached = s.topicPool.cache.GetPeersRange(s.topicPool.topic, 5)
	s.Equal(3, len(cached))
}

func (s *TopicPoolSuite) TestNewTopicPoolInterface() {
	limits := params.NewLimits(1, 2)
	cache, err := newInMemoryCache()
	s.Require().NoError(err)

	topic := discv5.Topic("cap=cap1")
	t := newTopicPool(&DiscV5{}, topic, limits, 100*time.Millisecond, 200*time.Millisecond, cache)
	s.IsType(&TopicPool{}, t)

	mst := newCacheOnlyTopicPool(&DiscV5{}, MailServerDiscoveryTopic, limits, 100*time.Millisecond, 200*time.Millisecond, cache)
	s.IsType(&CacheOnlyTopicPool{}, mst)
}
