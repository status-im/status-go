package node

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	"go.opencensus.io/stats"
)

// A map of peer IDs to supported protocols
type PeerStats map[peer.ID][]string

type ConnStatus struct {
	IsOnline   bool
	HasHistory bool
	Peers      PeerStats
}

type ConnectionNotifier struct {
	h              host.Host
	ctx            context.Context
	DisconnectChan chan peer.ID
	quit           chan struct{}
}

func NewConnectionNotifier(ctx context.Context, h host.Host) ConnectionNotifier {
	return ConnectionNotifier{
		h:              h,
		ctx:            ctx,
		DisconnectChan: make(chan peer.ID, 100),
		quit:           make(chan struct{}),
	}
}

func (c ConnectionNotifier) Listen(n network.Network, m ma.Multiaddr) {
	// called when network starts listening on an addr
}

func (c ConnectionNotifier) ListenClose(n network.Network, m ma.Multiaddr) {
	// called when network stops listening on an addr
}

func (c ConnectionNotifier) Connected(n network.Network, cc network.Conn) {
	// called when a connection opened
	log.Info(fmt.Sprintf("Peer %s connected", cc.RemotePeer()))
	stats.Record(c.ctx, metrics.Peers.M(1))
}

func (c ConnectionNotifier) Disconnected(n network.Network, cc network.Conn) {
	// called when a connection closed
	log.Info(fmt.Sprintf("Peer %s disconnected", cc.RemotePeer()))
	stats.Record(c.ctx, metrics.Peers.M(-1))
	c.DisconnectChan <- cc.RemotePeer()
}

func (c ConnectionNotifier) OpenedStream(n network.Network, s network.Stream) {
	// called when a stream opened
}

func (c ConnectionNotifier) ClosedStream(n network.Network, s network.Stream) {
	// called when a stream closed
}

func (c ConnectionNotifier) Close() {
	close(c.quit)
}

func (w *WakuNode) sendConnStatus() {
	isOnline, hasHistory := w.Status()
	if w.connStatusChan != nil {
		connStatus := ConnStatus{IsOnline: isOnline, HasHistory: hasHistory, Peers: w.PeerStats()}
		w.connStatusChan <- connStatus
	}

}

func (w *WakuNode) connectednessListener() {
	for {
		select {
		case <-w.quit:
			return
		case <-w.protocolEventSub.Out():
		case <-w.identificationEventSub.Out():
		case <-w.connectionNotif.DisconnectChan:
		}
		w.sendConnStatus()
	}
}

func (w *WakuNode) Status() (isOnline bool, hasHistory bool) {
	hasRelay := false
	hasLightPush := false
	hasStore := false
	hasFilter := false

	for _, peer := range w.host.Network().Peers() {
		protocols, err := w.host.Peerstore().GetProtocols(peer)
		if err != nil {
			log.Warn(fmt.Errorf("could not read peer %s protocols", peer))
		}

		for _, protocol := range protocols {
			if !hasRelay && protocol == string(relay.WakuRelayID_v200) {
				hasRelay = true
			}
			if !hasLightPush && protocol == string(lightpush.LightPushID_v20beta1) {
				hasLightPush = true
			}
			if !hasStore && protocol == string(store.StoreID_v20beta3) {
				hasStore = true
			}
			if !hasFilter && protocol == string(filter.FilterID_v20beta1) {
				hasFilter = true
			}
		}
	}

	if hasStore {
		hasHistory = true
	}

	if hasRelay || hasLightPush && (hasStore || hasFilter) {
		isOnline = true
	}

	return
}
