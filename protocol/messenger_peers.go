package protocol

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/status-im/status-go/eth-node/types"
)

func (m *Messenger) AddStorePeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return m.transport.AddStorePeer(address)
}

func (m *Messenger) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return m.transport.AddRelayPeer(address)
}

func (m *Messenger) DialPeer(address multiaddr.Multiaddr) error {
	return m.transport.DialPeer(address)
}

func (m *Messenger) DialPeerByID(peerID peer.ID) error {
	return m.transport.DialPeerByID(peerID)
}

func (m *Messenger) DropPeer(peerID peer.ID) error {
	return m.transport.DropPeer(peerID)
}

func (m *Messenger) Peers() types.PeerStats {
	return m.transport.Peers()
}

func (m *Messenger) RelayPeersByTopic(topic string) (*types.PeerList, error) {
	return m.transport.RelayPeersByTopic(topic)
}

func (m *Messenger) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return m.transport.ListenAddresses()
}

func (m *Messenger) ENR() (*enode.Node, error) {
	return m.transport.ENR()
}

// Subscribe to a pubsub topic, passing an optional public key if the pubsub topic is protected
func (m *Messenger) SubscribeToPubsubTopic(topic string, optPublicKey *ecdsa.PublicKey) error {
	return m.transport.SubscribeToPubsubTopic(topic, optPublicKey)
}

func (m *Messenger) StorePubsubTopicKey(topic string, privKey *ecdsa.PrivateKey) error {
	return m.transport.StorePubsubTopicKey(topic, privKey)
}
