package protocol

import (
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/status-im/status-go/eth-node/types"
)

func (m *Messenger) AddStorePeer(address string) (peer.ID, error) {
	return m.transport.AddStorePeer(address)
}

func (m *Messenger) AddRelayPeer(address string) (peer.ID, error) {
	return m.transport.AddStorePeer(address)
}

func (m *Messenger) DialPeer(address string) error {
	return m.transport.DialPeer(address)
}

func (m *Messenger) DialPeerByID(peerID string) error {
	return m.transport.DialPeerByID(peerID)
}

func (m *Messenger) DropPeer(peerID string) error {
	return m.transport.DropPeer(peerID)
}

func (m *Messenger) Peers() map[string]types.WakuV2Peer {
	return m.transport.Peers()
}

func (m *Messenger) ListenAddresses() ([]string, error) {
	return m.transport.ListenAddresses()
}
