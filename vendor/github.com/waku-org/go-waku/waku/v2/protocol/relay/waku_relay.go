package relay

import (
	"context"
	"errors"
	"sync"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	proto "google.golang.org/protobuf/proto"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/waku-org/go-waku/logging"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
)

// WakuRelayID_v200 is the current protocol ID used for WakuRelay
const WakuRelayID_v200 = protocol.ID("/vac/waku/relay/2.0.0")

// DefaultWakuTopic is the default pubsub topic used across all Waku protocols
var DefaultWakuTopic string = waku_proto.DefaultPubsubTopic{}.String()

// WakuRelay is the implementation of the Waku Relay protocol
type WakuRelay struct {
	host                host.Host
	opts                []pubsub.Option
	pubsub              *pubsub.PubSub
	params              pubsub.GossipSubParams
	peerScoreParams     *pubsub.PeerScoreParams
	peerScoreThresholds *pubsub.PeerScoreThresholds
	topicParams         *pubsub.TopicScoreParams
	timesource          timesource.Timesource
	metrics             Metrics

	log *zap.Logger

	bcaster Broadcaster

	minPeersToPublish int

	topicValidatorMutex    sync.RWMutex
	topicValidators        map[string][]validatorFn
	defaultTopicValidators []validatorFn

	// TODO: convert to concurrent maps
	topicsMutex     sync.RWMutex
	wakuRelayTopics map[string]*pubsub.Topic
	relaySubs       map[string]*pubsub.Subscription
	topicEvtHanders map[string]*pubsub.TopicEventHandler

	events   event.Bus
	emitters struct {
		EvtRelaySubscribed   event.Emitter
		EvtRelayUnsubscribed event.Emitter
		EvtPeerTopic         event.Emitter
	}
	contentSubs map[string]map[int]*Subscription
	*waku_proto.CommonService
}

// NewWakuRelay returns a new instance of a WakuRelay struct
func NewWakuRelay(bcaster Broadcaster, minPeersToPublish int, timesource timesource.Timesource,
	reg prometheus.Registerer, log *zap.Logger, opts ...pubsub.Option) *WakuRelay {
	w := new(WakuRelay)
	w.timesource = timesource
	w.wakuRelayTopics = make(map[string]*pubsub.Topic)
	w.relaySubs = make(map[string]*pubsub.Subscription)
	w.topicEvtHanders = make(map[string]*pubsub.TopicEventHandler)
	w.topicValidators = make(map[string][]validatorFn)
	w.bcaster = bcaster
	w.minPeersToPublish = minPeersToPublish
	w.CommonService = waku_proto.NewCommonService()
	w.log = log.Named("relay")
	w.events = eventbus.NewBus()
	w.metrics = newMetrics(reg, w.log)

	// default options required by WakuRelay
	w.opts = append(w.defaultPubsubOptions(), opts...)
	w.contentSubs = make(map[string]map[int]*Subscription)
	return w
}

func (w *WakuRelay) peerScoreInspector(peerScoresSnapshots map[peer.ID]*pubsub.PeerScoreSnapshot) {
	if w.host == nil {
		return
	}

	for pid, snap := range peerScoresSnapshots {
		if snap.Score < w.peerScoreThresholds.GraylistThreshold {
			// Disconnect bad peers
			err := w.host.Network().ClosePeer(pid)
			if err != nil {
				w.log.Error("could not disconnect peer", logging.HostID("peer", pid), zap.Error(err))
			}
		}
	}
}

// SetHost sets the host to be able to mount or consume a protocol
func (w *WakuRelay) SetHost(h host.Host) {
	w.host = h
}

// Start initiates the WakuRelay protocol
func (w *WakuRelay) Start(ctx context.Context) error {
	return w.CommonService.Start(ctx, w.start)
}

func (w *WakuRelay) start() error {
	if w.bcaster == nil {
		return errors.New("broadcaster not specified for relay")
	}
	ps, err := pubsub.NewGossipSub(w.Context(), w.host, w.opts...)
	if err != nil {
		return err
	}
	w.pubsub = ps

	err = w.CreateEventEmitters()
	if err != nil {
		return err
	}

	w.log.Info("Relay protocol started")
	return nil
}

