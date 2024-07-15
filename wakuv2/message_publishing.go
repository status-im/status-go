package wakuv2

import (
	"container/heap"
	"errors"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/wakuv2/common"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

const defaultPriority = 2

type envelopePriority struct {
	envelope *protocol.Envelope
	priority int
	index    int
}

type envelopePriorityQueue []*envelopePriority

func (pq envelopePriorityQueue) Len() int { return len(pq) }

func (pq envelopePriorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

func (pq envelopePriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *envelopePriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*envelopePriority)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *envelopePriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

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
	msgPriority := defaultPriority
	if priority != nil {
		msgPriority = *priority
	}

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

	if w.cfg.UseThrottledPublish {
		w.throttledPrioritySendQueue <- &envelopePriority{
			envelope: envelope,
			priority: msgPriority,
		}
	} else {
		w.sendQueue <- envelope
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

func (w *Waku) handleEnvelopePriority() {
	defer w.wg.Done()

	if !w.cfg.UseThrottledPublish {
		return
	}

	for {
		select {
		case envelopePriority := <-w.throttledPrioritySendQueue:
			heap.Push(&w.envelopePriorityQueue, envelopePriority)
			w.envelopeToSendAvailable <- struct{}{}
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *Waku) broadcast() {
	for {
		var envelope *protocol.Envelope

		select {
		case <-w.envelopeToSendAvailable:
			envelope = heap.Pop(&w.envelopePriorityQueue).(*envelopePriority).envelope

		case envelope = <-w.sendQueue:

		case <-w.ctx.Done():
			return
		}

		logger := w.logger.With(zap.Stringer("envelopeHash", envelope.Hash()), zap.String("pubsubTopic", envelope.PubsubTopic()), zap.String("contentTopic", envelope.Message().ContentTopic), zap.Int64("timestamp", envelope.Message().GetTimestamp()))

		var fn publishFn
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

		w.wg.Add(1)
		go w.publishEnvelope(envelope, fn, logger)
	}
}
