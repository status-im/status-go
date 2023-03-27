package rendezvous

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
)

type RendezvousSync interface {
	Register(p peer.ID, ns string, addrs [][]byte, ttl int, counter uint64)
	Unregister(p peer.ID, ns string)
}

type RendezvousSyncSubscribable interface {
	Subscribe(ns string) (syncDetails string, err error)
	GetServiceType() string
}

type RendezvousSyncClient interface {
	Subscribe(ctx context.Context, syncDetails string) (<-chan *Registration, error)
	GetServiceType() string
}
