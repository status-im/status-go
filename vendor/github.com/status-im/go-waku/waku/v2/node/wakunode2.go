package node

import (
	"context"
	"fmt"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	p2pproto "github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"

	rendezvous "github.com/status-im/go-waku-rendezvous"
	v2 "github.com/status-im/go-waku/waku/v2"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	"github.com/status-im/go-waku/waku/v2/utils"
)

var log = logging.Logger("wakunode")

type Message []byte

type WakuNode struct {
	host host.Host
	opts *WakuNodeParameters

	relay      *relay.WakuRelay
	filter     *filter.WakuFilter
	lightPush  *lightpush.WakuLightPush
	rendezvous *rendezvous.RendezvousService
	store      *store.WakuStore

	bcaster v2.Broadcaster

	connectionNotif        ConnectionNotifier
	protocolEventSub       event.Subscription
	identificationEventSub event.Subscription

	ctx    context.Context
	cancel context.CancelFunc
	quit   chan struct{}

	// Channel passed to WakuNode constructor
	// receiving connection status notifications
	connStatusChan chan ConnStatus
}

func New(ctx context.Context, opts ...WakuNodeOption) (*WakuNode, error) {
	params := new(WakuNodeParameters)

	ctx, cancel := context.WithCancel(ctx)

	params.libP2POpts = DefaultLibP2POptions

	for _, opt := range opts {
		err := opt(params)
		if err != nil {
			cancel()
			return nil, err
		}
	}

	if len(params.multiAddr) > 0 {
		params.libP2POpts = append(params.libP2POpts, libp2p.ListenAddrs(params.multiAddr...))
	}

	if params.privKey != nil {
		params.libP2POpts = append(params.libP2POpts, params.Identity())
	}

	if params.addressFactory != nil {
		params.libP2POpts = append(params.libP2POpts, libp2p.AddrsFactory(params.addressFactory))
	}

	host, err := libp2p.New(ctx, params.libP2POpts...)
	if err != nil {
		cancel()
		return nil, err
	}

	w := new(WakuNode)
	w.bcaster = v2.NewBroadcaster(1024)
	w.host = host
	w.cancel = cancel
	w.ctx = ctx
	w.opts = params
	w.quit = make(chan struct{})

	if w.protocolEventSub, err = host.EventBus().Subscribe(new(event.EvtPeerProtocolsUpdated)); err != nil {
		return nil, err
	}

	if w.identificationEventSub, err = host.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted)); err != nil {
		return nil, err
	}

	if params.connStatusC != nil {
		w.connStatusChan = params.connStatusC
	}

	w.connectionNotif = NewConnectionNotifier(ctx, host)
	w.host.Network().Notify(w.connectionNotif)

	go w.connectednessListener()

	if w.opts.keepAliveInterval > time.Duration(0) {
		w.startKeepAlive(w.opts.keepAliveInterval)
	}

	for _, addr := range w.ListenAddresses() {
		log.Info("Listening on ", addr)
	}

	return w, nil
}

func (w *WakuNode) Start() error {
	w.store = store.NewWakuStore(w.host, w.opts.messageProvider, w.opts.maxMessages, w.opts.maxDuration)
	if w.opts.enableStore {
		w.startStore()
	}

	if w.opts.enableFilter {
		w.filter = filter.NewWakuFilter(w.ctx, w.host, w.opts.isFilterFullNode)
	}

	if w.opts.enableRendezvous {
		rendezvous := rendezvous.NewRendezvousDiscovery(w.host)
		w.opts.wOpts = append(w.opts.wOpts, pubsub.WithDiscovery(rendezvous, w.opts.rendezvousOpts...))
	}

	err := w.mountRelay(w.opts.wOpts...)
	if err != nil {
		return err
	}

	w.lightPush = lightpush.NewWakuLightPush(w.ctx, w.host, w.relay)
	if w.opts.enableLightPush {
		if err := w.lightPush.Start(); err != nil {
			return err
		}
	}

	if w.opts.enableRendezvousServer {
		err := w.mountRendezvous()
		if err != nil {
			return err
		}
	}

	// Subscribe store to topic
	if w.opts.storeMsgs {
		log.Info("Subscribing store to broadcaster")
		w.bcaster.Register(w.store.MsgC)
	}

	if w.filter != nil {
		log.Info("Subscribing filter to broadcaster")
		w.bcaster.Register(w.filter.MsgC)
	}

	return nil
}

func (w *WakuNode) Stop() {
	defer w.cancel()

	close(w.quit)

	w.bcaster.Close()

	defer w.connectionNotif.Close()
	defer w.protocolEventSub.Close()
	defer w.identificationEventSub.Close()

	if w.rendezvous != nil {
		w.rendezvous.Stop()
	}

	if w.filter != nil {
		w.filter.Stop()
	}

	w.relay.Stop()
	w.lightPush.Stop()
	w.store.Stop()

	w.host.Close()
}

func (w *WakuNode) Host() host.Host {
	return w.host
}

func (w *WakuNode) ID() string {
	return w.host.ID().Pretty()
}

func (w *WakuNode) ListenAddresses() []ma.Multiaddr {
	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", w.host.ID().Pretty()))
	var result []ma.Multiaddr
	for _, addr := range w.host.Addrs() {
		result = append(result, addr.Encapsulate(hostInfo))
	}
	return result
}

