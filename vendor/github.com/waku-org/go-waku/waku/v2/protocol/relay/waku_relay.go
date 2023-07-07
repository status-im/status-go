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
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
	proto "google.golang.org/protobuf/proto"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/hash"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
)

const WakuRelayID_v200 = protocol.ID("/vac/waku/relay/2.0.0")

var DefaultWakuTopic string = waku_proto.DefaultPubsubTopic().String()

type WakuRelay struct {
	host                host.Host
	opts                []pubsub.Option
	pubsub              *pubsub.PubSub
	params              pubsub.GossipSubParams
	peerScoreParams     *pubsub.PeerScoreParams
	peerScoreThresholds *pubsub.PeerScoreThresholds
	topicParams         *pubsub.TopicScoreParams
	timesource          timesource.Timesource

	log *zap.Logger

	bcaster Broadcaster

	minPeersToPublish int

	// TODO: convert to concurrent maps
	topicsMutex     sync.Mutex
	wakuRelayTopics map[string]*pubsub.Topic
	relaySubs       map[string]*pubsub.Subscription

	events   event.Bus
	emitters struct {
		EvtRelaySubscribed   event.Emitter
		EvtRelayUnsubscribed event.Emitter
	}

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// EvtRelaySubscribed is an event emitted when a new subscription to a pubsub topic is created
type EvtRelaySubscribed struct {
	Topic string
}

// EvtRelayUnsubscribed is an event emitted when a subscription to a pubsub topic is closed
type EvtRelayUnsubscribed struct {
	Topic string
}

func msgIdFn(pmsg *pubsub_pb.Message) string {
	return string(hash.SHA256(pmsg.Data))
}

// NewWakuRelay returns a new instance of a WakuRelay struct
func NewWakuRelay(bcaster Broadcaster, minPeersToPublish int, timesource timesource.Timesource, log *zap.Logger, opts ...pubsub.Option) *WakuRelay {
	w := new(WakuRelay)
	w.timesource = timesource
	w.wakuRelayTopics = make(map[string]*pubsub.Topic)
	w.relaySubs = make(map[string]*pubsub.Subscription)
	w.bcaster = bcaster
	w.minPeersToPublish = minPeersToPublish
	w.wg = sync.WaitGroup{}
	w.log = log.Named("relay")
	w.events = eventbus.NewBus()

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
		pubsub.WithMessageIdFn(msgIdFn),
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
		pubsub.WithDefaultValidator(func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool {
			msg := new(pb.WakuMessage)
			err := proto.Unmarshal(message.Data, msg)
			return err == nil
		}),
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

// Sets the host to be able to mount or consume a protocol
func (w *WakuRelay) SetHost(h host.Host) {
	w.host = h
}

func (w *WakuRelay) Start(ctx context.Context) error {
	w.wg.Wait()
	ctx, cancel := context.WithCancel(ctx)
	w.ctx = ctx // TODO: create worker for creating subscriptions instead of storing context
	w.cancel = cancel

	ps, err := pubsub.NewGossipSub(ctx, w.host, w.opts...)
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

	w.log.Info("Relay protocol started")
	return nil
}

// PubSub returns the implementation of the pubsub system
func (w *WakuRelay) PubSub() *pubsub.PubSub {
	return w.pubsub
}

// Topics returns a list of all the pubsub topics currently subscribed to
func (w *WakuRelay) Topics() []string {
	defer w.topicsMutex.Unlock()
	w.topicsMutex.Lock()

	var result []string
	for topic := range w.relaySubs {
		result = append(result, topic)
	}
	return result
}

func (w *WakuRelay) IsSubscribed(topic string) bool {
	defer w.topicsMutex.Unlock()
	w.topicsMutex.Lock()
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

		w.relaySubs[topic] = sub

		err = w.emitters.EvtRelaySubscribed.Emit(EvtRelaySubscribed{topic})
		if err != nil {
			return nil, err
		}

		if w.bcaster != nil {
			w.wg.Add(1)
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
	if w.cancel == nil {
		return // Not started
	}

	w.host.RemoveStreamHandler(WakuRelayID_v200)
	w.emitters.EvtRelaySubscribed.Close()
	w.emitters.EvtRelayUnsubscribed.Close()
	w.cancel()
	w.wg.Wait()
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

// SubscribeToTopic returns a Subscription to receive messages from the default waku pubsub topic
func (w *WakuRelay) Subscribe(ctx context.Context) (*Subscription, error) {
	return w.SubscribeToTopic(ctx, DefaultWakuTopic)
}

// Unsubscribe closes a subscription to a pubsub topic
func (w *WakuRelay) Unsubscribe(ctx context.Context, topic string) error {
	sub, ok := w.relaySubs[topic]
	if !ok {
		return fmt.Errorf("not subscribed to topic")
	}
	w.log.Info("unsubscribing from topic", zap.String("topic", sub.Topic()))

	w.relaySubs[topic].Cancel()
	delete(w.relaySubs, topic)

	err := w.wakuRelayTopics[topic].Close()
	if err != nil {
		return err
	}
	delete(w.wakuRelayTopics, topic)

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
	defer w.wg.Done()

	ctx, err := tag.New(w.ctx, tag.Insert(metrics.KeyType, "relay"))
	if err != nil {
		w.log.Error("creating tag map", zap.Error(err))
		return
	}

	subChannel := w.nextMessage(w.ctx, sub)
	for {
		select {
		case <-ctx.Done():
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

			payloadSizeInBytes := len(wakuMessage.Payload)
			payloadSizeInKb := payloadSizeInBytes / 1000
			stats.Record(ctx, metrics.Messages.M(1), metrics.MessageSize.M(int64(payloadSizeInKb)))

			envelope := waku_proto.NewEnvelope(wakuMessage, w.timesource.Now().UnixNano(), pubsubTopic)
			w.log.Debug("waku.relay received", zap.String("pubsubTopic", pubsubTopic), logging.HexString("hash", envelope.Hash()), zap.Int64("receivedTime", envelope.Index().ReceiverTime), zap.Int("payloadSizeBytes", payloadSizeInBytes))

			if w.bcaster != nil {
				w.bcaster.Submit(envelope)
			}
		}
	}

}

func (w *WakuRelay) Params() pubsub.GossipSubParams {
	return w.params
}

// Events returns the event bus on which WakuRelay events will be emitted
func (w *WakuRelay) Events() event.Bus {
	return w.events
}
