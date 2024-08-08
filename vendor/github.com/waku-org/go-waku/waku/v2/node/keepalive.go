package node

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/waku-org/go-waku/logging"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

const maxAllowedPingFailures = 2

// If the difference between the last time the keep alive code was executed and now is greater
// than sleepDectectionIntervalFactor * keepAlivePeriod, force the ping verification to disconnect
// the peers if they don't reply back
const sleepDetectionIntervalFactor = 3

const maxPeersToPingPerProtocol = 10

const maxAllowedSubsequentPingFailures = 2

func disconnectAllPeers(host host.Host, logger *zap.Logger) {
	for _, p := range host.Network().Peers() {
		err := host.Network().ClosePeer(p)
		if err != nil {
			logger.Debug("closing conn to peer", zap.Error(err))
		}
	}
}

// startKeepAlive creates a go routine that periodically pings connected peers.
// This is necessary because TCP connections are automatically closed due to inactivity,
// and doing a ping will avoid this (with a small bandwidth cost)
func (w *WakuNode) startKeepAlive(ctx context.Context, randomPeersPingDuration time.Duration, allPeersPingDuration time.Duration) {
	defer w.wg.Done()

	if !w.opts.enableRelay {
		return
	}

	w.log.Info("setting up ping protocol", zap.Duration("randomPeersPingDuration", randomPeersPingDuration), zap.Duration("allPeersPingDuration", allPeersPingDuration))

	randomPeersTickerC := make(<-chan time.Time)
	if randomPeersPingDuration != 0 {
		randomPeersTicker := time.NewTicker(randomPeersPingDuration)
		defer randomPeersTicker.Stop()
		randomPeersTickerC = randomPeersTicker.C
	}

	allPeersTickerC := make(<-chan time.Time)
	if allPeersPingDuration != 0 {
		allPeersTicker := time.NewTicker(allPeersPingDuration)
		defer allPeersTicker.Stop()
		randomPeersTickerC = allPeersTicker.C
	}

	lastTimeExecuted := w.timesource.Now()

	sleepDetectionInterval := int64(randomPeersPingDuration) * sleepDetectionIntervalFactor

	var iterationFailure int
	for {
		peersToPing := []peer.ID{}

		select {
		case <-allPeersTickerC:
			if w.opts.enableRelay {
				relayPeersSet := make(map[peer.ID]struct{})
				for _, t := range w.Relay().Topics() {
					for _, p := range w.Relay().PubSub().ListPeers(t) {
						relayPeersSet[p] = struct{}{}
					}
				}
				peersToPing = append(peersToPing, maps.Keys(relayPeersSet)...)
			}

		case <-randomPeersTickerC:
			difference := w.timesource.Now().UnixNano() - lastTimeExecuted.UnixNano()
			if difference > sleepDetectionInterval {
				lastTimeExecuted = w.timesource.Now()
				w.log.Warn("keep alive hasnt been executed recently. Killing all connections")
				disconnectAllPeers(w.host, w.log)
				continue
			} else if iterationFailure >= maxAllowedSubsequentPingFailures {
				iterationFailure = 0
				w.log.Warn("Pinging random peers failed, node is likely disconnected. Killing all connections")
				disconnectAllPeers(w.host, w.log)
				continue
			}

			if w.opts.enableRelay {
				// Priorize mesh peers
				meshPeersSet := make(map[peer.ID]struct{})
				for _, t := range w.Relay().Topics() {
					for _, p := range w.Relay().PubSub().MeshPeers(t) {
						meshPeersSet[p] = struct{}{}
					}
				}
				peersToPing = append(peersToPing, maps.Keys(meshPeersSet)...)

				// Ping also some random relay peers
				if maxPeersToPingPerProtocol-len(peersToPing) > 0 {
					relayPeersSet := make(map[peer.ID]struct{})
					for _, t := range w.Relay().Topics() {
						for _, p := range w.Relay().PubSub().ListPeers(t) {
							if _, ok := meshPeersSet[p]; !ok {
								relayPeersSet[p] = struct{}{}
							}
						}
					}

					relayPeers := maps.Keys(relayPeersSet)
					rand.Shuffle(len(relayPeers), func(i, j int) { relayPeers[i], relayPeers[j] = relayPeers[j], relayPeers[i] })

					peerLen := maxPeersToPingPerProtocol - len(peersToPing)
					if peerLen > len(relayPeers) {
						peerLen = len(relayPeers)
					}
					peersToPing = append(peersToPing, relayPeers[0:peerLen]...)
				}
			}

			if w.opts.enableFilterLightNode {
				// We also ping all filter nodes
				filterPeersSet := make(map[peer.ID]struct{})
				for _, s := range w.FilterLightnode().Subscriptions() {
					filterPeersSet[s.PeerID] = struct{}{}
				}
				peersToPing = append(peersToPing, maps.Keys(filterPeersSet)...)
			}
		case <-ctx.Done():
			w.log.Info("stopping ping protocol")
			return
		}

		pingWg := sync.WaitGroup{}
		pingWg.Add(len(peersToPing))
		pingResultChan := make(chan bool, len(peersToPing))
		for _, p := range peersToPing {
			go w.pingPeer(ctx, &pingWg, p, pingResultChan)
		}
		pingWg.Wait()
		close(pingResultChan)

		failureCounter := 0
		for couldPing := range pingResultChan {
			if !couldPing {
				failureCounter++
			}
		}

		if len(peersToPing) > 0 && failureCounter == len(peersToPing) {
			iterationFailure++
		} else {
			iterationFailure = 0
		}

		lastTimeExecuted = w.timesource.Now()
	}
}

func (w *WakuNode) pingPeer(ctx context.Context, wg *sync.WaitGroup, peerID peer.ID, resultChan chan bool) {
	defer wg.Done()

	logger := w.log.With(logging.HostID("peer", peerID))

	for i := 0; i < maxAllowedPingFailures; i++ {
		if w.host.Network().Connectedness(peerID) != network.Connected {
			// Peer is no longer connected. No need to ping
			resultChan <- false
			return
		}

		logger.Debug("pinging")

		if w.tryPing(ctx, peerID, logger) {
			resultChan <- true
			return
		}
	}

	if w.host.Network().Connectedness(peerID) != network.Connected {
		resultChan <- false
		return
	}

	logger.Info("disconnecting dead peer")
	if err := w.host.Network().ClosePeer(peerID); err != nil {
		logger.Debug("closing conn to peer", zap.Error(err))
	}

	resultChan <- false
}

func (w *WakuNode) tryPing(ctx context.Context, peerID peer.ID, logger *zap.Logger) bool {
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	pr := ping.Ping(ctx, w.host, peerID)
	select {
	case res := <-pr:
		if res.Error != nil {
			logger.Debug("could not ping", zap.Error(res.Error))
			return false
		}
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			logger.Debug("could not ping (context)", zap.Error(ctx.Err()))
		}
		return false
	}
	return true
}
