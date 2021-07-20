package peers

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/params"
)

type TopicPoolSuite struct {
	suite.Suite

	peer      *p2p.Server
	topicPool *TopicPool
}

func TestTopicPoolSuite(t *testing.T) {
	suite.Run(t, new(TopicPoolSuite))
}

func (s *TopicPoolSuite) createDiscV5Node(ip net.IP, port uint16) (enode.ID, *discv5.Node) {
	id, err := crypto.GenerateKey()
	s.Require().NoError(err)
	nodeID := enode.PubkeyToIDV4(&id.PublicKey)
	nodeV5 := discv5.NewNode(discv5.PubkeyID(&id.PublicKey), ip, port, port)
	return nodeID, nodeV5
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
	s.topicPool = newTopicPool(nil, topic, limits, 100*time.Millisecond, 200*time.Millisecond, cache)
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

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	nodeID2, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))

	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID2)

	cached := s.topicPool.cache.GetPeersRange(s.topicPool.topic, 10)
	s.Contains(cached, peer1)
	s.Contains(cached, peer2)

	s.topicPool.ConfirmDropped(s.peer, nodeID2)
	cached = s.topicPool.cache.GetPeersRange(s.topicPool.topic, 10)
	s.Contains(cached, peer1)
	s.Contains(cached, peer2)

	// A peer that drops by itself, should be removed from the cache.
	s.topicPool.ConfirmDropped(s.peer, nodeID1)
	s.Equal([]*discv5.Node{peer2}, s.topicPool.cache.GetPeersRange(s.topicPool.topic, 10))
}

func (s *TopicPoolSuite) TestSyncSwitches() {
	nodeID, peer := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer))
	s.topicPool.ConfirmAdded(s.peer, nodeID)
	s.AssertConsumed(s.topicPool.period, s.topicPool.slowMode, time.Second)
	s.NotNil(s.topicPool.connectedPeers[nodeID])
	s.topicPool.ConfirmDropped(s.peer, nodeID)
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
	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID2, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID3, peer3 := s.createDiscV5Node(s.peer.Self().IP(), 32311)

	// add 3 nodes and confirm connection for 1 and 2
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer3))
	s.Len(s.topicPool.pendingPeers, 3)
	s.Len(s.topicPool.discoveredPeersQueue, 0)
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.Contains(s.topicPool.connectedPeers, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID2)
	s.Contains(s.topicPool.connectedPeers, nodeID2)
	s.topicPool.ConfirmAdded(s.peer, nodeID3)
	s.topicPool.ConfirmDropped(s.peer, nodeID3)
	s.Contains(s.topicPool.pendingPeers, nodeID3)
	s.Len(s.topicPool.pendingPeers, 1)
	s.Len(s.topicPool.discoveredPeersQueue, 1)
	// drop peer1
	s.True(s.topicPool.ConfirmDropped(s.peer, nodeID1))
	s.NotContains(s.topicPool.connectedPeers, nodeID1)
	// add peer from the pool
	s.Equal(peer3.ID, s.topicPool.AddPeerFromTable(s.peer).ID)
	s.Len(s.topicPool.pendingPeers, 1)
	s.Len(s.topicPool.discoveredPeersQueue, 0)
}

type mockConstantClock struct{}

func (mockConstantClock) Now() time.Time { return time.Unix(0, 0) }

// Regression test for https://github.com/status-im/nim-status-client/issues/522
func (s *TopicPoolSuite) TestNewPeerLackOfClockPrecision() {
	_, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	_, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	_, peer3 := s.createDiscV5Node(s.peer.Self().IP(), 32311)

	s.topicPool.maxPendingPeers = 2

	s.topicPool.clock = mockConstantClock{}

	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer3))
}

func (s *TopicPoolSuite) TestRequestedDoesntRemove() {
	// max limit is 1 because we test that 2nd peer will stay in local table
	// when we request to drop it
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID2, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)

	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID2)
	s.False(s.topicPool.connectedPeers[nodeID1].dismissed)
	s.True(s.topicPool.connectedPeers[nodeID2].dismissed)
	s.topicPool.ConfirmDropped(s.peer, nodeID2)
	s.Contains(s.topicPool.pendingPeers, nodeID2)
	s.NotContains(s.topicPool.connectedPeers, nodeID2)
	s.topicPool.ConfirmDropped(s.peer, nodeID1)
	s.NotContains(s.topicPool.pendingPeers, nodeID1)
	s.NotContains(s.topicPool.connectedPeers, nodeID1)
}

