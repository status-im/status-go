package messaging

import (
	"github.com/libp2p/go-libp2p/core/peer"

	adapters "github.com/status-im/status-go/pkg/messaging/adapters"
	types "github.com/status-im/status-go/pkg/messaging/types"
)

func (a *API) Peers() types.PeerStats {
	return adapters.FromWakuPeerStats(a.core.stack.Transport.Peers())
}

func (a *API) PeerID() peer.ID {
	return a.core.waku.PeerID()
}
