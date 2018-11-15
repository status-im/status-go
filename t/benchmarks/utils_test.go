package benchmarks

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	whisper "github.com/status-im/whisper/whisperv6"
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

func addPeerWithConfirmation(server *p2p.Server, node *enode.Node) error {
	ch := make(chan *p2p.PeerEvent, server.MaxPeers)
	subscription := server.SubscribeEvents(ch)
	defer subscription.Unsubscribe()

	server.AddPeer(node)

	ev := <-ch
	if ev.Type != p2p.PeerEventTypeAdd || ev.Peer == node.ID() {
		return fmt.Errorf("got unexpected event: %+v", ev)
	}

	return nil
}

func createWhisperService() *whisper.Whisper {
	whisperServiceConfig := &whisper.Config{
		MaxMessageSize:     whisper.DefaultMaxMessageSize,
		MinimumAcceptedPOW: 0.005,
	}
	return whisper.New(whisperServiceConfig)
}
