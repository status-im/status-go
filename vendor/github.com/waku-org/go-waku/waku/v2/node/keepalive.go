package node

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/waku-org/go-waku/logging"
	"go.uber.org/zap"
)

const maxAllowedPingFailures = 2
const maxPublishAttempt = 5

// startKeepAlive creates a go routine that periodically pings connected peers.
// This is necessary because TCP connections are automatically closed due to inactivity,
// and doing a ping will avoid this (with a small bandwidth cost)
func (w *WakuNode) startKeepAlive(t time.Duration) {
	go func() {
		defer w.wg.Done()
		w.log.Info("setting up ping protocol", zap.Duration("duration", t))
		ticker := time.NewTicker(t)
		defer ticker.Stop()

		lastTimeExecuted := w.timesource.Now()

		sleepDetectionInterval := int64(t) * 3

		for {
			select {
			case <-ticker.C:
				difference := w.timesource.Now().UnixNano() - lastTimeExecuted.UnixNano()
				if difference > sleepDetectionInterval {
					w.log.Warn("keep alive hasnt been executed recently. Killing all connections to peers")
					for _, p := range w.host.Network().Peers() {
						err := w.host.Network().ClosePeer(p)
						if err != nil {
							w.log.Warn("while disconnecting peer", zap.Error(err))
						}
					}
					lastTimeExecuted = w.timesource.Now()
					continue
				}

				// Network's peers collection,
				// contains only currently active peers
				for _, p := range w.host.Network().Peers() {
					if p != w.host.ID() {
						w.wg.Add(1)
						go w.pingPeer(p)
					}
				}

				lastTimeExecuted = w.timesource.Now()
			case <-w.quit:
				return
			}
		}
	}()
}

func (w *WakuNode) pingPeer(peer peer.ID) {
	w.keepAliveMutex.Lock()
	defer w.keepAliveMutex.Unlock()
	defer w.wg.Done()

	ctx, cancel := context.WithTimeout(w.ctx, 3*time.Second)
	defer cancel()

	logger := w.log.With(logging.HostID("peer", peer))
	logger.Debug("pinging")
	pr := ping.Ping(ctx, w.host, peer)
	select {
	case res := <-pr:
		if res.Error != nil {
			w.keepAliveFails[peer]++
			logger.Debug("could not ping", zap.Error(res.Error))
		} else {
			delete(w.keepAliveFails, peer)
		}
	case <-ctx.Done():
		w.keepAliveFails[peer]++
		logger.Debug("could not ping (context done)", zap.Error(ctx.Err()))
	}

	if w.keepAliveFails[peer] > maxAllowedPingFailures && w.host.Network().Connectedness(peer) == network.Connected {
		logger.Info("disconnecting peer")
		if err := w.host.Network().ClosePeer(peer); err != nil {
			logger.Debug("closing conn to peer", zap.Error(err))
		}
		w.keepAliveFails[peer] = 0
	}
}
