package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p"
	"go.uber.org/zap"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	ma "github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"

	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/try"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/protocol/swap"

	"github.com/waku-org/go-waku/waku/v2/utils"
)

type Peer struct {
	ID        peer.ID        `json:"peerID"`
	Protocols []string       `json:"protocols"`
	Addrs     []ma.Multiaddr `json:"addrs"`
	Connected bool           `json:"connected"`
}

type storeFactory func(w *WakuNode) store.Store

type MembershipKeyPair = struct {
	IDKey        [32]byte `json:"idKey"`
	IDCommitment [32]byte `json:"idCommitment"`
}

type RLNRelay interface {
	MembershipKeyPair() *MembershipKeyPair
	MembershipIndex() uint
	MembershipContractAddress() common.Address
	AppendRLNProof(msg *pb.WakuMessage, senderEpochTime time.Time) error
	Stop()
}

type WakuNode struct {
	host host.Host
	opts *WakuNodeParameters
	log  *zap.Logger

	relay     *relay.WakuRelay
	filter    *filter.WakuFilter
	lightPush *lightpush.WakuLightPush
	store     store.Store
	swap      *swap.WakuSwap
	rlnRelay  RLNRelay
	wakuFlag  utils.WakuEnrBitfield

	localNode *enode.LocalNode

	addrChan chan ma.Multiaddr

	discoveryV5  *discv5.DiscoveryV5
	peerExchange *peer_exchange.WakuPeerExchange

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

	storeFactory storeFactory
}

func defaultStoreFactory(w *WakuNode) store.Store {
	return store.NewWakuStore(w.host, w.swap, w.opts.messageProvider, w.log)
}

// New is used to instantiate a WakuNode using a set of WakuNodeOptions
func New(ctx context.Context, opts ...WakuNodeOption) (*WakuNode, error) {
	params := new(WakuNodeParameters)

	params.libP2POpts = DefaultLibP2POptions

	opts = append(DefaultWakuNodeOptions, opts...)
	for _, opt := range opts {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	if params.privKey == nil {
		prvKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		params.privKey = prvKey
	}

	if params.enableWSS {
		params.libP2POpts = append(params.libP2POpts, libp2p.Transport(ws.New, ws.WithTLSConfig(params.tlsConfig)))
	} else {
		// Enable WS transport by default
		params.libP2POpts = append(params.libP2POpts, libp2p.Transport(ws.New))
	}

	// Setting default host address if none was provided
	if params.hostAddr == nil {
		err := WithHostAddress(&net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0})(params)
		if err != nil {
			return nil, err
		}
	}
	if len(params.multiAddr) > 0 {
		params.libP2POpts = append(params.libP2POpts, libp2p.ListenAddrs(params.multiAddr...))
	}

	params.libP2POpts = append(params.libP2POpts, params.Identity())

	if params.addressFactory != nil {
		params.libP2POpts = append(params.libP2POpts, libp2p.AddrsFactory(params.addressFactory))
	}

	host, err := libp2p.New(params.libP2POpts...)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

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

	if params.storeFactory != nil {
		w.storeFactory = params.storeFactory
	} else {
		w.storeFactory = defaultStoreFactory
	}

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
		_ = m
		// TODO: determine if still needed. Otherwise remove
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
			close(w.addrChan)
			return
		case <-first:
			w.log.Info("listening", logging.MultiAddrs("multiaddr", addrs...))
		case <-w.addressChangesSub.Out():
			newAddrs := w.ListenAddresses()
			diff := false
			if len(addrs) != len(newAddrs) {
				diff = true
			} else {
				for i := range newAddrs {
					if addrs[i].String() != newAddrs[i].String() {
						diff = true
						break
					}
				}
			}
			if diff {
				addrs = newAddrs
				w.log.Info("listening addresses update received", logging.MultiAddrs("multiaddr", addrs...))
				for _, addr := range addrs {
					w.addrChan <- addr
				}
				_ = w.setupENR(addrs)
			}
		}
	}
}

