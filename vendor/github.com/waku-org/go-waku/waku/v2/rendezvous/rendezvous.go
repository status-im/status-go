package rendezvous

import (
	"context"

	rvs "github.com/berty/go-libp2p-rendezvous"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

const RendezvousID = rvs.RendezvousProto

type Rendezvous struct {
	host          host.Host
	peerConnector PeerConnector
	db            *DB
	rendezvousSvc *rvs.RendezvousService

	log *zap.Logger
}

type PeerConnector interface {
	PeerChannel() chan<- peer.AddrInfo
}

func NewRendezvous(host host.Host, db *DB, peerConnector PeerConnector, log *zap.Logger) *Rendezvous {
	logger := log.Named("rendezvous")

	return &Rendezvous{
		host:          host,
		db:            db,
		peerConnector: peerConnector,
		log:           logger,
	}
}

func (r *Rendezvous) Start(ctx context.Context) error {
	err := r.db.Start(ctx)
	if err != nil {
		return err
	}

	r.rendezvousSvc = rvs.NewRendezvousService(r.host, r.db)
	r.log.Info("rendezvous protocol started")
	return nil
}

func (r *Rendezvous) Stop() {
	r.host.RemoveStreamHandler(rvs.RendezvousProto)
	r.rendezvousSvc = nil
}
