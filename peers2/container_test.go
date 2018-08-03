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
	defer bootnode.Stop()
	bootnodeV5 := discv5.NewNode(
		bootnode.DiscV5.Self().ID,
		net.ParseIP("127.0.0.1"),
		bootnode.Self().TCP,
		bootnode.Self().TCP,
	)

	server, err := createServer()
	require.NoError(t, err)
	require.NoError(t, server.Start())
	defer server.Stop()

	// subscribe to server events
	events := make(chan *p2p.PeerEvent, 20)
	sub := server.SubscribeEvents(events)
	defer sub.Unsubscribe()

	// prepare container elements
	disc := discovery.NewDiscV5(server.PrivateKey, server.ListenAddr, []*discv5.Node{bootnodeV5})
	topic := discv5.Topic("test-topic")
	topicPool := NewTopicPoolWithLimits(
		NewTopicPoolBase(
			disc,
			topic,
			SetPeersHandler(&SkipSelfPeers{server.Self().ID}),
			SetPeriod(newFastSlowDiscoverPeriod(time.Millisecond*100, time.Second, time.Second*5)),
		),
		1, 1,
	)
	container := NewDiscoveryContainer(disc, []TopicPool{topicPool}, nil)
	require.NoError(t, container.Start(server, 0))
	defer func() { require.NoError(t, container.Stop()) }()

	// register topic
	registerServer, err := createServer()
	require.NoError(t, err)
	require.NoError(t, registerServer.Start())
	defer registerServer.Stop()
	registerDisc := discovery.NewDiscV5(registerServer.PrivateKey, registerServer.ListenAddr, []*discv5.Node{bootnodeV5})
	require.NoError(t, registerDisc.Start())
	register := peers.NewRegister(registerDisc, topic)
	require.NoError(t, register.Start())

	// check peers
	peerID, err := helpers.PeerFromEvent(events, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	require.Equal(t, registerServer.Self().ID[:], peerID[:])
}

func TestContainerDiscoveryTimeout(t *testing.T) {
	server, err := createServer()
	require.NoError(t, err)
	require.NoError(t, server.Start())
	defer server.Stop()

	timeout := time.Millisecond * 50
	disc := &discoveryMock{}
	container := NewDiscoveryContainer(disc, nil, nil)
	require.NoError(t, container.Start(server, timeout))
	defer func() { require.NoError(t, container.Stop()) }()

	require.True(t, disc.Running())
	time.Sleep(timeout * 2)
	require.False(t, disc.Running())
}
