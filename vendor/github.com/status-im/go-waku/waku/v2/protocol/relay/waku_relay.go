package relay

import (
	"context"
	"errors"
	"sync"

	proto "github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	wakurelay "github.com/status-im/go-wakurelay-pubsub"
)

var log = logging.Logger("wakurelay")

type Topic string

const DefaultWakuTopic Topic = "/waku/2/default-waku/proto"

type WakuRelay struct {
	host   host.Host
	pubsub *wakurelay.PubSub

	topics          map[Topic]bool
	topicsMutex     sync.Mutex
	wakuRelayTopics map[Topic]*wakurelay.Topic
	relaySubs       map[Topic]*wakurelay.Subscription
}

func NewWakuRelay(ctx context.Context, h host.Host, opts ...wakurelay.Option) (*WakuRelay, error) {
	w := new(WakuRelay)
	w.host = h
	w.topics = make(map[Topic]bool)
	w.wakuRelayTopics = make(map[Topic]*wakurelay.Topic)
	w.relaySubs = make(map[Topic]*wakurelay.Subscription)

	ps, err := wakurelay.NewWakuRelaySub(ctx, h, opts...)
	if err != nil {
		return nil, err
	}
	w.pubsub = ps

	log.Info("Relay protocol started")

	return w, nil
}

func (w *WakuRelay) PubSub() *wakurelay.PubSub {
	return w.pubsub
}

func (w *WakuRelay) Topics() []Topic {
	defer w.topicsMutex.Unlock()
	w.topicsMutex.Lock()

	var result []Topic
	for topic, _ := range w.topics {
		result = append(result, topic)
	}
	return result
}

func (w *WakuRelay) SetPubSub(pubSub *wakurelay.PubSub) {
	w.pubsub = pubSub
}

func (w *WakuRelay) upsertTopic(topic Topic) (*wakurelay.Topic, error) {
	defer w.topicsMutex.Unlock()
	w.topicsMutex.Lock()

	w.topics[topic] = true
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

func (w *WakuRelay) Subscribe(topic Topic) (subs *wakurelay.Subscription, isNew bool, err error) {

	sub, ok := w.relaySubs[topic]
	if !ok {
		pubSubTopic, err := w.upsertTopic(topic)
		if err != nil {
			return nil, false, err
		}

		sub, err = pubSubTopic.Subscribe()
		if err != nil {
			return nil, false, err
		}
		w.relaySubs[topic] = sub

		log.Info("Subscribing to topic ", topic)
	}

	isNew = !ok // ok will be true if subscription already exists
	return sub, isNew, nil
}

func (w *WakuRelay) Publish(ctx context.Context, message *pb.WakuMessage, topic *Topic) ([]byte, error) {
	// Publish a `WakuMessage` to a PubSub topic.

	if w.pubsub == nil {
		return nil, errors.New("PubSub hasn't been set.")
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
