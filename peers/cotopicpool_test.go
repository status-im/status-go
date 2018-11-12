package peers

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/suite"
	"github.com/ethereum/go-ethereum/p2p/enode"
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
	s.topicPool = newCacheOnlyTopicPool(t, &testTrueVerifier{})
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
	key1, _:=crypto.GenerateKey()
	peer1 := discv5.NewNode(discv5.PubkeyID(&key1.PublicKey), s.peer.Self().IP(), 32311, 32311)
	key2, _:=crypto.GenerateKey()
	peer2 := discv5.NewNode(discv5.PubkeyID(&key2.PublicKey), s.peer.Self().IP(), 32311, 32311)
	s.topicPool.processFoundNode(s.peer, peer1)
	s.topicPool.processFoundNode(s.peer, peer2)

	n,err:=Discv5ToEnode(*peer1)
	s.NoError(err)
	s.topicPool.ConfirmAdded(s.peer, n.ID())

	n,err=Discv5ToEnode(*peer1)
	s.NoError(err)
	s.topicPool.ConfirmAdded(s.peer, n.ID())
	s.True(s.topicPool.MaxReached())

	// When we stop searching for peers (when Max limit is reached)
	s.topicPool.StopSearch(s.peer)
	s.True(s.topicPool.MaxReached())
	s.Equal(0, s.topicPool.limits.Max)
	s.Equal(0, s.topicPool.limits.Min)

	// Then we should drop all connected peers
	s.Equal(len(s.topicPool.connectedPeers), 0)

	// And cached peers should remain
	cachedPeers := s.topicPool.cache.GetPeersRange(s.topicPool.topic, s.topicPool.maxCachedPeers)
	s.Equal(1, len(cachedPeers))
}

func (s *CacheOnlyTopicPoolSuite) TestConfirmAddedSignals() {
	sentNodeID := ""
	sentTopic := ""
	sendEnodeDiscovered = func(enode, topic string) {
		sentNodeID = enode
		sentTopic = topic
	}

	key1, _:=crypto.GenerateKey()
	nodeID:=discv5.PubkeyID(&key1.PublicKey)
	peer1 := discv5.NewNode(nodeID, s.peer.Self().IP(), 32311, 32311)
	n,err:=Discv5ToEnode(*peer1)
	s.NoError(err)

	s.topicPool.ConfirmAdded(s.peer, n.ID())

	s.Equal(n.ID().String(), sentNodeID)
	s.Equal(MailServerDiscoveryTopic, sentTopic)
}

func (s *CacheOnlyTopicPoolSuite) TestNotTrustedPeer() {
	var signalCalled bool
	sendEnodeDiscovered = func(_, _ string) { signalCalled = true }

	s.topicPool.limits = params.NewLimits(1, 1)
	s.topicPool.maxCachedPeers = 1
	s.topicPool.verifier = &testFalseVerifier{}

	key1, _:=crypto.GenerateKey()
	foundPeer := discv5.NewNode(discv5.PubkeyID(&key1.PublicKey), s.peer.Self().IP(), 32311, 32311)
	s.topicPool.processFoundNode(s.peer, foundPeer)

	n,err:=Discv5ToEnode(*foundPeer)
	s.NoError(err)
	s.topicPool.ConfirmAdded(s.peer, n.ID())

	s.False(signalCalled)
	// limits should not change
	s.Equal(1, s.topicPool.limits.Max)
	s.Equal(1, s.topicPool.limits.Min)
	// not verified peer shoud not be added to the cache
	s.Equal(0, len(s.topicPool.cache.GetPeersRange(s.topicPool.topic, s.topicPool.limits.Max)))
}

type testTrueVerifier struct{}

func (v *testTrueVerifier) VerifyNode(context.Context, enode.ID) bool {
	return true
}

type testFalseVerifier struct{}

func (v *testFalseVerifier) VerifyNode(context.Context, enode.ID) bool {
	return false
}
