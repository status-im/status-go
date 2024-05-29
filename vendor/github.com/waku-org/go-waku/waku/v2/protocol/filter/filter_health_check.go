package filter

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

func (wf *WakuFilterLightNode) PingPeers() {
	//Send a ping to all the peers and report their status to corresponding subscriptions
	// Alive or not or set state of subcription??
	for _, peer := range wf.subscriptions.GetSubscribedPeers() {
		go wf.PingPeer(peer)
	}
}

func (wf *WakuFilterLightNode) PingPeer(peer peer.ID) {
	ctxWithTimeout, cancel := context.WithTimeout(wf.CommonService.Context(), wf.peerPingInterval)
	defer cancel()
	err := wf.Ping(ctxWithTimeout, peer)
	if err != nil {
		wf.log.Warn("Filter ping failed towards peer", zap.Stringer("peer", peer), zap.Error(err))

		subscriptions := wf.subscriptions.GetAllSubscriptionsForPeer(peer)
		for _, subscription := range subscriptions {
			wf.log.Debug("Notifying sub closing", zap.String("subID", subscription.ID))

			//Indicating that subscription is closing,
			subscription.Closing <- true
		}
	}
}

func (wf *WakuFilterLightNode) FilterHealthCheckLoop() {
	defer wf.WaitGroup().Done()
	ticker := time.NewTicker(wf.peerPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			wf.PingPeers()
		case <-wf.CommonService.Context().Done():
			return
		}
	}
}
