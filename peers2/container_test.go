package peers2

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/discovery"
	"github.com/status-im/status-go/peers"
	"github.com/status-im/status-go/t/helpers"
)

func TestDiscoveryContainer(t *testing.T) {
	bootnodeKey, _ := crypto.GenerateKey()
	bootnodePort := 62971
	bootnode := &p2p.Server{
		Config: p2p.Config{
			MaxPeers:    10,
			ListenAddr:  fmt.Sprintf("0.0.0.0:%d", bootnodePort),
			PrivateKey:  bootnodeKey,
			DiscoveryV5: true,
			NoDiscovery: true,
		},
	}
	require.NoError(t, bootnode.Start())
	bootnodeV5 := discv5.NewNode(
		bootnode.DiscV5.Self().ID,
		net.ParseIP("127.0.0.1"),
		bootnode.Self().TCP,
		bootnode.Self().TCP,
	)

	server, err := createServer()
	require.NoError(t, err)
	require.NoError(t, server.Start())

	// subscribe to server events
	events := make(chan *p2p.PeerEvent, 20)
	sub := server.SubscribeEvents(events)
	defer sub.Unsubscribe()

	// prepare container elements
	disc := discovery.NewDiscV5(server.PrivateKey, server.ListenAddr, []*discv5.Node{bootnodeV5})
	topic := discv5.Topic("test-topic")
	topicPool := NewTopicPoolWithLimits(
		NewTopicPoolBase(disc, topic, SetPeersHandler(&SkipSelfPeersHandler{server.Self().ID})),
		1, 1,
	)
	container := NewDiscoveryContainer(disc, []TopicPool{topicPool}, nil)
	require.NoError(t, container.Start(server, 0))
	defer func() { require.NoError(t, container.Stop()) }()

	// register topic
	registerServer, err := createServer()
	require.NoError(t, err)
	require.NoError(t, registerServer.Start())
	registerDisc := discovery.NewDiscV5(registerServer.PrivateKey, registerServer.ListenAddr, []*discv5.Node{bootnodeV5})
	require.NoError(t, registerDisc.Start())
	register := peers.NewRegister(registerDisc, topic)
	require.NoError(t, register.Start())

	// check peers
	peerID, err := helpers.PeerFromEvent(events, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	require.Equal(t, registerServer.Self().ID[:], peerID[:])

	time.Sleep(time.Second * 2)
	require.False(t, container.discoveryRunning)
}
