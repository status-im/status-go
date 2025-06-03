//go:build use_nwaku
// +build use_nwaku

package wakuv2

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/waku-org/waku-go-bindings/waku"

	commonapi "github.com/waku-org/go-waku/waku/v2/api/common"
)

type pinger struct {
	node *waku.WakuNode
}

func newPinger(node *waku.WakuNode) commonapi.Pinger {
	return &pinger{
		node: node,
	}
}

func (p *pinger) PingPeer(ctx context.Context, peerInfo peer.AddrInfo) (time.Duration, error) {
	return p.node.PingPeer(ctx, peerInfo)
}
