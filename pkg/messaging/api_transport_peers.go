package messaging

import (
	"github.com/libp2p/go-libp2p/core/peer"

	adapters "github.com/status-im/status-go/pkg/messaging/adapters"
	types "github.com/status-im/status-go/pkg/messaging/types"
)

// The helpers below leak transport implementation details and are retained ONLY
// for the Python functional tests (tests-functional): Peers backs the
// wait_for_online / test_discovery checks and PeerID backs node identification.
// They are not used by status-app — clients needing peer information should use
// Prometheus metrics or the Waku node's RPC interface instead.

func (a *API) Peers() types.PeerStats {
	return adapters.FromWakuPeerStats(a.core.stack.Transport.Peers())
}

func (a *API) PeerID() peer.ID {
	return a.core.waku.PeerID()
}
