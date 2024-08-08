package publish

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
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

type MessageSender struct {
	publishMethod    PublishMethod
	lightPush        *lightpush.WakuLightPush
	relay            *relay.WakuRelay
	messageSentCheck ISentCheck
	rateLimiter      *PublishRateLimiter
	logger           *zap.Logger
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

func NewMessageSender(publishMethod PublishMethod, lightPush *lightpush.WakuLightPush, relay *relay.WakuRelay, logger *zap.Logger) (*MessageSender, error) {
	if publishMethod == UnknownMethod {
		return nil, errors.New("publish method is required")
	}
	return &MessageSender{
		publishMethod: publishMethod,
		lightPush:     lightPush,
		relay:         relay,
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
		if ms.lightPush == nil {
			return errors.New("lightpush is not available")
		}
		logger.Info("publishing message via lightpush")
		_, err := ms.lightPush.Publish(
			req.ctx,
			req.envelope.Message(),
			lightpush.WithPubSubTopic(req.envelope.PubsubTopic()),
			lightpush.WithMaxPeers(DefaultPeersToPublishForLightpush),
		)
		if err != nil {
			return err
		}
	case Relay:
		if ms.relay == nil {
			return errors.New("relay is not available")
		}
		peerCnt := len(ms.relay.PubSub().ListPeers(req.envelope.PubsubTopic()))
		logger.Info("publishing message via relay", zap.Int("peerCnt", peerCnt))
		_, err := ms.relay.Publish(req.ctx, req.envelope.Message(), relay.WithPubSubTopic(req.envelope.PubsubTopic()))
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

func (ms *MessageSender) SetStorePeerID(peerID peer.ID) {
	if ms.messageSentCheck != nil {
		ms.messageSentCheck.SetStorePeerID(peerID)
	}
}
