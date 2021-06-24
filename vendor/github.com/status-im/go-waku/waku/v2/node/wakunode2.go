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
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	wakurelay "github.com/status-im/go-wakurelay-pubsub"
)

var log = logging.Logger("wakunode")

// Default clientId
const clientId string = "Go Waku v2 node"

type Message []byte

type WakuNode struct {
	host host.Host
	opts *WakuNodeParameters

	relay     *relay.WakuRelay
	filter    *filter.WakuFilter
	lightPush *lightpush.WakuLightPush

	subscriptions      map[relay.Topic][]*Subscription
	subscriptionsMutex sync.Mutex

	bcaster Broadcaster

	filters filter.Filters

	ctx    context.Context
	cancel context.CancelFunc
}

func New(ctx context.Context, opts ...WakuNodeOption) (*WakuNode, error) {
	params := new(WakuNodeParameters)

	ctx, cancel := context.WithCancel(ctx)
	_ = cancel

	params.libP2POpts = DefaultLibP2POptions

	for _, opt := range opts {
		err := opt(params)
		if err != nil {
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
		return nil, err
	}

	w := new(WakuNode)
	w.bcaster = NewBroadcaster(1024)
	w.host = host
	w.cancel = cancel
	w.ctx = ctx
	w.subscriptions = make(map[relay.Topic][]*Subscription)
	w.opts = params

	if params.enableRelay {
		err := w.mountRelay(params.wOpts...)
		if err != nil {
			return nil, err
		}
	}

	if params.enableFilter {
		w.filters = make(filter.Filters)
		err := w.mountFilter()
		if err != nil {
			return nil, err
		}
	}

	if params.enableStore {
		err := w.startStore()
		if err != nil {
			return nil, err
		}
	}

	if params.enableLightPush {
		w.mountLightPush()
	}

	for _, addr := range w.ListenAddresses() {
		log.Info("Listening on ", addr)
	}

	return w, nil
}

func (w *WakuNode) Stop() {
	w.subscriptionsMutex.Lock()
	defer w.subscriptionsMutex.Unlock()
	defer w.cancel()

	for _, topic := range w.relay.Topics() {
		for _, sub := range w.subscriptions[topic] {
			sub.Unsubscribe()
		}
	}

	w.subscriptions = nil
}

func (w *WakuNode) Host() host.Host {
	return w.host
}

func (w *WakuNode) ID() string {
	return w.host.ID().Pretty()
}

func (w *WakuNode) ListenAddresses() []string {
	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", w.host.ID().Pretty()))
	var result []string
	for _, addr := range w.host.Addrs() {
		result = append(result, addr.Encapsulate(hostInfo).String())
	}
	return result
}

func (w *WakuNode) Relay() *relay.WakuRelay {
	return w.relay
}

func (w *WakuNode) Filter() *filter.WakuFilter {
	return w.filter
}

func (w *WakuNode) mountRelay(opts ...wakurelay.Option) error {
	var err error
	w.relay, err = relay.NewWakuRelay(w.ctx, w.host, opts...)

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

func (w *WakuNode) startStore() error {
	w.opts.store.Start(w.host)

	w.opts.store.Resume(w.ctx, string(relay.GetTopic(nil)), nil)

	_, err := w.Subscribe(nil)
	if err != nil {
		return err
	}

	return nil
}

func (w *WakuNode) AddStorePeer(address string) (*peer.ID, error) {
	if w.opts.store == nil {
		return nil, errors.New("WakuStore is not set")
	}

	storePeer, err := ma.NewMultiaddr(address)
	if err != nil {
		return nil, err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(storePeer)
	if err != nil {
		return nil, err
	}

	return &info.ID, w.opts.store.AddPeer(info.ID, info.Addrs)
}

// TODO Remove code duplication
func (w *WakuNode) AddFilterPeer(address string) (*peer.ID, error) {
	if w.filter == nil {
		return nil, errors.New("WakuFilter is not set")
	}

	filterPeer, err := ma.NewMultiaddr(address)
	if err != nil {
		return nil, err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(filterPeer)
	if err != nil {
		return nil, err
	}

	return &info.ID, w.filter.AddPeer(info.ID, info.Addrs)
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

	result, err := w.opts.store.Resume(ctx, string(relay.DefaultWakuTopic), peerList)
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
		return nil, errors.New("WakuRelay hasn't been set.")
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
func (node *WakuNode) SubscribeFilter(ctx context.Context, request pb.FilterRequest, ch filter.ContentFilterChan) {
	// Registers for messages that match a specific filter. Triggers the handler whenever a message is received.
	// ContentFilterChan takes MessagePush structs

	// Status: Implemented.

	// Sanity check for well-formed subscribe FilterRequest
	//doAssert(request.subscribe, "invalid subscribe request")

	log.Info("SubscribeFilter, request: ", request)

	var id string

	if node.filter != nil {
		id, err := node.filter.Subscribe(ctx, request)

		if id == "" || err != nil {
			// Failed to subscribe
			log.Error("remote subscription to filter failed", request)
			//waku_node_errors.inc(labelValues = ["subscribe_filter_failure"])
			id = string(protocol.GenerateRequestId())
		}
	}

	// Register handler for filter, whether remote subscription succeeded or not
	node.filters[id] = filter.Filter{ContentFilters: request.ContentFilters, Chan: ch}
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
		for id, _ := range node.filters {
			if id == rId {
				delete(node.filters, id)
				break
			}
		}
	}
}

func (node *WakuNode) Publish(ctx context.Context, message *pb.WakuMessage, topic *relay.Topic) ([]byte, error) {
	if node.relay == nil {
		return nil, errors.New("WakuRelay hasn't been set.")
	}

	if message == nil {
		return nil, errors.New("message can't be null")
	}

	hash, err := node.relay.Publish(ctx, message, topic)
	if err != nil {
		return nil, err
	}

	return hash, nil
}

func (node *WakuNode) LightPush(ctx context.Context, message *pb.WakuMessage, topic *relay.Topic, opts ...lightpush.LightPushOption) ([]byte, error) {
	if node.lightPush == nil {
		return nil, errors.New("WakuLightPush hasn't been set.")
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

func (w *WakuNode) DialPeer(address string) error {
	p, err := ma.NewMultiaddr(address)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(p)
	if err != nil {
		return err
	}

	w.host.Connect(w.ctx, *info)
	return nil
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
	return w.host.Network().ClosePeer(id)
}

func (w *WakuNode) PeerCount() int {
	return len(w.host.Network().Peers())
}
