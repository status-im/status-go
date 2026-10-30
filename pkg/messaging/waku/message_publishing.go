package waku

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/diagnostics"
	common "github.com/status-im/status-go/pkg/messaging/waku/common"
)

// sendEnvelope injects a prebuilt WakuMessage envelope into the waku send
// queue, to be distributed in the network in the coming cycles. It is the
// low-level publish step shared by the exported Send method and tests.
func (w *Waku) sendEnvelope(pubsubTopic string, msg *pb.WakuMessage, priority *int, trace ...diagnostics.Metadata) ([]byte, error) {
	pubsubTopic = w.GetPubsubTopic(pubsubTopic)

	envelope := protocol.NewEnvelope(msg, msg.GetTimestamp(), pubsubTopic)
	metadata := diagnostics.Metadata{}
	if diagnostics.Enabled() {
		if len(trace) > 0 {
			metadata = trace[0]
		}
		w.diagnosticAttempts.Store(envelope.Hash().String(), metadata)
		diagnostics.Emit(diagnostics.Event{Metadata: metadata, Stage: "queue_entered", EnvelopeHashes: []string{envelope.Hash().String()}})
	}
	queueFailure := func(err error) ([]byte, error) {
		if diagnostics.Enabled() {
			w.diagnosticAttempts.Delete(envelope.Hash().String())
			diagnostics.Emit(diagnostics.Event{Metadata: metadata, Stage: "publication_failed", EnvelopeHashes: []string{envelope.Hash().String()}, Source: "queue", Error: err.Error()})
		}
		return nil, err
	}

	if priority != nil {
		err := w.sendQueue.Push(w.ctx, envelope, *priority)
		if err != nil {
			return queueFailure(err)
		}
	} else {
		err := w.sendQueue.Push(w.ctx, envelope)
		if err != nil {
			return queueFailure(err)
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
	metadata := diagnostics.Metadata{}
	var started time.Time
	if diagnostics.Enabled() {
		if value, ok := w.diagnosticAttempts.LoadAndDelete(envelope.Hash().String()); ok {
			metadata = value.(diagnostics.Metadata)
		}
		started = time.Now()
		diagnostics.Emit(diagnostics.Event{Metadata: metadata, Stage: "publish_started", EnvelopeHashes: []string{envelope.Hash().String()}, Source: w.messageSender.PublishMethod().String(), PublicationAttempt: 1})
	}

	var err error
	publicationAttempts := 1
	// only used in testing to simulate going offline
	if w.cfg.SkipPublishToTopic {
		logger.Debug("skipping publish to topic")
		err = errors.New("test send failure")
	} else {
		send := func(ctx context.Context) error {
			return w.messageSender.Send(publish.NewRequest(ctx, envelope))
		}
		if w.messageSender.PublishMethod() == publish.LightPush {
			publicationAttempts, err = publishWithLightpushRetry(w.ctx, send, func(attempt int, delay time.Duration, cause error) {
				logger.Debug("scheduling rate-limited lightpush publication retry", zap.Int("publicationAttempt", attempt), zap.Duration("retryDelay", delay))
				if diagnostics.Enabled() {
					diagnostics.Emit(diagnostics.Event{
						Metadata: metadata, Stage: "publication_retry_scheduled",
						EnvelopeHashes: []string{envelope.Hash().String()}, Source: publish.LightPush.String(),
						Error: cause.Error(), PublicationAttempt: attempt, RetryDelayNS: int64(delay),
					})
				}
			})
		} else {
			err = send(w.ctx)
		}
	}
	if diagnostics.Enabled() {
		event := diagnostics.Event{Metadata: metadata, Stage: "publication_accepted", EnvelopeHashes: []string{envelope.Hash().String()}, Source: w.messageSender.PublishMethod().String(), DurationNS: time.Since(started).Nanoseconds(), PublicationAttempt: publicationAttempts}
		if err != nil {
			event.Stage = "publication_failed"
			event.Error = err.Error()
		}
		diagnostics.Emit(event)
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