// PubSub returns the implementation of the pubsub system
func (w *WakuRelay) PubSub() *pubsub.PubSub {
	return w.pubsub
}

// Topics returns a list of all the pubsub topics currently subscribed to
func (w *WakuRelay) Topics() []string {
	defer w.topicsMutex.RUnlock()
	w.topicsMutex.RLock()

	var result []string
	for topic := range w.relaySubs {
		result = append(result, topic)
	}
	return result
}

// IsSubscribed indicates whether the node is subscribed to a pubsub topic or not
func (w *WakuRelay) IsSubscribed(topic string) bool {
	w.topicsMutex.RLock()
	defer w.topicsMutex.RUnlock()
	_, ok := w.relaySubs[topic]
	return ok
}

// SetPubSub is used to set an implementation of the pubsub system
func (w *WakuRelay) SetPubSub(pubSub *pubsub.PubSub) {
	w.pubsub = pubSub
}

func (w *WakuRelay) upsertTopic(topic string) (*pubsub.Topic, error) {
	w.topicsMutex.Lock()
	defer w.topicsMutex.Unlock()

	pubSubTopic, ok := w.wakuRelayTopics[topic]
	if !ok { // Joins topic if node hasn't joined yet
		err := w.pubsub.RegisterTopicValidator(topic, w.topicValidator(topic))
		if err != nil {
			return nil, err
		}

		newTopic, err := w.pubsub.Join(string(topic))
		if err != nil {
			return nil, err
		}

		err = newTopic.SetScoreParams(w.topicParams)
		if err != nil {
			return nil, err
		}

		w.wakuRelayTopics[topic] = newTopic
		pubSubTopic = newTopic
	}
	return pubSubTopic, nil
}

func (w *WakuRelay) subscribeToPubsubTopic(topic string) (subs *pubsub.Subscription, err error) {
	sub, ok := w.relaySubs[topic]
	if !ok {
		pubSubTopic, err := w.upsertTopic(topic)
		if err != nil {
			return nil, err
		}

		sub, err = pubSubTopic.Subscribe(pubsub.WithBufferSize(1024))
		if err != nil {
			return nil, err
		}

		w.WaitGroup().Add(1)
		go w.pubsubTopicMsgHandler(topic, sub)

		evtHandler, err := w.addPeerTopicEventListener(pubSubTopic)
		if err != nil {
			return nil, err
		}
		w.topicEvtHanders[topic] = evtHandler
		w.relaySubs[topic] = sub

		err = w.emitters.EvtRelaySubscribed.Emit(EvtRelaySubscribed{topic, pubSubTopic})
		if err != nil {
			return nil, err
		}

		w.log.Info("subscribing to topic", zap.String("topic", sub.Topic()))
	}

	return sub, nil
}

// PublishToTopic is used to broadcast a WakuMessage to a pubsub topic. The pubsubTopic is derived from contentTopic
// specified in the message via autosharding. To publish to a specific pubsubTopic, the `WithPubSubTopic` option should
// be provided
func (w *WakuRelay) Publish(ctx context.Context, message *pb.WakuMessage, opts ...PublishOption) ([]byte, error) {
	// Publish a `WakuMessage` to a PubSub topic.
	if w.pubsub == nil {
		return nil, errors.New("PubSub hasn't been set")
	}

	if message == nil {
		return nil, errors.New("message can't be null")
	}

	err := message.Validate()
	if err != nil {
		return nil, err
	}

	params := new(publishParameters)
	for _, opt := range opts {
		opt(params)
	}

	if params.pubsubTopic == "" {
		params.pubsubTopic, err = waku_proto.GetPubSubTopicFromContentTopic(message.ContentTopic)
		if err != nil {
			return nil, err
		}
	}

	if !w.EnoughPeersToPublishToTopic(params.pubsubTopic) {
		return nil, errors.New("not enough peers to publish")
	}

	pubSubTopic, err := w.upsertTopic(params.pubsubTopic)
	if err != nil {
		return nil, err
	}

	out, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	err = pubSubTopic.Publish(ctx, out)
	if err != nil {
		return nil, err
	}

	hash := message.Hash(params.pubsubTopic)

	w.log.Debug("waku.relay published", zap.String("pubsubTopic", params.pubsubTopic), logging.HexString("hash", hash), zap.Int64("publishTime", w.timesource.Now().UnixNano()), zap.Int("payloadSizeBytes", len(message.Payload)))

	return hash, nil
}