// Start initializes all the protocols that were setup in the WakuNode
func (w *WakuNode) Start() error {
	if w.opts.enableSwap {
		w.swap = swap.NewWakuSwap(w.log, []swap.SwapOption{
			swap.WithMode(w.opts.swapMode),
			swap.WithThreshold(w.opts.swapPaymentThreshold, w.opts.swapDisconnectThreshold),
		}...)
	}

	w.store = w.storeFactory(w)
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

	err := w.setupENR(w.ListenAddresses())
	if err != nil {
		return err
	}

	if w.opts.enableDiscV5 {
		err := w.mountDiscV5()
		if err != nil {
			return err
		}
	}

	if w.opts.enablePeerExchange {
		err := w.mountPeerExchange()
		if err != nil {
			return err
		}
	}

	if w.opts.enableDiscV5 {
		w.opts.wOpts = append(w.opts.wOpts, pubsub.WithDiscovery(w.discoveryV5, w.opts.discV5Opts...))
	}

	err = w.mountRelay(w.opts.minRelayPeersToPublish, w.opts.wOpts...)
	if err != nil {
		return err
	}

	if w.opts.enableRLN {
		err = w.mountRlnRelay()
		if err != nil {
			return err
		}
	}

	w.lightPush = lightpush.NewWakuLightPush(w.ctx, w.host, w.relay, w.log)
	if w.opts.enableLightPush {
		if err := w.lightPush.Start(); err != nil {
			return err
		}
	}

	// Subscribe store to topic
	if w.opts.storeMsgs {
		w.log.Info("Subscribing store to broadcaster")
		w.bcaster.Register(nil, w.store.MessageChannel())
	}

	if w.filter != nil {
		w.log.Info("Subscribing filter to broadcaster")
		w.bcaster.Register(nil, w.filter.MsgC)
	}

	return nil
}

// Stop stops the WakuNode and closess all connections to the host
func (w *WakuNode) Stop() {
	defer w.cancel()

	close(w.quit)

	w.bcaster.Close()

	defer w.connectionNotif.Close()
	defer w.protocolEventSub.Close()
	defer w.identificationEventSub.Close()
	defer w.addressChangesSub.Close()

	if w.filter != nil {
		w.filter.Stop()
	}

	if w.peerExchange != nil {
		w.peerExchange.Stop()
	}

	if w.discoveryV5 != nil {
		w.discoveryV5.Stop()
	}

	w.relay.Stop()
	w.lightPush.Stop()
	w.store.Stop()
	_ = w.stopRlnRelay()

	w.host.Close()

	w.wg.Wait()
}

// Host returns the libp2p Host used by the WakuNode
func (w *WakuNode) Host() host.Host {
	return w.host
}

// ID returns the base58 encoded ID from the host
func (w *WakuNode) ID() string {
	return w.host.ID().Pretty()
}

// ListenAddresses returns all the multiaddresses used by the host
func (w *WakuNode) ListenAddresses() []ma.Multiaddr {
	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", w.host.ID().Pretty()))
	var result []ma.Multiaddr
	for _, addr := range w.host.Addrs() {
		result = append(result, addr.Encapsulate(hostInfo))
	}
	return result
}

// ENR returns the ENR address of the node
func (w *WakuNode) ENR() *enode.Node {
	return w.localNode.Node()
}

// Relay is used to access any operation related to Waku Relay protocol
func (w *WakuNode) Relay() *relay.WakuRelay {
	return w.relay
}

// Store is used to access any operation related to Waku Store protocol
func (w *WakuNode) Store() store.Store {
	return w.store
}

// Filter is used to access any operation related to Waku Filter protocol
func (w *WakuNode) Filter() *filter.WakuFilter {
	return w.filter
}

// Lightpush is used to access any operation related to Waku Lightpush protocol
func (w *WakuNode) Lightpush() *lightpush.WakuLightPush {
	return w.lightPush
}

// DiscV5 is used to access any operation related to DiscoveryV5
func (w *WakuNode) DiscV5() *discv5.DiscoveryV5 {
	return w.discoveryV5
}

// PeerExchange is used to access any operation related to Peer Exchange
func (w *WakuNode) PeerExchange() *peer_exchange.WakuPeerExchange {
	return w.peerExchange
}

