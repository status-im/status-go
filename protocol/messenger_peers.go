package protocol

import (
	"crypto/ecdsa"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/p2p/enode"

	wakutypes "github.com/status-im/status-go/waku/types"
)

func (m *Messenger) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return m.messaging.AddRelayPeer(address)
}

func (m *Messenger) DialPeer(address multiaddr.Multiaddr) error {
	return m.messaging.DialPeer(address)
}

func (m *Messenger) DialPeerByID(peerID peer.ID) error {
	return m.messaging.DialPeerByID(peerID)
}

func (m *Messenger) DropPeer(peerID peer.ID) error {
	return m.messaging.DropPeer(peerID)
}

func (m *Messenger) Peers() wakutypes.PeerStats {
	return m.messaging.Peers()
}

func (m *Messenger) RelayPeersByTopic(topic string) (*wakutypes.PeerList, error) {
	return m.messaging.RelayPeersByTopic(topic)
}

func (m *Messenger) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return m.messaging.ListenAddresses()
}

func (m *Messenger) ENR() (*enode.Node, error) {
	return m.messaging.ENR()
}

// Subscribe to a pubsub topic, passing an optional public key if the pubsub topic is protected
func (m *Messenger) SubscribeToPubsubTopic(topic string, optPublicKey *ecdsa.PublicKey) error {
	return m.messaging.SubscribeToPubsubTopic(topic, optPublicKey)
}

func (m *Messenger) StorePubsubTopicKey(topic string, privKey *ecdsa.PrivateKey) error {
	return m.messaging.StorePubsubTopicKey(topic, privKey)
}
