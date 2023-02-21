package node

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"go.opencensus.io/stats"
	"go.uber.org/zap"
)

// PeerStatis is a map of peer IDs to supported protocols
type PeerStats map[peer.ID][]protocol.ID

// ConnStatus is used to indicate if the node is online, has access to history
// and also see the list of peers the node is aware of
type ConnStatus struct {
	IsOnline   bool
	HasHistory bool
	Peers      PeerStats
}

// ConnectionNotifier is a custom Notifier to be used to display when a peer
// connects or disconnects to the node
type ConnectionNotifier struct {
	h              host.Host
	ctx            context.Context
	log            *zap.Logger
	DisconnectChan chan peer.ID
	quit           chan struct{}
}

func NewConnectionNotifier(ctx context.Context, h host.Host, log *zap.Logger) ConnectionNotifier {
	return ConnectionNotifier{
		h:              h,
		ctx:            ctx,
		DisconnectChan: make(chan peer.ID, 100),
		quit:           make(chan struct{}),
		log:            log,
	}
}

// Listen is called when network starts listening on an addr
func (c ConnectionNotifier) Listen(n network.Network, m multiaddr.Multiaddr) {
}

// ListenClose is called when network stops listening on an address
func (c ConnectionNotifier) ListenClose(n network.Network, m multiaddr.Multiaddr) {
}

// Connected is called when a connection is opened
func (c ConnectionNotifier) Connected(n network.Network, cc network.Conn) {
	c.log.Info("peer connected", logging.HostID("peer", cc.RemotePeer()))
	stats.Record(c.ctx, metrics.Peers.M(1))
}

// Disconnected is called when a connection closed
func (c ConnectionNotifier) Disconnected(n network.Network, cc network.Conn) {
	c.log.Info("peer disconnected", logging.HostID("peer", cc.RemotePeer()))
	stats.Record(c.ctx, metrics.Peers.M(-1))
	c.DisconnectChan <- cc.RemotePeer()
}

// OpenedStream is called when a stream opened
func (c ConnectionNotifier) OpenedStream(n network.Network, s network.Stream) {
}

// ClosedStream is called when a stream closed
func (c ConnectionNotifier) ClosedStream(n network.Network, s network.Stream) {
}

// Close quits the ConnectionNotifier
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

func (w *WakuNode) connectednessListener(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.protocolEventSub.Out():
		case <-w.identificationEventSub.Out():
		case <-w.connectionNotif.DisconnectChan:
		}
		w.sendConnStatus()
	}
}

// Status returns the current status of the node (online or not)
// and if the node has access to history nodes or not
func (w *WakuNode) Status() (isOnline bool, hasHistory bool) {
	hasRelay := false
	hasLightPush := false
	hasStore := false
	hasFilter := false

	for _, peer := range w.host.Network().Peers() {
		protocols, err := w.host.Peerstore().GetProtocols(peer)
		if err != nil {
			w.log.Warn("reading peer protocols", logging.HostID("peer", peer), zap.Error(err))
		}

		for _, protocol := range protocols {
			if !hasRelay && protocol == relay.WakuRelayID_v200 {
				hasRelay = true
			}
			if !hasLightPush && protocol == lightpush.LightPushID_v20beta1 {
				hasLightPush = true
			}
			if !hasStore && protocol == store.StoreID_v20beta4 {
				hasStore = true
			}
			if !hasFilter && protocol == filter.FilterID_v20beta1 {
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
