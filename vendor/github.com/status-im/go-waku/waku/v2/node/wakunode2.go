package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	proto "github.com/golang/protobuf/proto"
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
	"go.opencensus.io/tag"

	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
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
	ping       *ping.PingService
	store      *store.WakuStore

	subscriptions      map[relay.Topic][]*Subscription
	subscriptionsMutex sync.Mutex

	bcaster Broadcaster

	filters filter.Filters

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
	w.bcaster = NewBroadcaster(1024)
	w.host = host
	w.cancel = cancel
	w.ctx = ctx
	w.subscriptions = make(map[relay.Topic][]*Subscription)
	w.opts = params
	w.quit = make(chan struct{})

	if w.protocolEventSub, err = host.EventBus().Subscribe(new(event.EvtPeerProtocolsUpdated)); err != nil {
		return nil, err
	}

	if w.identificationEventSub, err = host.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted)); err != nil {
		return nil, err
	}

	if params.connStatusChan != nil {
		w.connStatusChan = params.connStatusChan
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
	if w.opts.enableStore {
		w.startStore()
	}

	if w.opts.enableFilter {
		w.filters = make(filter.Filters)
		err := w.mountFilter()
		if err != nil {
			return err
		}
	}

	if w.opts.enableRendezvous {
		rendezvous := rendezvous.NewRendezvousDiscovery(w.host)
		w.opts.wOpts = append(w.opts.wOpts, pubsub.WithDiscovery(rendezvous, w.opts.rendezvousOpts...))
	}

	err := w.mountRelay(w.opts.enableRelay, w.opts.wOpts...)
	if err != nil {
		return err
	}

	if w.opts.enableLightPush {
		w.mountLightPush()
	}

	if w.opts.enableRendezvousServer {
		err := w.mountRendezvous()
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *WakuNode) Stop() {
	w.subscriptionsMutex.Lock()
	defer w.subscriptionsMutex.Unlock()
	defer w.cancel()

	close(w.quit)

	defer w.connectionNotif.Close()
	defer w.protocolEventSub.Close()
	defer w.identificationEventSub.Close()

	if w.rendezvous != nil {
		w.rendezvous.Stop()
	}

	if w.relay != nil {
		for _, topic := range w.relay.Topics() {
			for _, sub := range w.subscriptions[topic] {
				sub.Unsubscribe()
			}
		}
		w.subscriptions = nil
	}

	if w.filter != nil {
		w.filter.Stop()
		for _, filter := range w.filters {
			close(filter.Chan)
		}
		w.filters = nil
	}

	if w.lightPush != nil {
		w.lightPush.Stop()
	}

	if w.store != nil {
		w.store.Stop()
	}

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

func (w *WakuNode) Filter() *filter.WakuFilter {
	return w.filter
}

func (w *WakuNode) mountRelay(shouldRelayMessages bool, opts ...pubsub.Option) error {
	var err error
	w.relay, err = relay.NewWakuRelay(w.ctx, w.host, opts...)

	if shouldRelayMessages {
		_, err := w.Subscribe(w.ctx, nil)
		if err != nil {
			return err
		}
	}

	// TODO: rlnRelay

	return err
}

func (w *WakuNode) mountFilter() error {
	filterHandler := func(requestId string, msg pb.MessagePush) {
		for _, message := range msg.Messages {
			w.filters.Notify(message, requestId) // Trigger filter handlers on a light node
		}
	}

	w.filter = filter.NewWakuFilter(w.ctx, w.host, filterHandler)

	return nil
}

func (w *WakuNode) mountLightPush() {
	w.lightPush = lightpush.NewWakuLightPush(w.ctx, w.host, w.relay)
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
	w.store = w.opts.store
	w.store.Start(w.ctx, w.host)

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
				if err := w.Resume(ctxWithTimeout, nil); err != nil {
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

func (w *WakuNode) Query(ctx context.Context, contentTopics []string, startTime float64, endTime float64, opts ...store.HistoryRequestOption) (*pb.HistoryResponse, error) {
	if w.store == nil {
		return nil, errors.New("WakuStore is not set")
	}

	query := new(pb.HistoryQuery)

	for _, ct := range contentTopics {
		query.ContentFilters = append(query.ContentFilters, &pb.ContentFilter{ContentTopic: ct})
	}

	query.StartTime = startTime
	query.EndTime = endTime
	query.PagingInfo = new(pb.PagingInfo)
	result, err := w.store.Query(ctx, query, opts...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (w *WakuNode) Resume(ctx context.Context, peerList []peer.ID) error {
	if w.store == nil {
		return errors.New("WakuStore is not set")
	}

	result, err := w.store.Resume(ctx, string(relay.DefaultWakuTopic), peerList)
	if err != nil {
		return err
	}

	log.Info("Retrieved messages since the last online time: ", result)

	return nil
}

func (node *WakuNode) Subscribe(ctx context.Context, topic *relay.Topic) (*Subscription, error) {
	// Subscribes to a PubSub topic.
	// NOTE The data field SHOULD be decoded as a WakuMessage.
	if node.relay == nil {
		return nil, errors.New("WakuRelay hasn't been set")
	}

	t := relay.GetTopic(topic)
	sub, isNew, err := node.relay.Subscribe(t)

	// Subscribe store to topic
	if isNew && node.opts.store != nil && node.opts.storeMsgs {
		log.Info("Subscribing store to topic ", t)
		node.bcaster.Register(node.opts.store.MsgC)
	}

	// Subscribe filter
	if isNew && node.filter != nil {
		log.Info("Subscribing filter to topic ", t)
		node.bcaster.Register(node.filter.MsgC)
	}

	if err != nil {
		return nil, err
	}

	// Create client subscription
	subscription := new(Subscription)
	subscription.closed = false
	subscription.C = make(chan *protocol.Envelope, 1024) // To avoid blocking
	subscription.quit = make(chan struct{})

	node.subscriptionsMutex.Lock()
	defer node.subscriptionsMutex.Unlock()

	node.subscriptions[t] = append(node.subscriptions[t], subscription)

	node.bcaster.Register(subscription.C)

	go node.subscribeToTopic(t, subscription, sub)

	return subscription, nil
}

func (node *WakuNode) subscribeToTopic(t relay.Topic, subscription *Subscription, sub *pubsub.Subscription) {
	nextMsgTicker := time.NewTicker(time.Millisecond * 10)
	defer nextMsgTicker.Stop()

	ctx, err := tag.New(node.ctx, tag.Insert(metrics.KeyType, "relay"))
	if err != nil {
		log.Error(err)
		return
	}

	for {
		select {
		case <-subscription.quit:
			subscription.mutex.Lock()
			node.bcaster.Unregister(subscription.C) // Remove from broadcast list
			close(subscription.C)
			subscription.mutex.Unlock()
		case <-nextMsgTicker.C:
			msg, err := sub.Next(ctx)
			if err != nil {
				subscription.mutex.Lock()
				for _, subscription := range node.subscriptions[t] {
					subscription.Unsubscribe()
				}
				subscription.mutex.Unlock()
				return
			}

			stats.Record(ctx, metrics.Messages.M(1))

			wakuMessage := &pb.WakuMessage{}
			if err := proto.Unmarshal(msg.Data, wakuMessage); err != nil {
				log.Error("could not decode message", err)
				return
			}

			envelope := protocol.NewEnvelope(wakuMessage, string(t))

			node.bcaster.Submit(envelope)
		}
	}
}

// Wrapper around WakuFilter.Subscribe
// that adds a Filter object to node.filters
func (node *WakuNode) SubscribeFilter(ctx context.Context, f filter.ContentFilter) (filterID string, ch chan *protocol.Envelope, err error) {
	if node.filter == nil {
		err = errors.New("WakuFilter is not set")
		return
	}

	// TODO: should be possible to pass the peerID as option or autoselect peer.
	// TODO: check if there's an existing pubsub topic that uses the same peer. If so, reuse filter, and return same channel and filterID

	// Registers for messages that match a specific filter. Triggers the handler whenever a message is received.
	// ContentFilterChan takes MessagePush structs
	subs, err := node.filter.Subscribe(ctx, f)
	if subs.RequestID == "" || err != nil {
		// Failed to subscribe
		log.Error("remote subscription to filter failed", err)
		return
	}

	ch = make(chan *protocol.Envelope, 1024) // To avoid blocking

	// Register handler for filter, whether remote subscription succeeded or not
	node.filters[subs.RequestID] = filter.Filter{
		PeerID:         subs.Peer,
		Topic:          f.Topic,
		ContentFilters: f.ContentTopics,
		Chan:           ch,
	}

	return subs.RequestID, ch, nil
}

// UnsubscribeFilterByID removes a subscription to a filter node completely
// using the filterID returned when the subscription was created
func (node *WakuNode) UnsubscribeFilterByID(ctx context.Context, filterID string) error {

	var f filter.Filter
	var ok bool
	if f, ok = node.filters[filterID]; !ok {
		return errors.New("filter not found")
	}

	cf := filter.ContentFilter{
		Topic:         f.Topic,
		ContentTopics: f.ContentFilters,
	}

	err := node.filter.Unsubscribe(ctx, cf, f.PeerID)
	if err != nil {
		return err
	}

	close(f.Chan)
	delete(node.filters, filterID)

	return nil
}

// Unsubscribe filter removes content topics from a filter subscription. If all
// the contentTopics are removed the subscription is dropped completely
func (node *WakuNode) UnsubscribeFilter(ctx context.Context, cf filter.ContentFilter) error {
	// Remove local filter
	var idsToRemove []string
	for id, f := range node.filters {
		if f.Topic != cf.Topic {
			continue
		}

		// Send message to full node in order to unsubscribe
		err := node.filter.Unsubscribe(ctx, cf, f.PeerID)
		if err != nil {
			return err
		}

		// Iterate filter entries to remove matching content topics
		// make sure we delete the content filter
		// if no more topics are left
		for _, cfToDelete := range cf.ContentTopics {
			for i, cf := range f.ContentFilters {
				if cf == cfToDelete {
					l := len(f.ContentFilters) - 1
					f.ContentFilters[l], f.ContentFilters[i] = f.ContentFilters[i], f.ContentFilters[l]
					f.ContentFilters = f.ContentFilters[:l]
					break
				}

			}
			if len(f.ContentFilters) == 0 {
				idsToRemove = append(idsToRemove, id)
			}
		}
	}

	for _, rId := range idsToRemove {
		for id := range node.filters {
			if id == rId {
				close(node.filters[id].Chan)
				delete(node.filters, id)
				break
			}
		}
	}

	return nil
}

func (node *WakuNode) Publish(ctx context.Context, message *pb.WakuMessage, topic *relay.Topic) ([]byte, error) {
	if node.relay == nil {
		return nil, errors.New("WakuRelay hasn't been set")
	}

	if message == nil {
		return nil, errors.New("message can't be null")
	}

	if node.lightPush != nil {
		return node.LightPush(ctx, message, topic)
	}

	hash, err := node.relay.Publish(ctx, message, topic)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func (node *WakuNode) LightPush(ctx context.Context, message *pb.WakuMessage, topic *relay.Topic, opts ...lightpush.LightPushOption) ([]byte, error) {
	if node.lightPush == nil {
		return nil, errors.New("WakuLightPush hasn't been set")
	}

	if message == nil {
		return nil, errors.New("message can't be null")
	}

	req := new(pb.PushRequest)
	req.Message = message
	req.PubsubTopic = string(relay.GetTopic(topic))

	response, err := node.lightPush.Request(ctx, req, opts...)
	if err != nil {
		return nil, err
	}

	if response.IsSuccess {
		hash, _ := message.Hash()
		return hash, nil
	} else {
		return nil, errors.New(response.Info)
	}
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

func (w *WakuNode) startKeepAlive(t time.Duration) {
	log.Info("Setting up ping protocol with duration of ", t)

	w.ping = ping.NewPingService(w.host)
	ticker := time.NewTicker(t)

	go func() {
		for {
			select {
			case <-ticker.C:
				for _, p := range w.host.Network().Peers() {
					log.Debug("Pinging ", p)
					go func(peer peer.ID) {
						ctx, cancel := context.WithTimeout(w.ctx, 3*time.Second)
						defer cancel()
						pr := w.ping.Ping(ctx, peer)
						select {
						case res := <-pr:
							if res.Error != nil {
								log.Error(fmt.Sprintf("Could not ping %s: %s", peer, res.Error.Error()))
							}
						case <-ctx.Done():
							log.Error(fmt.Sprintf("Could not ping %s: %s", peer, ctx.Err()))
						}
					}(p)
				}
			case <-w.quit:
				ticker.Stop()
				return
			}
		}
	}()
}