func (s *TopicPoolSuite) TestTheMostRecentPeerIsSelected() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID2, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID3, peer3 := s.createDiscV5Node(s.peer.Self().IP(), 32311)

	// after these operations, peer1 is confirmed and peer3 and peer2
	// was added to the pool; peer3 is the most recent one
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer3))
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID2)
	s.topicPool.ConfirmAdded(s.peer, nodeID3)

	s.topicPool.ConfirmDropped(s.peer, nodeID2)
	s.topicPool.ConfirmDropped(s.peer, nodeID3)
	// peer1 has dropped
	s.topicPool.ConfirmDropped(s.peer, nodeID1)
	// and peer3 is take from the pool as the most recent
	s.True(s.topicPool.pendingPeers[nodeID2].discoveredTime.Before(s.topicPool.pendingPeers[nodeID3].discoveredTime))
	s.Equal(peer3.ID, s.topicPool.AddPeerFromTable(s.peer).ID)
}

func (s *TopicPoolSuite) TestSelectPeerAfterMaxLimit() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID2, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID3, peer3 := s.createDiscV5Node(s.peer.Self().IP(), 32311)

	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID2)
	s.topicPool.ConfirmDropped(s.peer, nodeID2)
	s.Len(s.topicPool.pendingPeers, 1)
	s.Contains(s.topicPool.pendingPeers, nodeID2)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer3))
	s.Len(s.topicPool.pendingPeers, 2)
	s.Contains(s.topicPool.pendingPeers, nodeID3)
	s.Equal(peer3, s.topicPool.AddPeerFromTable(s.peer))
}

func (s *TopicPoolSuite) TestReplacementPeerIsCounted() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID2, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID2)
	s.topicPool.ConfirmDropped(s.peer, nodeID2)
	s.topicPool.ConfirmDropped(s.peer, nodeID1)

	s.NotContains(s.topicPool.pendingPeers, nodeID1)
	s.NotContains(s.topicPool.connectedPeers, nodeID1)
	s.Contains(s.topicPool.pendingPeers, nodeID2)
	s.topicPool.pendingPeers[nodeID2].added = true
	s.topicPool.ConfirmAdded(s.peer, nodeID2)
	s.True(s.topicPool.MaxReached())
}

func (s *TopicPoolSuite) TestPeerDontAddTwice() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	_, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	// peer2 already added to p2p server no reason to add it again
	s.Nil(s.topicPool.AddPeerFromTable(s.peer))
}

func (s *TopicPoolSuite) TestMaxCachedPeers() {
	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 3
	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID2, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID3, peer3 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer2))
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer3))
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID2)
	s.topicPool.ConfirmAdded(s.peer, nodeID3)

	s.Equal(3, len(s.topicPool.connectedPeers))
	s.False(s.topicPool.connectedPeers[nodeID1].dismissed)
	s.True(s.topicPool.connectedPeers[nodeID2].dismissed)
	s.True(s.topicPool.connectedPeers[nodeID3].dismissed)

	cached := s.topicPool.cache.GetPeersRange(s.topicPool.topic, 5)
	s.Equal(3, len(cached))

	cachedMap := make(map[discv5.NodeID]*discv5.Node)
	for _, peer := range cached {
		cachedMap[peer.ID] = peer
	}

	s.topicPool.ConfirmDropped(s.peer, nodeID2)
	s.topicPool.ConfirmDropped(s.peer, nodeID3)

	s.Contains(cachedMap, peer1.ID)
	s.Contains(cachedMap, peer2.ID)
	s.Contains(cachedMap, peer3.ID)

	s.Contains(s.topicPool.connectedPeers, nodeID1)
	s.NotContains(s.topicPool.connectedPeers, nodeID2)
	s.NotContains(s.topicPool.connectedPeers, nodeID3)

	s.NotContains(s.topicPool.pendingPeers, nodeID1)
	s.Contains(s.topicPool.pendingPeers, nodeID2)
	s.Contains(s.topicPool.pendingPeers, nodeID3)

	s.True(s.topicPool.maxCachedPeersReached())
	cached = s.topicPool.cache.GetPeersRange(s.topicPool.topic, 5)
	s.Equal(3, len(cached))
}

