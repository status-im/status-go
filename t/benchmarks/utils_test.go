package benchmarks

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

var (
	topic = whisper.TopicType{0xfa, 0xfb, 0xfc, 0xfd}
)

func createNode() (*node.Node, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return node.New(&node.Config{
		DataDir: "",
		P2P: p2p.Config{
			PrivateKey:  key,
			DiscoveryV5: false,
			NoDiscovery: true,
			MaxPeers:    1,
			NAT:         nat.Any(),
		},
	})
}

func addPeerWithConfirmation(server *p2p.Server, node *discover.Node) error {
	ch := make(chan *p2p.PeerEvent, server.MaxPeers)
	subscription := server.SubscribeEvents(ch)
	defer subscription.Unsubscribe()

	server.AddPeer(node)

	ev := <-ch
	if ev.Type != p2p.PeerEventTypeAdd || ev.Peer != node.ID {
		return fmt.Errorf("got unexpected event: %+v", ev)
	}

	return nil
}

func createWhisperService() *whisper.Whisper {
	whisperServiceConfig := &whisper.Config{
		MaxMessageSize:     whisper.DefaultMaxMessageSize,
		MinimumAcceptedPOW: 0.005,
		TimeSource:         func() time.Time { return time.Now().UTC() },
	}
	return whisper.New(whisperServiceConfig)
}
