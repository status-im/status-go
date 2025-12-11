package protocol

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/p2p/enode"

	messagingtypes "github.com/status-im/status-go/messaging/types"
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

func (m *Messenger) Peers() messagingtypes.PeerStats {
	return m.messaging.Peers()
}

func (m *Messenger) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return m.messaging.ListenAddresses()
}

func (m *Messenger) ENR() (*enode.Node, error) {
	return m.messaging.ENR()
}
