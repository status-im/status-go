package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p"
	"go.uber.org/zap"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	p2pproto "github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"

	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/status-im/go-waku/waku/try"
	v2 "github.com/status-im/go-waku/waku/v2"
	"github.com/status-im/go-waku/waku/v2/discv5"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	"github.com/status-im/go-waku/waku/v2/protocol/swap"
	"github.com/status-im/go-waku/waku/v2/utils"
)

type Peer struct {
	ID        peer.ID
	Protocols []string
	Addrs     []ma.Multiaddr
	Connected bool
}

type WakuNode struct {
	host host.Host
	opts *WakuNodeParameters
	log  *zap.SugaredLogger

	relay      *relay.WakuRelay
	filter     *filter.WakuFilter
	lightPush  *lightpush.WakuLightPush
	rendezvous *rendezvous.RendezvousService
	store      *store.WakuStore
	swap       *swap.WakuSwap
	wakuFlag   utils.WakuEnrBitfield

	addrChan chan ma.Multiaddr

	discoveryV5 *discv5.DiscoveryV5

	bcaster v2.Broadcaster

	connectionNotif        ConnectionNotifier
	protocolEventSub       event.Subscription
	identificationEventSub event.Subscription
	addressChangesSub      event.Subscription

	keepAliveMutex sync.Mutex
	keepAliveFails map[peer.ID]int

	ctx    context.Context
	cancel context.CancelFunc
	quit   chan struct{}
	wg     *sync.WaitGroup

	// Channel passed to WakuNode constructor
	// receiving connection status notifications
	connStatusChan chan ConnStatus
}

func New(ctx context.Context, opts ...WakuNodeOption) (*WakuNode, error) {
	params := new(WakuNodeParameters)

	ctx, cancel := context.WithCancel(ctx)

	params.libP2POpts = DefaultLibP2POptions

	opts = append(DefaultWakuNodeOptions, opts...)
	for _, opt := range opts {
		err := opt(params)
		if err != nil {
			cancel()
			return nil, err
		}
	}

	// Setting default host address if none was provided
	if params.hostAddr == nil {
		err := WithHostAddress(&net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0})(params)
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
	w.log = params.logger.Named("node2")
	w.quit = make(chan struct{})
	w.wg = &sync.WaitGroup{}
	w.addrChan = make(chan ma.Multiaddr, 1024)
	w.keepAliveFails = make(map[peer.ID]int)
	w.wakuFlag = utils.NewWakuEnrBitfield(w.opts.enableLightPush, w.opts.enableFilter, w.opts.enableStore, w.opts.enableRelay)

	if w.protocolEventSub, err = host.EventBus().Subscribe(new(event.EvtPeerProtocolsUpdated)); err != nil {
		return nil, err
	}

	if w.identificationEventSub, err = host.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted)); err != nil {
		return nil, err
	}

	if w.addressChangesSub, err = host.EventBus().Subscribe(new(event.EvtLocalAddressesUpdated)); err != nil {
		return nil, err
	}

	if params.connStatusC != nil {
		w.connStatusChan = params.connStatusC
	}

	w.connectionNotif = NewConnectionNotifier(ctx, host, w.log)
	w.host.Network().Notify(w.connectionNotif)

	w.wg.Add(2)
	go w.connectednessListener()
	go w.checkForAddressChanges()
	go w.onAddrChange()

	if w.opts.keepAliveInterval > time.Duration(0) {
		w.wg.Add(1)
		w.startKeepAlive(w.opts.keepAliveInterval)
	}

	return w, nil
}

func (w *WakuNode) onAddrChange() {
	for m := range w.addrChan {
		ipStr, err := m.ValueForProtocol(ma.P_IP4)
		if err != nil {
			w.log.Error(fmt.Sprintf("could not extract ip from ma %s: %s", m, err.Error()))
			continue
		}
		ip := net.ParseIP(ipStr)
		if !ip.IsLoopback() && !ip.IsUnspecified() {
			if w.opts.enableDiscV5 {
				err := w.discoveryV5.UpdateAddr(ip)
				if err != nil {
					w.log.Error(fmt.Sprintf("could not update DiscV5 address with IP %s: %s", ip, err.Error()))
					continue
				}
			}
		}
	}
}

