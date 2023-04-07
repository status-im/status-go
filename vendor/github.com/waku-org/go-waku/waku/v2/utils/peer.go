package utils

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// ErrNoPeersAvailable is emitted when no suitable peers are found for
// some protocol
var ErrNoPeersAvailable = errors.New("no suitable peers found")

func GetPeerID(m multiaddr.Multiaddr) (peer.ID, error) {
	peerIDStr, err := m.ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return "", err
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return "", err
	}

	return peerID, nil
}

// SelectPeer is used to return a random peer that supports a given protocol.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the node peerstore
func SelectPeer(host host.Host, protocolId protocol.ID, specificPeers []peer.ID, log *zap.Logger) (peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?
	//  - default store peer?

	peerSet := specificPeers
	if len(peerSet) == 0 {
		peerSet = host.Peerstore().Peers()
	}

	var peers peer.IDSlice
	for _, peer := range peerSet {
		protocols, err := host.Peerstore().SupportsProtocols(peer, protocolId)
		if err != nil {
			return "", err
		}

		if len(protocols) > 0 {
			peers = append(peers, peer)
		}
	}

	if len(peers) >= 1 {
		// TODO: proper heuristic here that compares peer scores and selects "best" one. For now a random peer for the given protocol is returned
		return peers[rand.Intn(len(peers))], nil // nolint: gosec
	}

	return "", ErrNoPeersAvailable
}

type pingResult struct {
	p   peer.ID
	rtt time.Duration
}

// SelectPeerWithLowestRTT will select a peer that supports a specific protocol with the lowest reply time
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the node peerstore
func SelectPeerWithLowestRTT(ctx context.Context, host host.Host, protocolId protocol.ID, specificPeers []peer.ID, log *zap.Logger) (peer.ID, error) {
	var peers peer.IDSlice

	peerSet := specificPeers
	if len(peerSet) == 0 {
		peerSet = host.Peerstore().Peers()
	}

	for _, peer := range peerSet {
		protocols, err := host.Peerstore().SupportsProtocols(peer, protocolId)
		if err != nil {
			return "", err
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
			return "", ErrNoPeersAvailable
		}

		return min.p, nil
	case <-ctx.Done():
		return "", ErrNoPeersAvailable
	}
}
