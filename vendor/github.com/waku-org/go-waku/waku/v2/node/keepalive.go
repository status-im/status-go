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
		for {
			select {
			case <-ticker.C:
				// Network's peers collection,
				// contains only currently active peers
				for _, p := range w.host.Network().Peers() {
					if p != w.host.ID() {
						w.wg.Add(1)
						go w.pingPeer(p)
					}
				}
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
			w.keepAliveFails[peer] = 0
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
