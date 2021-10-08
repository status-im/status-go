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
	wakurelay "github.com/status-im/go-wakurelay-pubsub"
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
		params.libP2POpts = append(params.libP2POpts, libp2p.Identity(*params.privKey))
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

	w.connectionNotif = NewConnectionNotifier(host)
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
		w.opts.wOpts = append(w.opts.wOpts, wakurelay.WithDiscovery(rendezvous, w.opts.rendezvousOpts...))
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

	for _, topic := range w.relay.Topics() {
		for _, sub := range w.subscriptions[topic] {
			sub.Unsubscribe()
		}
	}

	w.subscriptions = nil

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

func (w *WakuNode) mountRelay(shouldRelayMessages bool, opts ...wakurelay.Option) error {
	var err error
	w.relay, err = relay.NewWakuRelay(w.ctx, w.host, opts...)

	if shouldRelayMessages {
		_, err := w.Subscribe(nil)
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
	w.opts.store.Start(w.ctx, w.host)

	if w.opts.shouldResume {
		if _, err := w.opts.store.Resume(string(relay.GetTopic(nil)), nil); err != nil {
			log.Error("failed to resume", err)
		}
	}
}

func (w *WakuNode) addPeer(info *peer.AddrInfo, protocolID p2pproto.ID) error {
	log.Info(fmt.Sprintf("adding peer %s", info.ID.Pretty()))
	w.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	return w.host.Peerstore().AddProtocols(info.ID, string(protocolID))

}

func (w *WakuNode) AddPeer(address ma.Multiaddr, protocolID p2pproto.ID) (*peer.ID, error) {
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return nil, err
	}

	return &info.ID, w.addPeer(info, protocolID)
}

func (w *WakuNode) Query(ctx context.Context, contentTopics []string, startTime float64, endTime float64, opts ...store.HistoryRequestOption) (*pb.HistoryResponse, error) {
	if w.opts.store == nil {
		return nil, errors.New("WakuStore is not set")
	}

	query := new(pb.HistoryQuery)

	for _, ct := range contentTopics {
		query.ContentFilters = append(query.ContentFilters, &pb.ContentFilter{ContentTopic: ct})
	}

	query.StartTime = startTime
	query.EndTime = endTime
	query.PagingInfo = new(pb.PagingInfo)
	result, err := w.opts.store.Query(ctx, query, opts...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (w *WakuNode) Resume(ctx context.Context, peerList []peer.ID) error {
	if w.opts.store == nil {
		return errors.New("WakuStore is not set")
	}

	result, err := w.opts.store.Resume(string(relay.DefaultWakuTopic), peerList)
	if err != nil {
		return err
	}

	log.Info("the number of retrieved messages since the last online time: ", result)

	return nil
}

func (node *WakuNode) Subscribe(topic *relay.Topic) (*Subscription, error) {
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

	go func(t relay.Topic) {
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
				msg, err := sub.Next(node.ctx)
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
	}(t)

	return subscription, nil
}

// Wrapper around WakuFilter.Subscribe
// that adds a Filter object to node.filters
func (node *WakuNode) SubscribeFilter(ctx context.Context, request pb.FilterRequest, ch filter.ContentFilterChan) error {
	// Registers for messages that match a specific filter. Triggers the handler whenever a message is received.
	// ContentFilterChan takes MessagePush structs

	// Status: Implemented.

	// Sanity check for well-formed subscribe FilterRequest
	//doAssert(request.subscribe, "invalid subscribe request")

	log.Info("SubscribeFilter, request: ", request)

	var id string
	var err error

	if node.filter == nil {
		return errors.New("WakuFilter is not set")
	}

	id, err = node.filter.Subscribe(ctx, request)
	if id == "" || err != nil {
		// Failed to subscribe
		log.Error("remote subscription to filter failed", request)
		//waku_node_errors.inc(labelValues = ["subscribe_filter_failure"])
		return err
	}

	// Register handler for filter, whether remote subscription succeeded or not
	node.filters[id] = filter.Filter{
		Topic:          request.Topic,
		ContentFilters: request.ContentFilters,
		Chan:           ch,
	}

	return nil
}

func (node *WakuNode) UnsubscribeFilter(ctx context.Context, request pb.FilterRequest) {

	log.Info("UnsubscribeFilter, request: ", request)
	// Send message to full node in order to unsubscribe
	node.filter.Unsubscribe(ctx, request)

	// Remove local filter
	var idsToRemove []string
	for id, f := range node.filters {
		// Iterate filter entries to remove matching content topics
		// make sure we delete the content filter
		// if no more topics are left
		for _, cfToDelete := range request.ContentFilters {
			for i, cf := range f.ContentFilters {
				if cf.ContentTopic == cfToDelete.ContentTopic {
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
				delete(node.filters, id)
				break
			}
		}
	}
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
				for _, peer := range w.host.Network().Peers() {
					log.Debug("Pinging", peer)
					w.ping.Ping(w.ctx, peer)
				}
			case <-w.quit:
				ticker.Stop()
				return
			}
		}
	}()
}
