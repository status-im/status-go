package relay

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	proto "google.golang.org/protobuf/proto"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/hash"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
)

// WakuRelayID_v200 is the current protocol ID used for WakuRelay
const WakuRelayID_v200 = protocol.ID("/vac/waku/relay/2.0.0")

// DefaultWakuTopic is the default pubsub topic used across all Waku protocols
var DefaultWakuTopic string = waku_proto.DefaultPubsubTopic().String()

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

	*waku_proto.CommonService
}

// EvtRelaySubscribed is an event emitted when a new subscription to a pubsub topic is created
type EvtRelaySubscribed struct {
	Topic string
}

// EvtRelayUnsubscribed is an event emitted when a subscription to a pubsub topic is closed
type EvtRelayUnsubscribed struct {
	Topic string
}

type PeerTopicState int

const (
	PEER_JOINED = iota
	PEER_LEFT
)

type EvtPeerTopic struct {
	Topic  string
	PeerID peer.ID
	State  PeerTopicState
}

func msgIDFn(pmsg *pubsub_pb.Message) string {
	return string(hash.SHA256(pmsg.Data))
}

// NewWakuRelay returns a new instance of a WakuRelay struct
func NewWakuRelay(bcaster Broadcaster, minPeersToPublish int, timesource timesource.Timesource, reg prometheus.Registerer, log *zap.Logger, opts ...pubsub.Option) *WakuRelay {
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

	cfg := pubsub.DefaultGossipSubParams()
	cfg.PruneBackoff = time.Minute
	cfg.UnsubscribeBackoff = 5 * time.Second
	cfg.GossipFactor = 0.25
	cfg.D = 6
	cfg.Dlo = 4
	cfg.Dhi = 12
	cfg.Dout = 3
	cfg.Dlazy = 6
	cfg.HeartbeatInterval = time.Second
	cfg.HistoryLength = 6
	cfg.HistoryGossip = 3
	cfg.FanoutTTL = time.Minute

	w.peerScoreParams = &pubsub.PeerScoreParams{
		Topics:        make(map[string]*pubsub.TopicScoreParams),
		DecayInterval: 12 * time.Second, // how often peer scoring is updated
		DecayToZero:   0.01,             // below this we consider the parameter to be zero
		RetainScore:   10 * time.Minute, // remember peer score during x after it disconnects
		// p5: application specific, unset
		AppSpecificScore: func(p peer.ID) float64 {
			return 0
		},
		AppSpecificWeight: 0.0,
		// p6: penalizes peers sharing more than threshold ips
		IPColocationFactorWeight:    -50,
		IPColocationFactorThreshold: 5.0,
		// p7: penalizes bad behaviour (weight and decay)
		BehaviourPenaltyWeight: -10,
		BehaviourPenaltyDecay:  0.986,
	}

	w.peerScoreThresholds = &pubsub.PeerScoreThresholds{
		GossipThreshold:             -100,   // no gossip is sent to peers below this score
		PublishThreshold:            -1000,  // no self-published msgs are sent to peers below this score
		GraylistThreshold:           -10000, // used to trigger disconnections + ignore peer if below this score
		OpportunisticGraftThreshold: 0,      // grafts better peers if the mesh median score drops below this. unset.
	}

	w.topicParams = &pubsub.TopicScoreParams{
		TopicWeight: 1,
		// p1: favours peers already in the mesh
		TimeInMeshWeight:  0.01,
		TimeInMeshQuantum: time.Second,
		TimeInMeshCap:     10.0,
		// p2: rewards fast peers
		FirstMessageDeliveriesWeight: 1.0,
		FirstMessageDeliveriesDecay:  0.5,
		FirstMessageDeliveriesCap:    10.0,
		// p3: penalizes lazy peers. safe low value
		MeshMessageDeliveriesWeight:     0,
		MeshMessageDeliveriesDecay:      0,
		MeshMessageDeliveriesCap:        0,
		MeshMessageDeliveriesThreshold:  0,
		MeshMessageDeliveriesWindow:     0,
		MeshMessageDeliveriesActivation: 0,
		// p3b: tracks history of prunes
		MeshFailurePenaltyWeight: 0,
		MeshFailurePenaltyDecay:  0,
		// p4: penalizes invalid messages. highly penalize peers sending wrong messages
		InvalidMessageDeliveriesWeight: -100.0,
		InvalidMessageDeliveriesDecay:  0.5,
	}

	// default options required by WakuRelay
	w.opts = append([]pubsub.Option{
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
		pubsub.WithNoAuthor(),
		pubsub.WithMessageIdFn(msgIDFn),
		pubsub.WithGossipSubProtocols(
			[]protocol.ID{WakuRelayID_v200, pubsub.GossipSubID_v11, pubsub.GossipSubID_v10, pubsub.FloodSubID},
			func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
				switch feat {
				case pubsub.GossipSubFeatureMesh:
					return proto == pubsub.GossipSubID_v11 || proto == pubsub.GossipSubID_v10 || proto == WakuRelayID_v200
				case pubsub.GossipSubFeaturePX:
					return proto == pubsub.GossipSubID_v11 || proto == WakuRelayID_v200
				default:
					return false
				}
			},
		),
		pubsub.WithGossipSubParams(cfg),
		pubsub.WithFloodPublish(true),
		pubsub.WithSeenMessagesTTL(2 * time.Minute),
		pubsub.WithPeerScore(w.peerScoreParams, w.peerScoreThresholds),
		pubsub.WithPeerScoreInspect(w.peerScoreInspector, 6*time.Second),
	}, opts...)

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
	ps, err := pubsub.NewGossipSub(w.Context(), w.host, w.opts...)
	if err != nil {
		return err
	}
	w.pubsub = ps

	w.emitters.EvtRelaySubscribed, err = w.events.Emitter(new(EvtRelaySubscribed))
	if err != nil {
		return err
	}
	w.emitters.EvtRelayUnsubscribed, err = w.events.Emitter(new(EvtRelayUnsubscribed))
	if err != nil {
		return err
	}

	w.emitters.EvtPeerTopic, err = w.events.Emitter(new(EvtPeerTopic))
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

