package wakuv2

import (
	"errors"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/wakuv2/common"
	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

type PublishMethod int

const (
	LightPush PublishMethod = iota
	Relay
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

// Send injects a message into the waku send queue, to be distributed in the
// network in the coming cycles.
func (w *Waku) Send(pubsubTopic string, msg *pb.WakuMessage, priority *int) ([]byte, error) {
	pubsubTopic = w.GetPubsubTopic(pubsubTopic)
	if w.protectedTopicStore != nil {
		privKey, err := w.protectedTopicStore.FetchPrivateKey(pubsubTopic)
		if err != nil {
			return nil, err
		}

		if privKey != nil {
			err = relay.SignMessage(privKey, msg, pubsubTopic)
			if err != nil {
				return nil, err
			}
		}
	}

	envelope := protocol.NewEnvelope(msg, msg.GetTimestamp(), pubsubTopic)

	if priority != nil {
		w.sendQueue.Push(envelope, *priority)
	} else {
		w.sendQueue.Push(envelope)
	}

	w.poolMu.Lock()
	alreadyCached := w.envelopeCache.Has(gethcommon.BytesToHash(envelope.Hash().Bytes()))
	w.poolMu.Unlock()
	if !alreadyCached {
		recvMessage := common.NewReceivedMessage(envelope, common.SendMessageType)
		w.postEvent(recvMessage) // notify the local node about the new message
		w.addEnvelope(recvMessage)
	}

	return envelope.Hash().Bytes(), nil
}

func (w *Waku) broadcast() {
	for {
		var envelope *protocol.Envelope

		select {
		case envelope = <-w.sendQueue.Pop():

		case <-w.ctx.Done():
			return
		}

		logger := w.logger.With(zap.Stringer("envelopeHash", envelope.Hash()), zap.String("pubsubTopic", envelope.PubsubTopic()), zap.String("contentTopic", envelope.Message().ContentTopic), zap.Int64("timestamp", envelope.Message().GetTimestamp()))

		var fn publish.PublishFn
		var publishMethod PublishMethod

		if w.cfg.SkipPublishToTopic {
			// For now only used in testing to simulate going offline
			publishMethod = LightPush
			fn = func(env *protocol.Envelope, logger *zap.Logger) error {
				return errors.New("test send failure")
			}
		} else if w.cfg.LightClient {
			publishMethod = LightPush
			fn = func(env *protocol.Envelope, logger *zap.Logger) error {
				logger.Info("publishing message via lightpush")
				_, err := w.node.Lightpush().Publish(w.ctx, env.Message(), lightpush.WithPubSubTopic(env.PubsubTopic()), lightpush.WithMaxPeers(peersToPublishForLightpush))
				return err
			}
		} else {
			publishMethod = Relay
			fn = func(env *protocol.Envelope, logger *zap.Logger) error {
				peerCnt := len(w.node.Relay().PubSub().ListPeers(env.PubsubTopic()))
				logger.Info("publishing message via relay", zap.Int("peerCnt", peerCnt))
				_, err := w.node.Relay().Publish(w.ctx, env.Message(), relay.WithPubSubTopic(env.PubsubTopic()))
				return err
			}
		}

		// Wraps the publish function with a call to the telemetry client
		if w.statusTelemetryClient != nil {
			sendFn := fn
			fn = func(env *protocol.Envelope, logger *zap.Logger) error {
				err := sendFn(env, logger)
				if err == nil {
					w.statusTelemetryClient.PushSentEnvelope(SentEnvelope{Envelope: env, PublishMethod: publishMethod})
				} else {
					w.statusTelemetryClient.PushErrorSendingEnvelope(ErrorSendingEnvelope{Error: err, SentEnvelope: SentEnvelope{Envelope: env, PublishMethod: publishMethod}})
				}
				return err
			}
		}

		// Wraps the publish function with rate limiter
		fn = w.limiter.ThrottlePublishFn(w.ctx, fn)

		w.wg.Add(1)
		go w.publishEnvelope(envelope, fn, logger)
	}
}

func (w *Waku) publishEnvelope(envelope *protocol.Envelope, publishFn publish.PublishFn, logger *zap.Logger) {
	defer w.wg.Done()

	if err := publishFn(envelope, logger); err != nil {
		logger.Error("could not send message", zap.Error(err))
		w.SendEnvelopeEvent(common.EnvelopeEvent{
			Hash:  gethcommon.BytesToHash(envelope.Hash().Bytes()),
			Event: common.EventEnvelopeExpired,
		})
		return
	} else {
		if !w.cfg.EnableStoreConfirmationForMessagesSent {
			w.SendEnvelopeEvent(common.EnvelopeEvent{
				Hash:  gethcommon.BytesToHash(envelope.Hash().Bytes()),
				Event: common.EventEnvelopeSent,
			})
		}
	}
}
