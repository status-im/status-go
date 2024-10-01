package peermanager

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/waku-org/go-waku/logging"
	"go.uber.org/zap"
)

type FastestPeerSelector struct {
	sync.RWMutex

	host host.Host

	logger *zap.Logger
}

func NewFastestPeerSelector(logger *zap.Logger) *FastestPeerSelector {
	return &FastestPeerSelector{
		logger: logger.Named("rtt-cache"),
	}
}

func (r *FastestPeerSelector) SetHost(h host.Host) {
	r.host = h
}

func (r *FastestPeerSelector) PingPeer(ctx context.Context, peer peer.ID) (time.Duration, error) {
	if peer == r.host.ID() {
		return 0, errors.New("can't ping yourself")
	}

	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()

	case result := <-ping.Ping(ctx, r.host, peer):
		r.Lock()
		defer r.Unlock()

		if result.Error == nil {
			return result.RTT, nil
		} else {
			r.logger.Debug("could not ping", logging.HostID("peer", peer), zap.Error(result.Error))
			return 0, result.Error
		}
	}

}

func (r *FastestPeerSelector) FastestPeer(ctx context.Context, peers peer.IDSlice) (peer.ID, error) {
	var peerRTT []pingResult
	var peerRTTMutex sync.Mutex

	wg := sync.WaitGroup{}
	pingCh := make(chan peer.ID)

	pinged := make(map[peer.ID]struct{})

	go func() {
		// Ping any peer with no latency recorded
		for peerToPing := range pingCh {
			go func(p peer.ID) {
				defer wg.Done()
				rtt := time.Hour
				result, err := r.PingPeer(ctx, p)
				if err == nil {
					rtt = result
				}

				peerRTTMutex.Lock()
				peerRTT = append(peerRTT, pingResult{
					peerID:        p,
					rtt:           rtt,
					connectedness: r.host.Network().Connectedness(p),
				})
				peerRTTMutex.Unlock()
			}(peerToPing)
		}
	}()

	for _, p := range peers {
		latency := r.host.Peerstore().LatencyEWMA(p)
		if latency == 0 {
			wg.Add(1)
			pinged[p] = struct{}{} // To avoid double pings
			pingCh <- p
		} else {
			peerRTTMutex.Lock()
			peerRTT = append(peerRTT, pingResult{
				peerID:        p,
				rtt:           latency,
				connectedness: r.host.Network().Connectedness(p),
			})
			peerRTTMutex.Unlock()
		}
	}

	// Wait for pings to be done (if any)
	wg.Wait()
	close(pingCh)

	sort.Sort(pingSort(peerRTT))

	for _, p := range peerRTT {
		if p.rtt == time.Hour {
			break
		}

		// Make sure peer is reachable
		_, exists := pinged[p.peerID] // Did we just ping the peer?
		if !exists {
			_, err := r.PingPeer(ctx, p.peerID)
			if err != nil {
				continue
			} else {
				if p.rtt != time.Hour {
					return p.peerID, nil
				}
			}
		} else {
			if p.rtt != time.Hour {
				return p.peerID, nil
			}
		}
	}

	return "", ErrNoPeersAvailable
}

type pingResult struct {
	peerID        peer.ID
	rtt           time.Duration
	connectedness network.Connectedness
}

type pingSort []pingResult

func (a pingSort) Len() int {
	return len(a)
}

func (a pingSort) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

var connectednessPriority map[network.Connectedness]int

func init() {
	// Closer to 0 is prefered
	connectednessPriority = map[network.Connectedness]int{
		network.Connected:    1,
		network.NotConnected: 2,
		network.Limited:      3,
	}
}

func (a pingSort) Less(i, j int) bool {
	return connectednessPriority[a[i].connectedness] < connectednessPriority[a[j].connectedness] && a[i].rtt < a[j].rtt
}
