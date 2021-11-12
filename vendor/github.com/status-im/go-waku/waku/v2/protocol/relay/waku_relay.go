package relay

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"

	proto "github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	v2 "github.com/status-im/go-waku/waku/v2"
	"github.com/status-im/go-waku/waku/v2/metrics"
	waku_proto "github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

var log = logging.Logger("wakurelay")

type Topic string

const WakuRelayID_v200 = protocol.ID("/vac/waku/relay/2.0.0")

var DefaultWakuTopic Topic = Topic(waku_proto.DefaultPubsubTopic().String())

type WakuRelay struct {
	host   host.Host
	pubsub *pubsub.PubSub

	bcaster v2.Broadcaster

	// TODO: convert to concurrent maps
	topics          map[Topic]struct{}
	topicsMutex     sync.Mutex
	wakuRelayTopics map[Topic]*pubsub.Topic
	relaySubs       map[Topic]*pubsub.Subscription

	// TODO: convert to concurrent maps
	subscriptions      map[Topic][]*Subscription
	subscriptionsMutex sync.Mutex
}

// Once https://github.com/status-im/nim-waku/issues/420 is fixed, implement a custom messageIdFn
func msgIdFn(pmsg *pubsub_pb.Message) string {
	hash := sha256.Sum256(pmsg.Data)
	return string(hash[:])
}

func NewWakuRelay(ctx context.Context, h host.Host, bcaster v2.Broadcaster, opts ...pubsub.Option) (*WakuRelay, error) {
	w := new(WakuRelay)
	w.host = h
	w.topics = make(map[Topic]struct{})
	w.wakuRelayTopics = make(map[Topic]*pubsub.Topic)
	w.relaySubs = make(map[Topic]*pubsub.Subscription)
	w.subscriptions = make(map[Topic][]*Subscription)
	w.bcaster = bcaster

	// default options required by WakuRelay
	opts = append(opts, pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign))
	opts = append(opts, pubsub.WithNoAuthor())
	opts = append(opts, pubsub.WithMessageIdFn(msgIdFn))

	opts = append(opts, pubsub.WithGossipSubProtocols(
		[]protocol.ID{pubsub.GossipSubID_v11, pubsub.GossipSubID_v10, pubsub.FloodSubID, WakuRelayID_v200},
		func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
			switch feat {
			case pubsub.GossipSubFeatureMesh:
				return proto == pubsub.GossipSubID_v11 || proto == pubsub.GossipSubID_v10
			case pubsub.GossipSubFeaturePX:
				return proto == pubsub.GossipSubID_v11
			default:
				return false
			}
		},
	))

	ps, err := pubsub.NewGossipSub(ctx, h, opts...)
	if err != nil {
		return nil, err
	}
	w.pubsub = ps

	log.Info("Relay protocol started")

	return w, nil
}

func (w *WakuRelay) PubSub() *pubsub.PubSub {
	return w.pubsub
}

func (w *WakuRelay) Topics() []Topic {
	defer w.topicsMutex.Unlock()
	w.topicsMutex.Lock()

	var result []Topic
	for topic := range w.topics {
		result = append(result, topic)
	}
	return result
}

func (w *WakuRelay) SetPubSub(pubSub *pubsub.PubSub) {
	w.pubsub = pubSub
}

func (w *WakuRelay) upsertTopic(topic Topic) (*pubsub.Topic, error) {
	defer w.topicsMutex.Unlock()
	w.topicsMutex.Lock()

	w.topics[topic] = struct{}{}
	pubSubTopic, ok := w.wakuRelayTopics[topic]
	if !ok { // Joins topic if node hasn't joined yet
		newTopic, err := w.pubsub.Join(string(topic))
		if err != nil {
			return nil, err
		}
		w.wakuRelayTopics[topic] = newTopic
		pubSubTopic = newTopic
	}
	return pubSubTopic, nil
}

func (w *WakuRelay) subscribe(topic Topic) (subs *pubsub.Subscription, err error) {
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

		log.Info("Subscribing to topic ", topic)
	}

	return sub, nil
}

