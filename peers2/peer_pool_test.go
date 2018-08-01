package peers2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/peers"
	"github.com/status-im/status-go/t/helpers"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
)

func init() {
	if err := logutils.OverrideRootLog(true, "DEBUG", "", false); err != nil {
		panic(err)
	}
}

func newInMemoryCache() (*peers.Cache, error) {
	memdb, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return peers.NewCache(memdb), nil
}

func createServer() (*p2p.Server, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return &p2p.Server{
		Config: p2p.Config{
			PrivateKey:  key,
			MaxPeers:    10,
			NoDiscovery: true,
			ListenAddr:  "0.0.0.0:0",
		},
	}, nil
}

func TestPeerPoolRequestToAddPeer(t *testing.T) {
	var err error

	// start a peer
	peer, err := createServer()
	require.NoError(t, err)
	require.NoError(t, peer.Start())

	// start a server
	server, err := createServer()
	require.NoError(t, err)
	require.NoError(t, server.Start())

	// creat TopicPool and PeerPool
	topic := discv5.Topic("test-topic")
	topicPool := NewTopicPoolBase(nil, topic)
	peerPool := NewPeerPool([]TopicPool{topicPool}, nil)
	peerPool.Start(server)

	// subscribe to peer events
	events := make(chan *p2p.PeerEvent, 20)
	sub := peerPool.SubscribePeerEvents(events)
	defer sub.Unsubscribe()

	// add peer to server
	peerPool.RequestToAddPeer(
		topic,
		discv5.NewNode(discv5.NodeID(peer.Self().ID), peer.Self().IP, peer.Self().TCP, peer.Self().UDP),
	)
	// verify the peer was added and the PeerPool state
	nodeID, err := helpers.PeerFromEvent(events, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	require.Equal(t, 1, server.PeerCount())
	require.Equal(t, peer.Self().ID, nodeID)
	require.NotNil(t, peerPool.nodeIDToPeerInfo[nodeID])
	require.Equal(t, []discv5.Topic{topic}, peerPool.nodeIDToPeerInfo[nodeID].topics)
}

func TestPeerPoolCache(t *testing.T) {
	var err error

	// start a peer
	peer, err := createServer()
	require.NoError(t, err)
	require.NoError(t, peer.Start())

	peerV5 := discv5.NewNode(discv5.NodeID(peer.Self().ID), peer.Self().IP, peer.Self().TCP, peer.Self().UDP)

	// start a server
	server, err := createServer()
	require.NoError(t, err)
	require.NoError(t, server.Start())

	// create TopicPool and PeerPool
	topic := discv5.Topic("test-topic")
	topicPool := NewTopicPoolBase(nil, topic)
	cache, err := newInMemoryCache()
	require.NoError(t, err)
	peerPool := NewPeerPool([]TopicPool{topicPool}, cache)
	peerPool.Start(server)

	// subscribe to peer events
	events := make(chan *p2p.PeerEvent, 20)
	sub := peerPool.SubscribePeerEvents(events)
	defer sub.Unsubscribe()

	// add a peer to server
	peerPool.RequestToAddPeer(topic, peerV5)
	// verify the peer was added and the PeerPool state
	_, err = helpers.PeerFromEvent(events, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	require.Len(t, cache.GetPeersRange(topic, 10), 1)

	// drop the peer
	server.RemovePeer(peer.Self())
	_, err = helpers.PeerFromEvent(events, p2p.PeerEventTypeDrop)
	require.NoError(t, err)
	require.Len(t, cache.GetPeersRange(topic, 10), 0)
}

func TestPeerPoolWithTopicPoolWithLimits(t *testing.T) {
	var err error

	// start a peer
	var peers []*p2p.Server
	for i := 0; i < 2; i++ {
		peer, err := createServer()
		require.NoError(t, err)
		require.NoError(t, peer.Start())
		peers = append(peers, peer)
	}

	// start a server
	server, err := createServer()
	require.NoError(t, err)
	require.NoError(t, server.Start())

	// create TopicPool and PeerPool
	topic := discv5.Topic("test-topic")
	topicPool := NewTopicPoolWithLimits(NewTopicPoolBase(nil, topic), 1, 1)
	cache, err := newInMemoryCache()
	require.NoError(t, err)
	peerPool := NewPeerPool([]TopicPool{topicPool}, cache)
	peerPool.Start(server)

	// subscribe to peer events
	events := make(chan *p2p.PeerEvent, 20)
	sub := peerPool.SubscribePeerEvents(events)
	defer sub.Unsubscribe()

	// add peers to server
	for _, peer := range peers {
		peerPool.RequestToAddPeer(
			topic,
			discv5.NewNode(discv5.NodeID(peer.Self().ID), peer.Self().IP, peer.Self().TCP, peer.Self().UDP),
		)
	}

	// verify the peer was added and the PeerPool state
	_, err = helpers.PeerFromEvent(events, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	_, err = helpers.PeerFromEvent(events, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	// because the max limit is reached
	_, err = helpers.PeerFromEvent(events, p2p.PeerEventTypeDrop)
	require.NoError(t, err)

	// verify cache
	require.Len(t, cache.GetPeersRange(topic, 10), 2)
}

func TestPeerPoolWithTopicPoolEphemeral(t *testing.T) {
	var err error

	// start a peer
	var peers []*p2p.Server
	for i := 0; i < 2; i++ {
		peer, err := createServer()
		require.NoError(t, err)
		require.NoError(t, peer.Start())
		peers = append(peers, peer)
	}

	// start a server
	server, err := createServer()
	require.NoError(t, err)
	require.NoError(t, server.Start())

	// create TopicPool and PeerPool
	topic := discv5.Topic("test-topic")
	topicPool := NewTopicPoolEphemeral(
		NewTopicPoolWithLimits(NewTopicPoolBase(nil, topic), 1, 2),
	)
	cache, err := newInMemoryCache()
	require.NoError(t, err)
	peerPool := NewPeerPool([]TopicPool{topicPool}, cache)
	peerPool.Start(server)

	// subscribe to peer events
	events := make(chan *p2p.PeerEvent, 20)
	sub := peerPool.SubscribePeerEvents(events)
	defer sub.Unsubscribe()

	// Add peers to server. Peers are dropped immediately
	// after connection is established and verified.
	for _, peer := range peers {
		peerPool.RequestToAddPeer(
			topic,
			discv5.NewNode(discv5.NodeID(peer.Self().ID), peer.Self().IP, peer.Self().TCP, peer.Self().UDP),
		)

		_, err = helpers.PeerFromEvent(events, p2p.PeerEventTypeAdd)
		require.NoError(t, err)
		_, err = helpers.PeerFromEvent(events, p2p.PeerEventTypeDrop)
		require.NoError(t, err)
	}

	// verify cache
	require.Len(t, cache.GetPeersRange(topic, 10), 2)
}
