package node

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	"go.uber.org/zap"

	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
)

// PeerStatis is a map of peer IDs to supported protocols
type PeerStats map[peer.ID][]protocol.ID

type PeerConnection struct {
	PeerID    peer.ID
	Connected bool
}

// ConnectionNotifier is a custom Notifier to be used to display when a peer
// connects or disconnects to the node
type ConnectionNotifier struct {
	h              host.Host
	ctx            context.Context
	log            *zap.Logger
	metrics        Metrics
	connNotifCh    chan<- PeerConnection
	DisconnectChan chan peer.ID
}

// NewConnectionNotifier creates an instance of ConnectionNotifier to react to peer connection changes
func NewConnectionNotifier(ctx context.Context, h host.Host, connNotifCh chan<- PeerConnection, metrics Metrics, log *zap.Logger) ConnectionNotifier {
	return ConnectionNotifier{
		h:              h,
		ctx:            ctx,
		DisconnectChan: make(chan peer.ID, 100),
		connNotifCh:    connNotifCh,
		metrics:        metrics,
		log:            log.Named("connection-notifier"),
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
	c.log.Info("peer connected", logging.HostID("peer", cc.RemotePeer()), zap.String("direction", cc.Stat().Direction.String()))
	if c.connNotifCh != nil {
		select {
		case c.connNotifCh <- PeerConnection{cc.RemotePeer(), true}:
		default:
			c.log.Warn("subscriber is too slow")
		}
	}
	//TODO: Move this to be stored in Waku's own peerStore
	err := c.h.Peerstore().(wps.WakuPeerstore).SetDirection(cc.RemotePeer(), cc.Stat().Direction)
	if err != nil {
		c.log.Error("Failed to set peer direction for an outgoing connection", zap.Error(err))
	}

	c.metrics.RecordPeerConnected()
	c.metrics.SetPeerStoreSize(c.h.Peerstore().Peers().Len())
}

// Disconnected is called when a connection closed
func (c ConnectionNotifier) Disconnected(n network.Network, cc network.Conn) {
	c.log.Info("peer disconnected", logging.HostID("peer", cc.RemotePeer()))
	c.metrics.RecordPeerDisconnected()
	c.DisconnectChan <- cc.RemotePeer()
	if c.connNotifCh != nil {
		select {
		case c.connNotifCh <- PeerConnection{cc.RemotePeer(), false}:
		default:
			c.log.Warn("subscriber is too slow")
		}
	}
	c.metrics.SetPeerStoreSize(c.h.Peerstore().Peers().Len())
}

// OpenedStream is called when a stream opened
func (c ConnectionNotifier) OpenedStream(n network.Network, s network.Stream) {
}

// ClosedStream is called when a stream closed
func (c ConnectionNotifier) ClosedStream(n network.Network, s network.Stream) {
}

// Close quits the ConnectionNotifier
func (c ConnectionNotifier) Close() {
}

func (w *WakuNode) connectednessListener(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.connectionNotif.DisconnectChan:
		}
	}
}
