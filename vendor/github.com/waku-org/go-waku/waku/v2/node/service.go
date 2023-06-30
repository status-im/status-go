package node

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

type Service interface {
	SetHost(h host.Host)
	Start(context.Context) error
	Stop()
}

type ReceptorService interface {
	SetHost(h host.Host)
	Stop()
	Start(context.Context, relay.Subscription) error
}

type PeerConnectorService interface {
	Service
	PeerChannel() chan<- v2.PeerData
}