func (w *WakuNode) Relay() *relay.WakuRelay {
	return w.relay
}

func (w *WakuNode) Store() *store.WakuStore {
	return w.store
}

func (w *WakuNode) Filter() *filter.WakuFilter {
	return w.filter
}

func (w *WakuNode) Lightpush() *lightpush.WakuLightPush {
	return w.lightPush
}

func (w *WakuNode) mountRelay(opts ...pubsub.Option) error {
	var err error
	w.relay, err = relay.NewWakuRelay(w.ctx, w.host, w.bcaster, opts...)
	if err != nil {
		return err
	}

	if w.opts.enableRelay {
		_, err = w.relay.Subscribe(w.ctx, nil)
		if err != nil {
			return err
		}
	}

	// TODO: rlnRelay

	return err
}

func (w *WakuNode) mountRendezvous() error {
	w.rendezvous = rendezvous.NewRendezvousService(w.host, w.opts.rendevousStorage)

	if err := w.rendezvous.Start(); err != nil {
		return err
	}

	log.Info("Rendezvous service started")
	return nil
}

func (w *WakuNode) startStore() {
	w.store.Start(w.ctx)

	if w.opts.shouldResume {
		// TODO: extract this to a function and run it when you go offline
		// TODO: determine if a store is listening to a topic
		go func() {
			for {
				t := time.NewTicker(time.Second)
			peerVerif:
				for {
					select {
					case <-w.quit:
						return
					case <-t.C:
						_, err := utils.SelectPeer(w.host, string(store.StoreID_v20beta3))
						if err == nil {
							break peerVerif
						}
					}
				}

				ctxWithTimeout, ctxCancel := context.WithTimeout(w.ctx, 20*time.Second)
				defer ctxCancel()
				if _, err := w.store.Resume(ctxWithTimeout, string(relay.DefaultWakuTopic), nil); err != nil {
					log.Info("Retrying in 10s...")
					time.Sleep(10 * time.Second)
				} else {
					break
				}
			}
		}()
	}
}

func (w *WakuNode) addPeer(info *peer.AddrInfo, protocolID p2pproto.ID) error {
	log.Info(fmt.Sprintf("Adding peer %s to peerstore", info.ID.Pretty()))
	w.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	err := w.host.Peerstore().AddProtocols(info.ID, string(protocolID))
	if err != nil {
		return err
	}

	return nil
}

func (w *WakuNode) AddPeer(address ma.Multiaddr, protocolID p2pproto.ID) (*peer.ID, error) {
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return nil, err
	}

	return &info.ID, w.addPeer(info, protocolID)
}

func (w *WakuNode) DialPeerWithMultiAddress(ctx context.Context, address ma.Multiaddr) error {
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return err
	}

	return w.connect(ctx, *info)
}

func (w *WakuNode) DialPeer(ctx context.Context, address string) error {
	p, err := ma.NewMultiaddr(address)
	if err != nil {
		return err
	}

	info, err := peer.AddrInfoFromP2pAddr(p)
	if err != nil {
		return err
	}

	return w.connect(ctx, *info)
}

func (w *WakuNode) connect(ctx context.Context, info peer.AddrInfo) error {
	err := w.host.Connect(ctx, info)
	if err != nil {
		return err
	}

	stats.Record(ctx, metrics.Dials.M(1))
	return nil
}

func (w *WakuNode) DialPeerByID(ctx context.Context, peerID peer.ID) error {
	info := w.host.Peerstore().PeerInfo(peerID)
	return w.connect(ctx, info)
}

func (w *WakuNode) ClosePeerByAddress(address string) error {
	p, err := ma.NewMultiaddr(address)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(p)
	if err != nil {
		return err
	}

	return w.ClosePeerById(info.ID)
}

func (w *WakuNode) ClosePeerById(id peer.ID) error {
	err := w.host.Network().ClosePeer(id)
	if err != nil {
		return err
	}
	return nil
}

func (w *WakuNode) PeerCount() int {
	return len(w.host.Network().Peers())
}

func (w *WakuNode) Peers() PeerStats {
	p := make(PeerStats)
	for _, peerID := range w.host.Network().Peers() {
		protocols, err := w.host.Peerstore().GetProtocols(peerID)
		if err != nil {
			continue
		}
		p[peerID] = protocols
	}
	return p
}

// startKeepAlive creates a go routine that periodically pings connected peers.
// This is necessary because TCP connections are automatically closed due to inactivity,
// and doing a ping will avoid this (with a small bandwidth cost)
func (w *WakuNode) startKeepAlive(t time.Duration) {
	log.Info("Setting up ping protocol with duration of ", t)

	ticker := time.NewTicker(t)

	go func() {
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
						go pingPeer(w.ctx, w.host, p)
					}
				}
			case <-w.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func pingPeer(ctx context.Context, host host.Host, peer peer.ID) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	log.Debug("Pinging ", peer)
	pr := ping.Ping(ctx, host, peer)
	select {
	case res := <-pr:
		if res.Error != nil {
			log.Error(fmt.Sprintf("Could not ping %s: %s", peer, res.Error.Error()))
		}
	case <-ctx.Done():
		log.Error(fmt.Sprintf("Could not ping %s: %s", peer, ctx.Err()))
	}
}