func (s *TopicPoolSuite) TestMaxPendingPeers() {
	s.topicPool.maxPendingPeers = 2

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID2, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	nodeID3, peer3 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	pk1, _ := peer1.ID.Pubkey()
	pk2, _ := peer2.ID.Pubkey()
	pk3, _ := peer3.ID.Pubkey()

	s.topicPool.addToPendingPeers(&peerInfo{discoveredTime: time.Now(), node: peer1, publicKey: pk1})
	s.topicPool.addToPendingPeers(&peerInfo{discoveredTime: time.Now(), node: peer2, publicKey: pk2})
	s.topicPool.addToPendingPeers(&peerInfo{discoveredTime: time.Now(), node: peer3, publicKey: pk3})

	s.Equal(2, len(s.topicPool.pendingPeers))
	s.Require().NotContains(s.topicPool.pendingPeers, nodeID1)
	s.Require().Contains(s.topicPool.pendingPeers, nodeID2)
	s.Require().Contains(s.topicPool.pendingPeers, nodeID3)

	// maxPendingPeers = 0 means no limits.
	s.topicPool.maxPendingPeers = 0
	s.topicPool.addToPendingPeers(&peerInfo{discoveredTime: time.Now(), node: peer1, publicKey: pk1})

	s.Equal(3, len(s.topicPool.pendingPeers))
	s.Require().Contains(s.topicPool.pendingPeers, nodeID1)
	s.Require().Contains(s.topicPool.pendingPeers, nodeID2)
	s.Require().Contains(s.topicPool.pendingPeers, nodeID3)
}

func (s *TopicPoolSuite) TestQueueDuplicatePeers() {
	_, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	_, peer2 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	pk1, _ := peer1.ID.Pubkey()
	pk2, _ := peer2.ID.Pubkey()
	peerInfo1 := &peerInfo{discoveredTime: time.Now(), node: peer1, publicKey: pk1}
	peerInfo2 := &peerInfo{discoveredTime: time.Now(), node: peer2, publicKey: pk2}

	s.topicPool.addToPendingPeers(&peerInfo{discoveredTime: time.Now(), node: peer1, publicKey: pk1})
	s.topicPool.addToPendingPeers(&peerInfo{discoveredTime: time.Now(), node: peer2, publicKey: pk2})
	s.topicPool.addToQueue(peerInfo1)
	s.topicPool.addToQueue(peerInfo2)

	s.Equal(2, len(s.topicPool.discoveredPeersQueue))
	s.Equal(2, len(s.topicPool.discoveredPeers))

	s.topicPool.addToQueue(peerInfo1)

	s.Equal(2, len(s.topicPool.discoveredPeersQueue))
	s.Equal(2, len(s.topicPool.discoveredPeers))

	peer := s.topicPool.popFromQueue()

	s.Equal(1, len(s.topicPool.discoveredPeersQueue))
	s.Equal(1, len(s.topicPool.discoveredPeers))
	s.Require().NotContains(s.topicPool.discoveredPeers, peer.NodeID())
}

func (s *TopicPoolSuite) TestNewTopicPoolInterface() {
	limits := params.NewLimits(1, 2)
	cache, err := newInMemoryCache()
	s.Require().NoError(err)

	topic := discv5.Topic("cap=cap1")
	t := newTopicPool(nil, topic, limits, 100*time.Millisecond, 200*time.Millisecond, cache)
	s.IsType(&TopicPool{}, t)

	tp := newTopicPool(nil, MailServerDiscoveryTopic, limits, 100*time.Millisecond, 200*time.Millisecond, cache)
	cacheTP := newCacheOnlyTopicPool(tp, &testTrueVerifier{})
	s.IsType(&cacheOnlyTopicPool{}, cacheTP)
}

func (s *TopicPoolSuite) TestIgnoreInboundConnection() {
	s.topicPool.limits = params.NewLimits(0, 0)
	s.topicPool.maxCachedPeers = 0

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Contains(s.topicPool.pendingPeers, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.Contains(s.topicPool.pendingPeers, nodeID1)
	s.False(s.topicPool.pendingPeers[nodeID1].dismissed)
	s.NotContains(s.topicPool.connectedPeers, nodeID1)
}

func (s *TopicPoolSuite) TestConnectedButRemoved() {
	s.topicPool.limits = params.NewLimits(0, 0)
	s.topicPool.maxCachedPeers = 1

	nodeID1, peer1 := s.createDiscV5Node(s.peer.Self().IP(), 32311)
	s.Require().NoError(s.topicPool.processFoundNode(s.peer, peer1))
	s.Contains(s.topicPool.pendingPeers, nodeID1)
	s.topicPool.ConfirmAdded(s.peer, nodeID1)
	s.Contains(s.topicPool.connectedPeers, nodeID1)
	s.False(s.topicPool.ConfirmDropped(s.peer, nodeID1))
	s.False(s.topicPool.pendingPeers[nodeID1].added)
}
