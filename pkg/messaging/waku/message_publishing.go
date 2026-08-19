package waku

import (
	"errors"

	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/panics"
	common "github.com/status-im/status-go/pkg/messaging/waku/common"
)

// sendEnvelope injects a prebuilt WakuMessage envelope into the waku send
// queue, to be distributed in the network in the coming cycles. It is the
// low-level publish step shared by the exported Send method and tests.
func (w *Waku) sendEnvelope(pubsubTopic string, msg *pb.WakuMessage, priority *int) ([]byte, error) {
	pubsubTopic = w.GetPubsubTopic(pubsubTopic)

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
	defer panics.LogOnPanic()
	defer w.wg.Done()
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
	defer panics.LogOnPanic()
	defer w.wg.Done()

	logger := w.logger.With(zap.Stringer("envelopeHash", envelope.Hash()), zap.String("pubsubTopic", envelope.PubsubTopic()), zap.String("contentTopic", envelope.Message().ContentTopic), zap.Int64("timestamp", envelope.Message().GetTimestamp()))

	var err error
	// only used in testing to simulate going offline
	if w.cfg.SkipPublishToTopic {
		logger.Debug("skipping publish to topic")
		err = errors.New("test send failure")
	} else {
		err = w.messageSender.Send(publish.NewRequest(w.ctx, envelope))
	}

	if w.metricsHandler != nil {
		if err == nil {
			w.metricsHandler.PushSentEnvelope(SentEnvelope{Envelope: envelope, PublishMethod: w.messageSender.PublishMethod()})
		} else {
			w.metricsHandler.PushErrorSendingEnvelope(ErrorSendingEnvelope{Error: err, SentEnvelope: SentEnvelope{Envelope: envelope, PublishMethod: w.messageSender.PublishMethod()}})
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

	// A successful publish is the authoritative sent confirmation.
	w.SendEnvelopeEvent(common.EnvelopeEvent{
		Hash:  gethcommon.BytesToHash(envelope.Hash().Bytes()),
		Event: common.EventEnvelopeSent,
	})
}