func (w *WakuRelay) subscribe(topic string) (subs *pubsub.Subscription, err error) {
	sub, ok := w.relaySubs[topic]
	if !ok {
		pubSubTopic, err := w.upsertTopic(topic)
		if err != nil {
			return nil, err
		}

		sub, err = pubSubTopic.Subscribe()
		if err != nil {
			return nil, err
		}

		evtHandler, err := w.addPeerTopicEventListener(pubSubTopic)
		if err != nil {
			return nil, err
		}
		w.topicEvtHanders[topic] = evtHandler
		w.relaySubs[topic] = sub

		err = w.emitters.EvtRelaySubscribed.Emit(EvtRelaySubscribed{topic})
		if err != nil {
			return nil, err
		}

		if w.bcaster != nil {
			w.WaitGroup().Add(1)
			go w.subscribeToTopic(topic, sub)
		}
		w.log.Info("subscribing to topic", zap.String("topic", sub.Topic()))
	}

	return sub, nil
}

// PublishToTopic is used to broadcast a WakuMessage to a pubsub topic
func (w *WakuRelay) PublishToTopic(ctx context.Context, message *pb.WakuMessage, topic string) ([]byte, error) {
	// Publish a `WakuMessage` to a PubSub topic.
	if w.pubsub == nil {
		return nil, errors.New("PubSub hasn't been set")
	}

	if message == nil {
		return nil, errors.New("message can't be null")
	}

	if !w.EnoughPeersToPublishToTopic(topic) {
		return nil, errors.New("not enough peers to publish")
	}

	pubSubTopic, err := w.upsertTopic(topic)

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

	hash := message.Hash(topic)

	w.log.Debug("waku.relay published", zap.String("pubsubTopic", topic), logging.HexString("hash", hash), zap.Int64("publishTime", w.timesource.Now().UnixNano()), zap.Int("payloadSizeBytes", len(message.Payload)))

	return hash, nil
}