func (w *WakuRelay) GetSubscription(contentTopic string) (*Subscription, error) {
	pubSubTopic, err := waku_proto.GetPubSubTopicFromContentTopic(contentTopic)
	if err != nil {
		return nil, err
	}
	contentFilter := waku_proto.NewContentFilter(pubSubTopic, contentTopic)
	cSubs := w.contentSubs[pubSubTopic]
	for _, sub := range cSubs {
		if sub.contentFilter.Equals(contentFilter) {
			return sub, nil
		}
	}
	return nil, errors.New("no subscription found for content topic")
}

// Stop unmounts the relay protocol and stops all subscriptions
func (w *WakuRelay) Stop() {
	w.CommonService.Stop(func() {
		w.host.RemoveStreamHandler(WakuRelayID_v200)
		w.emitters.EvtRelaySubscribed.Close()
		w.emitters.EvtRelayUnsubscribed.Close()
	})
}

// EnoughPeersToPublish returns whether there are enough peers connected in the default waku pubsub topic
func (w *WakuRelay) EnoughPeersToPublish() bool {
	return w.EnoughPeersToPublishToTopic(DefaultWakuTopic)
}

// EnoughPeersToPublish returns whether there are enough peers connected in a pubsub topic
func (w *WakuRelay) EnoughPeersToPublishToTopic(topic string) bool {
	return len(w.PubSub().ListPeers(topic)) >= w.minPeersToPublish
}