func (w *WakuRelay) Publish(ctx context.Context, message *pb.WakuMessage, topic *Topic) ([]byte, error) {
	// Publish a `WakuMessage` to a PubSub topic.
	if w.pubsub == nil {
		return nil, errors.New("PubSub hasn't been set")
	}

	if message == nil {
		return nil, errors.New("message can't be null")
	}

	pubSubTopic, err := w.upsertTopic(GetTopic(topic))

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

	hash := pb.Hash(out)

	return hash, nil
}

func GetTopic(topic *Topic) Topic {
	var t Topic = DefaultWakuTopic
	if topic != nil {
		t = *topic
	}
	return t
}

func (w *WakuRelay) Stop() {
	w.host.RemoveStreamHandler(WakuRelayID_v200)
	w.subscriptionsMutex.Lock()
	defer w.subscriptionsMutex.Unlock()

	for _, topic := range w.Topics() {
		for _, sub := range w.subscriptions[topic] {
			sub.Unsubscribe()
		}
	}
	w.subscriptions = nil
}

func (w *WakuRelay) Subscribe(ctx context.Context, topic *Topic) (*Subscription, error) {
	// Subscribes to a PubSub topic.
	// NOTE The data field SHOULD be decoded as a WakuMessage.
	t := GetTopic(topic)
	sub, err := w.subscribe(t)

	if err != nil {
		return nil, err
	}

	// Create client subscription
	subscription := new(Subscription)
	subscription.closed = false
	subscription.C = make(chan *waku_proto.Envelope, 1024) // To avoid blocking
	subscription.quit = make(chan struct{})

	w.subscriptionsMutex.Lock()
	defer w.subscriptionsMutex.Unlock()

	w.subscriptions[t] = append(w.subscriptions[t], subscription)

	if w.bcaster != nil {
		w.bcaster.Register(subscription.C)
	}

	go w.subscribeToTopic(t, subscription, sub)

	return subscription, nil
}

func (w *WakuRelay) Unsubscribe(ctx context.Context, topic Topic) error {
	if _, ok := w.topics[topic]; !ok {
		return fmt.Errorf("topics %s is not subscribed", (string)(topic))
	}
	log.Info("Unsubscribing from topic ", topic)
	delete(w.topics, topic)

	for _, sub := range w.subscriptions[topic] {
		sub.Unsubscribe()
	}

	w.relaySubs[topic].Cancel()
	delete(w.relaySubs, topic)

	err := w.wakuRelayTopics[topic].Close()
	if err != nil {
		return err
	}
	delete(w.wakuRelayTopics, topic)

	return nil
}

func (w *WakuRelay) nextMessage(ctx context.Context, sub *pubsub.Subscription) <-chan *pubsub.Message {
	msgChannel := make(chan *pubsub.Message, 1024)
	go func(msgChannel chan *pubsub.Message) {
		defer func() {
			if r := recover(); r != nil {
				log.Debug("recovered msgChannel")
			}
		}()

		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				log.Error(fmt.Errorf("subscription failed: %w", err))
				sub.Cancel()
				close(msgChannel)
				for _, subscription := range w.subscriptions[Topic(sub.Topic())] {
					subscription.Unsubscribe()
				}
			}

			msgChannel <- msg
		}
	}(msgChannel)
	return msgChannel
}

func (w *WakuRelay) subscribeToTopic(t Topic, subscription *Subscription, sub *pubsub.Subscription) {
	ctx, err := tag.New(context.Background(), tag.Insert(metrics.KeyType, "relay"))
	if err != nil {
		log.Error(err)
		return
	}

	subChannel := w.nextMessage(ctx, sub)

	for {
		select {
		case <-subscription.quit:
			if w.bcaster != nil {
				w.bcaster.Unregister(subscription.C) // Remove from broadcast list
			}
			// TODO: if there are no more relay subscriptions, close the pubsub subscription
		case msg := <-subChannel:
			if msg == nil {
				return
			}
			stats.Record(ctx, metrics.Messages.M(1))
			wakuMessage := &pb.WakuMessage{}
			if err := proto.Unmarshal(msg.Data, wakuMessage); err != nil {
				log.Error("could not decode message", err)
				return
			}

			envelope := waku_proto.NewEnvelope(wakuMessage, string(t))

			if w.bcaster != nil {
				w.bcaster.Submit(envelope)
			}
		}
	}
}