// Publish is used to broadcast a WakuMessage to the default waku pubsub topic
func (w *WakuRelay) Publish(ctx context.Context, message *pb.WakuMessage) ([]byte, error) {
	return w.PublishToTopic(ctx, message, DefaultWakuTopic)
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

// SubscribeToTopic returns a Subscription to receive messages from a pubsub topic
func (w *WakuRelay) SubscribeToTopic(ctx context.Context, topic string) (*Subscription, error) {
	_, err := w.subscribe(topic)
	if err != nil {
		return nil, err
	}

	// Create client subscription
	subscription := NoopSubscription()
	if w.bcaster != nil {
		subscription = w.bcaster.Register(topic, 1024)
	}
	go func() {
		<-ctx.Done()
		subscription.Unsubscribe()
	}()
	return &subscription, nil
}

// Subscribe returns a Subscription to receive messages from the default waku pubsub topic
func (w *WakuRelay) Subscribe(ctx context.Context) (*Subscription, error) {
	return w.SubscribeToTopic(ctx, DefaultWakuTopic)
}

// Unsubscribe closes a subscription to a pubsub topic
func (w *WakuRelay) Unsubscribe(ctx context.Context, topic string) error {
	w.topicsMutex.Lock()
	defer w.topicsMutex.Unlock()

	sub, ok := w.relaySubs[topic]
	if !ok {
		return fmt.Errorf("not subscribed to topic")
	}
	w.log.Info("unsubscribing from topic", zap.String("topic", sub.Topic()))

	w.relaySubs[topic].Cancel()
	delete(w.relaySubs, topic)

	evtHandler, ok := w.topicEvtHanders[topic]
	if ok {
		evtHandler.Cancel()
		delete(w.topicEvtHanders, topic)
	}

	err := w.wakuRelayTopics[topic].Close()
	if err != nil {
		return err
	}
	delete(w.wakuRelayTopics, topic)

	w.RemoveTopicValidator(topic)

	err = w.emitters.EvtRelayUnsubscribed.Emit(EvtRelayUnsubscribed{topic})
	if err != nil {
		return err
	}

	return nil
}

func (w *WakuRelay) nextMessage(ctx context.Context, sub *pubsub.Subscription) <-chan *pubsub.Message {
	msgChannel := make(chan *pubsub.Message, 1024)
	go func() {
		defer close(msgChannel)
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					w.log.Error("getting message from subscription", zap.Error(err))
				}
				sub.Cancel()
				return
			}
			msgChannel <- msg
		}
	}()
	return msgChannel
}

func (w *WakuRelay) subscribeToTopic(pubsubTopic string, sub *pubsub.Subscription) {
	defer w.WaitGroup().Done()

	subChannel := w.nextMessage(w.Context(), sub)
	for {
		select {
		case <-w.Context().Done():
			return
			// TODO: if there are no more relay subscriptions, close the pubsub subscription
		case msg, ok := <-subChannel:
			if !ok {
				return
			}
			wakuMessage := &pb.WakuMessage{}
			if err := proto.Unmarshal(msg.Data, wakuMessage); err != nil {
				w.log.Error("decoding message", zap.Error(err))
				return
			}

			envelope := waku_proto.NewEnvelope(wakuMessage, w.timesource.Now().UnixNano(), pubsubTopic)

			w.metrics.RecordMessage(envelope)

			if w.bcaster != nil {
				w.bcaster.Submit(envelope)
			}
		}
	}

}

// Params returns the gossipsub configuration parameters used by WakuRelay
func (w *WakuRelay) Params() pubsub.GossipSubParams {
	return w.params
}

// Events returns the event bus on which WakuRelay events will be emitted
func (w *WakuRelay) Events() event.Bus {
	return w.events
}

func (w *WakuRelay) addPeerTopicEventListener(topic *pubsub.Topic) (*pubsub.TopicEventHandler, error) {
	handler, err := topic.EventHandler()
	if err != nil {
		return nil, err
	}
	w.WaitGroup().Add(1)
	go w.topicEventPoll(topic.String(), handler)
	return handler, nil
}

func (w *WakuRelay) topicEventPoll(topic string, handler *pubsub.TopicEventHandler) {
	defer w.WaitGroup().Done()
	for {
		evt, err := handler.NextPeerEvent(w.Context())
		if err != nil {
			if err == context.Canceled {
				break
			}
			w.log.Error("failed to get next peer event", zap.String("topic", topic), zap.Error(err))
			continue
		}
		if evt.Peer.Validate() != nil { //Empty peerEvent is returned when context passed in done.
			break
		}
		if evt.Type == pubsub.PeerJoin {
			w.log.Debug("received a PeerJoin event", zap.String("topic", topic), logging.HostID("peerID", evt.Peer))
			err = w.emitters.EvtPeerTopic.Emit(EvtPeerTopic{Topic: topic, PeerID: evt.Peer, State: PEER_JOINED})
			if err != nil {
				w.log.Error("failed to emit PeerJoin", zap.String("topic", topic), zap.Error(err))
			}
		} else if evt.Type == pubsub.PeerLeave {
			w.log.Debug("received a PeerLeave event", zap.String("topic", topic), logging.HostID("peerID", evt.Peer))
			err = w.emitters.EvtPeerTopic.Emit(EvtPeerTopic{Topic: topic, PeerID: evt.Peer, State: PEER_LEFT})
			if err != nil {
				w.log.Error("failed to emit PeerLeave", zap.String("topic", topic), zap.Error(err))
			}
		} else {
			w.log.Error("unknown event type received", zap.String("topic", topic),
				zap.Int("eventType", int(evt.Type)))
		}
	}
}
