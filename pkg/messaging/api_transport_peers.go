package messaging

import (
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	adapters2 "github.com/status-im/status-go/pkg/messaging/adapters"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
)

// These utilities leak transport implementation details and should be eventually removed.
// Client applications requiring peer information or statistics should utilize Prometheus
// metrics for observability or communicate directly with the Waku node's RPC interface.

// Deprecated: shouldn't be exposed
func (a *API) GetStats() types2.TransportStats {
	return adapters2.FromWakuTransportStats(a.core.stack.Transport.GetStats())
}

// Deprecated: shouldn't be exposed
func (a *API) PeerCount() int {
	return a.core.stack.Transport.PeerCount()
}

// Deprecated: shouldn't be exposed
func (a *API) Peers() types2.PeerStats {
	return adapters2.FromWakuPeerStats(a.core.stack.Transport.Peers())
}

// Deprecated: shouldn't be exposed
func (a *API) PeerID() peer.ID {
	return a.core.waku.PeerID()
}

// Deprecated: shouldn't be exposed
func (a *API) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return a.core.stack.Transport.ListenAddresses()
}

// Deprecated: shouldn't be exposed
func (a *API) ENR() (*enode.Node, error) {
	return a.core.stack.Transport.ENR()
}

// Deprecated: shouldn't be exposed
func (a *API) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return a.core.stack.Transport.AddRelayPeer(address)
}

// Deprecated: shouldn't be exposed
func (a *API) DialPeer(address multiaddr.Multiaddr) error {
	return a.core.stack.Transport.DialPeer(address)
}

// Deprecated: shouldn't be exposed
func (a *API) DialPeerByID(peerID peer.ID) error {
	return a.core.stack.Transport.DialPeerByID(peerID)
}

// Deprecated: shouldn't be exposed
func (a *API) DropPeer(peerID peer.ID) error {
	return a.core.stack.Transport.DropPeer(peerID)
}
