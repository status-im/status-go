package relay

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"

	proto "github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"

	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var log = logging.Logger("wakurelay")

type Topic string

const WakuRelayID_v200 = protocol.ID("/vac/waku/relay/2.0.0")
const DefaultWakuTopic Topic = "/waku/2/default-waku/proto"

type WakuRelay struct {
	host   host.Host
	pubsub *pubsub.PubSub

	topics          map[Topic]bool
	topicsMutex     sync.Mutex
	wakuRelayTopics map[Topic]*pubsub.Topic
	relaySubs       map[Topic]*pubsub.Subscription
}

// Once https://github.com/status-im/nim-waku/issues/420 is fixed, implement a custom messageIdFn
func msgIdFn(pmsg *pubsub_pb.Message) string {
	hash := sha256.Sum256(pmsg.Data)
	return string(hash[:])
}

func NewWakuRelay(ctx context.Context, h host.Host, opts ...pubsub.Option) (*WakuRelay, error) {
	w := new(WakuRelay)
	w.host = h
	w.topics = make(map[Topic]bool)
	w.wakuRelayTopics = make(map[Topic]*pubsub.Topic)
	w.relaySubs = make(map[Topic]*pubsub.Subscription)

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

func (w *WakuRelay) Subscribe(topic Topic) (subs *pubsub.Subscription, isNew bool, err error) {

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
}