func (w *WakuNode) logAddress(addr ma.Multiaddr) {
	w.log.Info("Listening on ", addr)

	// TODO: make this optional depending on DNS Disc being enabled
	if w.opts.privKey != nil {
		enr, ip, err := utils.GetENRandIP(addr, w.wakuFlag, w.opts.privKey)
		if err != nil {
			w.log.Error("could not obtain ENR record from multiaddress", err)
		} else {
			w.log.Info(fmt.Sprintf("ENR for IP %s:  %s", ip, enr))
		}
	}
}

func (w *WakuNode) checkForAddressChanges() {
	defer w.wg.Done()

	addrs := w.ListenAddresses()
	first := make(chan struct{}, 1)
	first <- struct{}{}
	for {
		select {
		case <-w.quit:
			return
		case <-first:
			for _, addr := range addrs {
				w.logAddress(addr)
			}
		case <-w.addressChangesSub.Out():
			newAddrs := w.ListenAddresses()
			print := false
			if len(addrs) != len(newAddrs) {
				print = true
			} else {
				for i := range newAddrs {
					if addrs[i].String() != newAddrs[i].String() {
						print = true
						break
					}
				}
			}
			if print {
				addrs = newAddrs
				w.log.Warn("Change in host multiaddresses")
				for _, addr := range newAddrs {
					w.addrChan <- addr
					w.logAddress(addr)
				}
			}
		}
	}
}

func (w *WakuNode) Start() error {
	w.swap = swap.NewWakuSwap(w.log, []swap.SwapOption{
		swap.WithMode(w.opts.swapMode),
		swap.WithThreshold(w.opts.swapPaymentThreshold, w.opts.swapDisconnectThreshold),
	}...)

	w.store = store.NewWakuStore(w.host, w.swap, w.opts.messageProvider, w.opts.maxMessages, w.opts.maxDuration, w.log)
	if w.opts.enableStore {
		w.startStore()
	}

	if w.opts.enableFilter {
		filter, err := filter.NewWakuFilter(w.ctx, w.host, w.opts.isFilterFullNode, w.log, w.opts.filterOpts...)
		if err != nil {
			return err
		}
		w.filter = filter
	}

	if w.opts.enableRendezvous {
		rendezvous := rendezvous.NewRendezvousDiscovery(w.host)
		w.opts.wOpts = append(w.opts.wOpts, pubsub.WithDiscovery(rendezvous, w.opts.rendezvousOpts...))
	}

	if w.opts.enableDiscV5 {
		err := w.mountDiscV5()
		if err != nil {
			return err
		}
	}

	if w.opts.enableDiscV5 {
		w.opts.wOpts = append(w.opts.wOpts, pubsub.WithDiscovery(w.discoveryV5, w.opts.discV5Opts...))
	}

	err := w.mountRelay(w.opts.minRelayPeersToPublish, w.opts.wOpts...)
	if err != nil {
		return err
	}

	w.lightPush = lightpush.NewWakuLightPush(w.ctx, w.host, w.relay, w.log)
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
		w.log.Info("Subscribing store to broadcaster")
		w.bcaster.Register(w.store.MsgC)
	}

	if w.filter != nil {
		w.log.Info("Subscribing filter to broadcaster")
		w.bcaster.Register(w.filter.MsgC)
	}

	return nil
}

