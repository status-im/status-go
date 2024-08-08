package wakuv2

import (
	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/wakuv2/common"
)

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
		err := w.sendQueue.Push(w.ctx, envelope, *priority)
		if err != nil {
			return nil, err
		}
	} else {
		err := w.sendQueue.Push(w.ctx, envelope)
		if err != nil {
			return nil, err
		}
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
		case envelope = <-w.sendQueue.Pop(w.ctx):

		case <-w.ctx.Done():
			return
		}

		w.wg.Add(1)
		go w.publishEnvelope(envelope)
	}
}

func (w *Waku) publishEnvelope(envelope *protocol.Envelope) {
	defer w.wg.Done()

	logger := w.logger.With(zap.Stringer("envelopeHash", envelope.Hash()), zap.String("pubsubTopic", envelope.PubsubTopic()), zap.String("contentTopic", envelope.Message().ContentTopic), zap.Int64("timestamp", envelope.Message().GetTimestamp()))

	// only used in testing to simulate going offline
	if w.cfg.SkipPublishToTopic {
		logger.Info("skipping publish to topic")
		return
	}

	err := w.messageSender.Send(publish.NewRequest(w.ctx, envelope))

	if w.statusTelemetryClient != nil {
		if err == nil {
			w.statusTelemetryClient.PushSentEnvelope(SentEnvelope{Envelope: envelope, PublishMethod: w.messageSender.PublishMethod()})
		} else {
			w.statusTelemetryClient.PushErrorSendingEnvelope(ErrorSendingEnvelope{Error: err, SentEnvelope: SentEnvelope{Envelope: envelope, PublishMethod: w.messageSender.PublishMethod()}})
		}
	}

	if err != nil {
		logger.Error("could not send message", zap.Error(err))
		w.SendEnvelopeEvent(common.EnvelopeEvent{
			Hash:  gethcommon.BytesToHash(envelope.Hash().Bytes()),
			Event: common.EventEnvelopeExpired,
		})
		return
	}

	if !w.cfg.EnableStoreConfirmationForMessagesSent {
		w.SendEnvelopeEvent(common.EnvelopeEvent{
			Hash:  gethcommon.BytesToHash(envelope.Hash().Bytes()),
			Event: common.EventEnvelopeSent,
		})
	}
}
