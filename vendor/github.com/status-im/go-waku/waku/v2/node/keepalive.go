package node

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

const maxAllowedPingFailures = 2
const maxPublishAttempt = 5

// startKeepAlive creates a go routine that periodically pings connected peers.
// This is necessary because TCP connections are automatically closed due to inactivity,
// and doing a ping will avoid this (with a small bandwidth cost)
func (w *WakuNode) startKeepAlive(t time.Duration) {
	go func() {
		defer w.wg.Done()
		w.log.Info("Setting up ping protocol with duration of ", t)
		ticker := time.NewTicker(t)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Compared to Network's peers collection,
				// Peerstore contains all peers ever connected to,
				// thus if a host goes down and back again,
				// pinging a peer will trigger identification process,
				// which is not possible when iterating
				// through Network's peer collection, as it will be empty
				for _, p := range w.host.Peerstore().Peers() {
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

	w.log.Debug("Pinging ", peer)
	pr := ping.Ping(ctx, w.host, peer)
	select {
	case res := <-pr:
		if res.Error != nil {
			w.keepAliveFails[peer]++
			w.log.Debug(fmt.Sprintf("Could not ping %s: %s", peer, res.Error.Error()))
		} else {
			w.keepAliveFails[peer] = 0
		}
	case <-ctx.Done():
		w.keepAliveFails[peer]++
		w.log.Debug(fmt.Sprintf("Could not ping %s: %s", peer, ctx.Err()))
	}

	if w.keepAliveFails[peer] > maxAllowedPingFailures && w.host.Network().Connectedness(peer) == network.Connected {
		w.log.Info("Disconnecting peer ", peer)
		if err := w.host.Network().ClosePeer(peer); err != nil {
			w.log.Debug(fmt.Sprintf("Could not close conn to peer %s: %s", peer, err))
		}
		w.keepAliveFails[peer] = 0
	}
}
