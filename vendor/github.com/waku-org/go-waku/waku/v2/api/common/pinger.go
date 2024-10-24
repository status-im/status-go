package common

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

type Pinger interface {
	PingPeer(ctx context.Context, peerID peer.ID) (time.Duration, error)
}

type defaultPingImpl struct {
	host host.Host
}

func NewDefaultPinger(host host.Host) Pinger {
	return &defaultPingImpl{
		host: host,
	}
}

func (d *defaultPingImpl) PingPeer(ctx context.Context, peerID peer.ID) (time.Duration, error) {
	pingResultCh := ping.Ping(ctx, d.host, peerID)
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case r := <-pingResultCh:
		if r.Error != nil {
			return 0, r.Error
		}
		return r.RTT, nil
	}
}