func (w *WakuNode) Stop() {
	defer w.cancel()

	close(w.quit)
	close(w.addrChan)

	w.bcaster.Close()

	defer w.connectionNotif.Close()
	defer w.protocolEventSub.Close()
	defer w.identificationEventSub.Close()
	defer w.addressChangesSub.Close()

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

	w.wg.Wait()
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

func (w *WakuNode) DiscV5() *discv5.DiscoveryV5 {
	return w.discoveryV5
}

func (w *WakuNode) Broadcaster() v2.Broadcaster {
	return w.bcaster
}

func (w *WakuNode) Publish(ctx context.Context, msg *pb.WakuMessage) error {
	if !w.opts.enableLightPush && !w.opts.enableRelay {
		return errors.New("cannot publish message, relay and lightpush are disabled")
	}

	hash, _ := msg.Hash()
	err := try.Do(func(attempt int) (bool, error) {
		var err error
		if !w.relay.EnoughPeersToPublish() {
			if !w.lightPush.IsStarted() {
				err = errors.New("not enought peers for relay and lightpush is not yet started")
			} else {
				w.log.Debug("publishing message via lightpush", hexutil.Encode(hash))
				_, err = w.Lightpush().Publish(ctx, msg)
			}
		} else {
			w.log.Debug("publishing message via relay", hexutil.Encode(hash))
			_, err = w.Relay().Publish(ctx, msg)
		}

		return attempt < maxPublishAttempt, err
	})

	return err
}

func (w *WakuNode) mountRelay(minRelayPeersToPublish int, opts ...pubsub.Option) error {
	var err error
	w.relay, err = relay.NewWakuRelay(w.ctx, w.host, w.bcaster, minRelayPeersToPublish, w.log, opts...)
	if err != nil {
		return err
	}

	if w.opts.enableRelay {
		sub, err := w.relay.Subscribe(w.ctx)
		if err != nil {
			return err
		}
		w.Broadcaster().Unregister(sub.C)
	}

	// TODO: rlnRelay

	return err
}

func (w *WakuNode) mountDiscV5() error {
	discV5Options := []discv5.DiscoveryV5Option{
		discv5.WithBootnodes(w.opts.discV5bootnodes),
		discv5.WithUDPPort(w.opts.udpPort),
		discv5.WithAutoUpdate(w.opts.discV5autoUpdate),
	}

	addr := w.ListenAddresses()[0]

	ipStr, err := addr.ValueForProtocol(ma.P_IP4)
	if err != nil {
		return err
	}

	portStr, err := addr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}

	w.discoveryV5, err = discv5.NewDiscoveryV5(w.Host(), net.ParseIP(ipStr), port, w.opts.privKey, w.wakuFlag, w.log, discV5Options...)

	return err
}

func (w *WakuNode) mountRendezvous() error {
	w.rendezvous = rendezvous.NewRendezvousService(w.host, w.opts.rendevousStorage)

	if err := w.rendezvous.Start(); err != nil {
		return err
	}

	w.log.Info("Rendezvous service started")
	return nil
}

func (w *WakuNode) startStore() {
	w.store.Start(w.ctx)

	if w.opts.shouldResume {
		// TODO: extract this to a function and run it when you go offline
		// TODO: determine if a store is listening to a topic
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()

			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
			peerVerif:
				for {
					select {
					case <-w.quit:
						return
					case <-ticker.C:
						_, err := utils.SelectPeer(w.host, string(store.StoreID_v20beta3), w.log)
						if err == nil {
							break peerVerif
						}
					}
				}

				ctxWithTimeout, ctxCancel := context.WithTimeout(w.ctx, 20*time.Second)
				defer ctxCancel()
				if _, err := w.store.Resume(ctxWithTimeout, string(relay.DefaultWakuTopic), nil); err != nil {
					w.log.Info("Retrying in 10s...")
					time.Sleep(10 * time.Second)
				} else {
					break
				}
			}
		}()
	}
}

func (w *WakuNode) addPeer(info *peer.AddrInfo, protocolID p2pproto.ID) error {
	w.log.Info(fmt.Sprintf("Adding peer %s to peerstore", info.ID.Pretty()))
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

func (w *WakuNode) PeerStats() PeerStats {
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

func (w *WakuNode) Peers() ([]*Peer, error) {
	var peers []*Peer
	for _, peerId := range w.host.Peerstore().Peers() {
		connected := w.host.Network().Connectedness(peerId) == network.Connected
		protocols, err := w.host.Peerstore().GetProtocols(peerId)
		if err != nil {
			return nil, err
		}

		addrs := w.host.Peerstore().Addrs(peerId)
		peers = append(peers, &Peer{
			ID:        peerId,
			Protocols: protocols,
			Connected: connected,
			Addrs:     addrs,
		})
	}
	return peers, nil
}