// subscribe returns list of Subscription to receive messages based on content filter
func (w *WakuRelay) subscribe(ctx context.Context, contentFilter waku_proto.ContentFilter, opts ...RelaySubscribeOption) ([]*Subscription, error) {

	var subscriptions []*Subscription
	pubSubTopicMap, err := waku_proto.ContentFilterToPubSubTopicMap(contentFilter)
	if err != nil {
		return nil, err
	}
	params := new(RelaySubscribeParameters)

	var optList []RelaySubscribeOption
	optList = append(optList, opts...)
	for _, opt := range optList {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	for pubSubTopic, cTopics := range pubSubTopicMap {
		w.log.Info("subscribing to", zap.String("pubsubTopic", pubSubTopic), zap.Strings("contenTopics", cTopics))
		var cFilter waku_proto.ContentFilter
		cFilter.PubsubTopic = pubSubTopic
		cFilter.ContentTopics = waku_proto.NewContentTopicSet(cTopics...)

		//Check if gossipsub subscription already exists for pubSubTopic
		if !w.IsSubscribed(pubSubTopic) {
			_, err := w.subscribeToPubsubTopic(cFilter.PubsubTopic)
			if err != nil {
				//TODO: Handle partial errors.
				return nil, err
			}
		}

		subscription := w.bcaster.Register(cFilter, WithBufferSize(DefaultRelaySubscriptionBufferSize),
			WithConsumerOption(params.dontConsume))

		// Create Content subscription
		w.topicsMutex.RLock()
		if _, ok := w.contentSubs[pubSubTopic]; !ok {
			w.contentSubs[pubSubTopic] = map[int]*Subscription{}
		}
		w.contentSubs[pubSubTopic][subscription.ID] = subscription

		w.topicsMutex.RUnlock()
		subscriptions = append(subscriptions, subscription)
		go func() {
			<-ctx.Done()
			subscription.Unsubscribe()
		}()
	}

	return subscriptions, nil
}

// Subscribe returns a Subscription to receive messages as per contentFilter
// contentFilter can contain pubSubTopic and contentTopics or only contentTopics(in case of autosharding)
func (w *WakuRelay) Subscribe(ctx context.Context, contentFilter waku_proto.ContentFilter, opts ...RelaySubscribeOption) ([]*Subscription, error) {
	return w.subscribe(ctx, contentFilter, opts...)
}

// Unsubscribe closes a subscription to a pubsub topic
func (w *WakuRelay) Unsubscribe(ctx context.Context, contentFilter waku_proto.ContentFilter) error {

	pubSubTopicMap, err := waku_proto.ContentFilterToPubSubTopicMap(contentFilter)
	if err != nil {
		return err
	}

	w.topicsMutex.Lock()
	defer w.topicsMutex.Unlock()

	for pubSubTopic, cTopics := range pubSubTopicMap {
		cfTemp := waku_proto.NewContentFilter(pubSubTopic, cTopics...)
		pubsubUnsubscribe := false
		sub, ok := w.relaySubs[pubSubTopic]
		if !ok {
			return errors.New("not subscribed to topic")
		}
		cSubs := w.contentSubs[pubSubTopic]
		if cSubs != nil {
			//Remove relevant subscription
			for subID, sub := range cSubs {
				if sub.contentFilter.Equals(cfTemp) {
					sub.Unsubscribe()
					delete(cSubs, subID)
				}
			}
			if len(cSubs) == 0 {
				pubsubUnsubscribe = true
			}
		} else {
			//Should not land here ideally
			w.log.Error("pubsub subscriptions exists, but contentSubscription doesn't for contentFilter",
				zap.String("pubsubTopic", pubSubTopic), zap.Strings("contentTopics", cTopics))

			return errors.New("unexpected error in unsubscribe")
		}

		if pubsubUnsubscribe {
			err = w.unsubscribeFromPubsubTopic(sub)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// unsubscribeFromPubsubTopic unsubscribes subscription from underlying pubsub.
// Note: caller has to acquire topicsMutex in order to avoid race conditions
func (w *WakuRelay) unsubscribeFromPubsubTopic(sub *pubsub.Subscription) error {

	pubSubTopic := sub.Topic()
	w.log.Info("unsubscribing from topic", zap.String("topic", pubSubTopic))

	sub.Cancel()
	delete(w.relaySubs, pubSubTopic)

	w.bcaster.UnRegister(pubSubTopic)

	delete(w.contentSubs, pubSubTopic)

	evtHandler, ok := w.topicEvtHanders[pubSubTopic]
	if ok {
		evtHandler.Cancel()
		delete(w.topicEvtHanders, pubSubTopic)
	}

	err := w.wakuRelayTopics[pubSubTopic].Close()
	if err != nil {
		return err
	}
	delete(w.wakuRelayTopics, pubSubTopic)

	w.RemoveTopicValidator(pubSubTopic)

	err = w.emitters.EvtRelayUnsubscribed.Emit(EvtRelayUnsubscribed{pubSubTopic})
	if err != nil {
		return err
	}
	return nil
}

func (w *WakuRelay) pubsubTopicMsgHandler(pubsubTopic string, sub *pubsub.Subscription) {
	defer w.WaitGroup().Done()

	for {
		msg, err := sub.Next(w.Context())
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				w.log.Error("getting message from subscription", zap.Error(err))
			}
			sub.Cancel()
			return
		}

		wakuMessage, err := pb.Unmarshal(msg.Data)
		if err != nil {
			w.log.Error("decoding message", zap.Error(err))
			return
		}

		envelope := waku_proto.NewEnvelope(wakuMessage, w.timesource.Now().UnixNano(), pubsubTopic)
		w.metrics.RecordMessage(envelope)

		w.bcaster.Submit(envelope)
	}

}

// Params returns the gossipsub configuration parameters used by WakuRelay
func (w *WakuRelay) Params() pubsub.GossipSubParams {
	return w.params
}
