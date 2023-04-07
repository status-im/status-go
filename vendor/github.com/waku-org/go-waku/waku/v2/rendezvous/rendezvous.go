package rendezvous

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	rvs "github.com/berty/go-libp2p-rendezvous"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

const RendezvousID = rvs.RendezvousProto

type rendezvousPoint struct {
	sync.RWMutex

	id     peer.ID
	cookie []byte
}

type Rendezvous struct {
	host host.Host

	enableServer  bool
	db            *DB
	rendezvousSvc *rvs.RendezvousService

	discoverPeers    bool
	rendezvousPoints []*rendezvousPoint
	peerConnector    PeerConnector

	log    *zap.Logger
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

type PeerConnector interface {
	PeerChannel() chan<- peer.AddrInfo
}

func NewRendezvous(host host.Host, enableServer bool, db *DB, discoverPeers bool, rendezvousPoints []peer.ID, peerConnector PeerConnector, log *zap.Logger) *Rendezvous {
	logger := log.Named("rendezvous")

	var rendevousPoints []*rendezvousPoint
	for _, rp := range rendezvousPoints {
		rendevousPoints = append(rendevousPoints, &rendezvousPoint{
			id: rp,
		})
	}

	return &Rendezvous{
		host:             host,
		enableServer:     enableServer,
		db:               db,
		discoverPeers:    discoverPeers,
		rendezvousPoints: rendevousPoints,
		peerConnector:    peerConnector,
		log:              logger,
	}
}

func (r *Rendezvous) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	if r.enableServer {
		err := r.db.Start(ctx)
		if err != nil {
			cancel()
			return err
		}

		r.rendezvousSvc = rvs.NewRendezvousService(r.host, r.db)
	}

	r.wg.Add(1)
	go r.register(ctx)

	if r.discoverPeers {
		r.wg.Add(1)
		go r.discover(ctx)
	}

	r.log.Info("rendezvous protocol started")
	return nil
}

const registerBackoff = 200 * time.Millisecond
const registerMaxRetries = 7

func (r *Rendezvous) getRandomServer() *rendezvousPoint {
	return r.rendezvousPoints[rand.Intn(len(r.rendezvousPoints))] // nolint: gosec
}

func (r *Rendezvous) discover(ctx context.Context) {
	defer r.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			server := r.getRandomServer()

			rendezvousClient := rvs.NewRendezvousClient(r.host, server.id)

			addrInfo, cookie, err := rendezvousClient.Discover(ctx, relay.DefaultWakuTopic, 5, server.cookie)
			if err != nil {
				r.log.Error("could not discover new peers", zap.Error(err))
				cookie = nil
			}

			if len(addrInfo) != 0 {
				server.Lock()
				server.cookie = cookie
				server.Unlock()

				for _, addr := range addrInfo {
					r.peerConnector.PeerChannel() <- addr
				}
			} else {
				// TODO: change log level to DEBUG in go-libp2p-rendezvous@v0.4.1/svc.go:234  discover query
				// TODO: improve this by adding an exponential backoff?
				time.Sleep(2 * time.Second)
			}
		}
	}
}

func (r *Rendezvous) callRegister(ctx context.Context, rendezvousClient rvs.RendezvousClient, retries int) (<-chan time.Time, int) {
	ttl, err := rendezvousClient.Register(ctx, relay.DefaultWakuTopic, rvs.DefaultTTL) // TODO: determine which topic to use
	var t <-chan time.Time
	if err != nil {
		r.log.Error("registering rendezvous client", zap.Error(err))
		backoff := registerBackoff * time.Duration(math.Exp2(float64(retries)))
		t = time.After(backoff)
		retries++
	} else {
		t = time.After(ttl)
	}

	return t, retries
}

func (r *Rendezvous) register(ctx context.Context) {
	defer r.wg.Done()

	for _, m := range r.rendezvousPoints {
		r.wg.Add(1)
		go func(m *rendezvousPoint) {
			r.wg.Done()

			rendezvousClient := rvs.NewRendezvousClient(r.host, m.id)
			retries := 0
			var t <-chan time.Time

			t, retries = r.callRegister(ctx, rendezvousClient, retries)
			for {
				select {
				case <-ctx.Done():
					return
				case <-t:
					t, retries = r.callRegister(ctx, rendezvousClient, retries)
					if retries >= registerMaxRetries {
						return
					}
				}
			}
		}(m)
	}
}

func (r *Rendezvous) Stop() {
	r.cancel()
	r.wg.Wait()
	r.host.RemoveStreamHandler(rvs.RendezvousProto)
	r.rendezvousSvc = nil
}
