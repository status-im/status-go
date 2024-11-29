package publish

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const DefaultPeersToPublishForLightpush = 2
const DefaultPublishingLimiterRate = rate.Limit(2)
const DefaultPublishingLimitBurst = 4

type PublishMethod int

const (
	LightPush PublishMethod = iota
	Relay
	UnknownMethod
)

func (pm PublishMethod) String() string {
	switch pm {
	case LightPush:
		return "LightPush"
	case Relay:
		return "Relay"
	default:
		return "Unknown"
	}
}

type Publisher interface {
	// RelayListPeers returns the list of peers for a pubsub topic
	RelayListPeers(pubsubTopic string) ([]peer.ID, error)

	// RelayPublish publishes a message via WakuRelay
	RelayPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string) (pb.MessageHash, error)

	// LightpushPublish publishes a message via WakuLightPush
	LightpushPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string, maxPeers int) (pb.MessageHash, error)
}

type MessageSender struct {
	publishMethod    PublishMethod
	publisher        Publisher
	messageSentCheck ISentCheck
	rateLimiter      *PublishRateLimiter
	logger           *zap.Logger
	evtMessageSent   event.Emitter
}

type MessageSent struct {
	Size      uint32 // Size of payload in bytes
	Timestamp int64
}

type Request struct {
	ctx           context.Context
	envelope      *protocol.Envelope
	publishMethod PublishMethod
}

func NewRequest(ctx context.Context, envelope *protocol.Envelope) *Request {
	return &Request{
		ctx:           ctx,
		envelope:      envelope,
		publishMethod: UnknownMethod,
	}
}

func (r *Request) WithPublishMethod(publishMethod PublishMethod) *Request {
	r.publishMethod = publishMethod
	return r
}

func NewMessageSender(publishMethod PublishMethod, publisher Publisher, logger *zap.Logger) (*MessageSender, error) {
	if publishMethod == UnknownMethod {
		return nil, errors.New("publish method is required")
	}
	return &MessageSender{
		publishMethod: publishMethod,
		publisher:     publisher,
		rateLimiter:   NewPublishRateLimiter(DefaultPublishingLimiterRate, DefaultPublishingLimitBurst),
		logger:        logger,
	}, nil
}

func (ms *MessageSender) WithMessageSentCheck(messageSentCheck ISentCheck) *MessageSender {
	ms.messageSentCheck = messageSentCheck
	return ms
}

func (ms *MessageSender) WithRateLimiting(rateLimiter *PublishRateLimiter) *MessageSender {
	ms.rateLimiter = rateLimiter
	return ms
}

func (ms *MessageSender) WithMessageSentEmitter(host host.Host) *MessageSender {
	evtMessageSent, err := host.EventBus().Emitter(new(MessageSent))
	if err != nil {
		ms.logger.Error("failed to create message sent emitter", zap.Error(err))
	}
	ms.evtMessageSent = evtMessageSent
	return ms
}

func (ms *MessageSender) Send(req *Request) error {
	logger := ms.logger.With(
		zap.Stringer("envelopeHash", req.envelope.Hash()),
		zap.String("pubsubTopic", req.envelope.PubsubTopic()),
		zap.String("contentTopic", req.envelope.Message().ContentTopic),
		zap.Int64("timestamp", req.envelope.Message().GetTimestamp()),
	)

	if ms.rateLimiter != nil {
		if err := ms.rateLimiter.Check(req.ctx, logger); err != nil {
			return err
		}
	}

	publishMethod := req.publishMethod
	if publishMethod == UnknownMethod {
		publishMethod = ms.publishMethod
	}

	switch publishMethod {
	case LightPush:
		logger.Info("publishing message via lightpush")
		_, err := ms.publisher.LightpushPublish(
			req.ctx,
			req.envelope.Message(),
			req.envelope.PubsubTopic(),
			DefaultPeersToPublishForLightpush,
		)
		if err != nil {
			return err
		}
	case Relay:
		peers, err := ms.publisher.RelayListPeers(req.envelope.PubsubTopic())
		if err != nil {
			return err
		}
		logger.Info("publishing message via relay", zap.Int("peerCnt", len(peers)))
		_, err = ms.publisher.RelayPublish(req.ctx, req.envelope.Message(), req.envelope.PubsubTopic())
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown publish method")
	}

	if ms.messageSentCheck != nil && !req.envelope.Message().GetEphemeral() {
		ms.messageSentCheck.Add(
			req.envelope.PubsubTopic(),
			common.BytesToHash(req.envelope.Hash().Bytes()),
			uint32(req.envelope.Message().GetTimestamp()/int64(time.Second)),
		)
	}

	if ms.evtMessageSent != nil {
		err := ms.evtMessageSent.Emit(MessageSent{
			Size:      uint32(len(req.envelope.Message().Payload)),
			Timestamp: req.envelope.Message().GetTimestamp(),
		})
		if err != nil {
			logger.Error("failed to emit message sent event", zap.Error(err))
		}
	}

	return nil
}

func (ms *MessageSender) Start() {
	if ms.messageSentCheck != nil {
		go ms.messageSentCheck.Start()
	}
}

func (ms *MessageSender) PublishMethod() PublishMethod {
	return ms.publishMethod
}

func (ms *MessageSender) MessagesDelivered(messageIDs []common.Hash) {
	if ms.messageSentCheck != nil {
		ms.messageSentCheck.DeleteByMessageIDs(messageIDs)
	}
}