// Broadcaster is used to access the message broadcaster that is used to push
// messages to different protocols
func (w *WakuNode) Broadcaster() v2.Broadcaster {
	return w.bcaster
}

// Publish will attempt to publish a message via WakuRelay if there are enough
// peers available, otherwise it will attempt to publish via Lightpush protocol
func (w *WakuNode) Publish(ctx context.Context, msg *pb.WakuMessage) error {
	if !w.opts.enableLightPush && !w.opts.enableRelay {
		return errors.New("cannot publish message, relay and lightpush are disabled")
	}

	hash, _, _ := msg.Hash()
	err := try.Do(func(attempt int) (bool, error) {
		var err error
		if !w.relay.EnoughPeersToPublish() {
			if !w.lightPush.IsStarted() {
				err = errors.New("not enought peers for relay and lightpush is not yet started")
			} else {
				w.log.Debug("publishing message via lightpush", logging.HexBytes("hash", hash))
				_, err = w.Lightpush().Publish(ctx, msg)
			}
		} else {
			w.log.Debug("publishing message via relay", logging.HexBytes("hash", hash))
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
		w.Broadcaster().Unregister(&relay.DefaultWakuTopic, sub.C)
	}

	return err
}

func (w *WakuNode) mountDiscV5() error {
	discV5Options := []discv5.DiscoveryV5Option{
		discv5.WithBootnodes(w.opts.discV5bootnodes),
		discv5.WithUDPPort(w.opts.udpPort),
		discv5.WithAutoUpdate(w.opts.discV5autoUpdate),
	}

	if w.opts.advertiseAddr != nil {
		discV5Options = append(discV5Options, discv5.WithAdvertiseAddr(*w.opts.advertiseAddr))
	}

	var err error
	w.discoveryV5, err = discv5.NewDiscoveryV5(w.ctx, w.Host(), w.opts.privKey, w.localNode, w.log, discV5Options...)

	return err
}

func (w *WakuNode) mountPeerExchange() error {
	w.peerExchange = peer_exchange.NewWakuPeerExchange(w.ctx, w.host, w.discoveryV5, w.log)
	return w.peerExchange.Start()
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
						_, err := utils.SelectPeer(w.host, string(store.StoreID_v20beta4), nil, w.log)
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

func (w *WakuNode) addPeer(info *peer.AddrInfo, protocols ...string) error {
	w.log.Info("adding peer to peerstore", logging.HostID("peer", info.ID))
	w.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	err := w.host.Peerstore().AddProtocols(info.ID, protocols...)
	if err != nil {
		return err
	}

	return nil
}

// AddPeer is used to add a peer and the protocols it support to the node peerstore
func (w *WakuNode) AddPeer(address ma.Multiaddr, protocols ...string) (*peer.ID, error) {
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return nil, err
	}

	return &info.ID, w.addPeer(info, protocols...)
}

// DialPeerWithMultiAddress is used to connect to a peer using a multiaddress
func (w *WakuNode) DialPeerWithMultiAddress(ctx context.Context, address ma.Multiaddr) error {
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return err
	}

	return w.connect(ctx, *info)
}

// DialPeer is used to connect to a peer using a string containing a multiaddress
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

// DialPeerByID is used to connect to an already known peer
func (w *WakuNode) DialPeerByID(ctx context.Context, peerID peer.ID) error {
	info := w.host.Peerstore().PeerInfo(peerID)
	return w.connect(ctx, info)
}

// ClosePeerByAddress is used to disconnect from a peer using its multiaddress
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

// ClosePeerById is used to close a connection to a peer
func (w *WakuNode) ClosePeerById(id peer.ID) error {
	err := w.host.Network().ClosePeer(id)
	if err != nil {
		return err
	}
	return nil
}

// PeerCount return the number of connected peers
func (w *WakuNode) PeerCount() int {
	return len(w.host.Network().Peers())
}

// PeerStats returns a list of peers and the protocols supported by them
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

// Peers return the list of peers, addresses, protocols supported and connection status
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
