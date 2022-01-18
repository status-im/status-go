package utils

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"go.uber.org/zap"
)

var ErrNoPeersAvailable = errors.New("no suitable peers found")
var PingServiceNotAvailable = errors.New("ping service not available")

// SelectPeer is used to return a random peer that supports a given protocol.
func SelectPeer(host host.Host, protocolId string, log *zap.SugaredLogger) (*peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?
	//  - default store peer?
	var peers peer.IDSlice
	for _, peer := range host.Peerstore().Peers() {
		protocols, err := host.Peerstore().SupportsProtocols(peer, protocolId)
		if err != nil {
			log.Error("error obtaining the protocols supported by peers", err)
			return nil, err
		}

		if len(protocols) > 0 {
			peers = append(peers, peer)
		}
	}

	if len(peers) >= 1 {
		// TODO: proper heuristic here that compares peer scores and selects "best" one. For now a random peer for the given protocol is returned
		return &peers[rand.Intn(len(peers))], nil // nolint: gosec
	}

	return nil, ErrNoPeersAvailable
}

type pingResult struct {
	p   peer.ID
	rtt time.Duration
}

func SelectPeerWithLowestRTT(ctx context.Context, host host.Host, protocolId string, log *zap.SugaredLogger) (*peer.ID, error) {
	var peers peer.IDSlice
	for _, peer := range host.Peerstore().Peers() {
		protocols, err := host.Peerstore().SupportsProtocols(peer, protocolId)
		if err != nil {
			log.Error("error obtaining the protocols supported by peers", err)
			return nil, err
		}

		if len(protocols) > 0 {
			peers = append(peers, peer)
		}
	}

	wg := sync.WaitGroup{}
	waitCh := make(chan struct{})
	pingCh := make(chan pingResult, 1000)

	wg.Add(len(peers))

	go func() {
		for _, p := range peers {
			go func(p peer.ID) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				result := <-ping.Ping(ctx, host, p)
				if result.Error == nil {
					pingCh <- pingResult{
						p:   p,
						rtt: result.RTT,
					}
				}
			}(p)
		}
		wg.Wait()
		close(waitCh)
		close(pingCh)
	}()

	select {
	case <-waitCh:
		var min *pingResult
		for p := range pingCh {
			if min == nil {
				min = &p
			} else {
				if p.rtt < min.rtt {
					min = &p
				}
			}
		}
		if min == nil {
			return nil, ErrNoPeersAvailable
		} else {
			return &min.p, nil
		}
	case <-ctx.Done():
		return nil, ErrNoPeersAvailable
	}
}
