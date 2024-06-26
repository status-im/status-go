package wakuv2

import (
	"context"
	"encoding/hex"
	"errors"
	"slices"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/wakuv2/common"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

const maxContentTopicsPerRequest = 10
const maxAttemptsToRetrieveHistory = 3
const delay = 10 * time.Second

type TopicInterest struct {
	peerID        peer.ID
	pubsubTopic   string
	contentTopics []string
	lastChecked   time.Time

	ctx    context.Context
	cancel context.CancelFunc
}

func (p TopicInterest) Equals(other TopicInterest) bool {
	if p.peerID != other.peerID {
		return false
	}

	if p.pubsubTopic != other.pubsubTopic {
		return false
	}

	slices.Sort(p.contentTopics)
	slices.Sort(other.contentTopics)

	if len(p.contentTopics) != len(other.contentTopics) {
		return false
	}

	for i, contentTopic := range p.contentTopics {
		if contentTopic != other.contentTopics[i] {
			return false
		}
	}

	return true
}

func (w *Waku) SetTopicsToVerifyForMissingMessages(peerID peer.ID, pubsubTopic string, contentTopics []string) {
	w.topicInterestMu.Lock()
	defer w.topicInterestMu.Unlock()

	ctx, cancel := context.WithCancel(w.ctx)
	newMissingMessageRequest := TopicInterest{
		peerID:        peerID,
		pubsubTopic:   pubsubTopic,
		contentTopics: contentTopics,
		lastChecked:   w.timesource.Now().Add(delay),
		ctx:           ctx,
		cancel:        cancel,
	}

	currMessageVerificationRequest, ok := w.topicInterest[pubsubTopic]

	if ok && currMessageVerificationRequest.Equals(newMissingMessageRequest) {
		return
	}

	if ok {
		// If there is an ongoing request, we cancel it before replacing it
		// by the new list. This can be probably optimized further by tracking
		// the last time a content topic was synced, but might not be necessary
		// since cancelling an ongoing request would mean cancelling just a single
		// page of results
		currMessageVerificationRequest.cancel()
	}

	w.topicInterest[pubsubTopic] = newMissingMessageRequest
}

func (w *Waku) checkForMissingMessages() {
	defer w.wg.Done()
	defer w.logger.Debug("checkForMissingMessages - done")

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	var semaphore = make(chan struct{}, 5)
	for {
		select {
		case <-t.C:
			w.logger.Debug("checking for missing messages...")
			w.topicInterestMu.Lock()
			for _, request := range w.topicInterest {
				select {
				case <-w.ctx.Done():
					return
				default:
					semaphore <- struct{}{}
					go func(r TopicInterest) {
						w.FetchHistory(r)
						<-semaphore
					}(request)
				}
			}
			w.topicInterestMu.Unlock()

		case <-w.ctx.Done():
			return
		}
	}
}

func (w *Waku) FetchHistory(missingHistoryRequest TopicInterest) {
	for i := 0; i < len(missingHistoryRequest.contentTopics); i += maxContentTopicsPerRequest {
		j := i + maxContentTopicsPerRequest
		if j > len(missingHistoryRequest.contentTopics) {
			j = len(missingHistoryRequest.contentTopics)
		}

		now := w.timesource.Now()
		err := w.fetchMessagesBatch(missingHistoryRequest, i, j, now)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			w.logger.Error("could not fetch history", zap.Stringer("peerID", missingHistoryRequest.peerID), zap.String("pubsubTopic", missingHistoryRequest.pubsubTopic), zap.Strings("contentTopics", missingHistoryRequest.contentTopics))
			continue
		}

		w.topicInterestMu.Lock()
		c := w.topicInterest[missingHistoryRequest.pubsubTopic]
		if c.Equals(missingHistoryRequest) {
			c.lastChecked = now
			w.topicInterest[missingHistoryRequest.pubsubTopic] = c
		}
		w.topicInterestMu.Unlock()
	}
}

func (w *Waku) storeQueryWithRetry(ctx context.Context, queryFunc func(ctx context.Context) (*store.Result, error), logger *zap.Logger, logMsg string) (*store.Result, error) {
	retry := true
	count := 1
	for retry && count <= maxAttemptsToRetrieveHistory {
		logger.Debug(logMsg, zap.Int("attempt", count))
		tCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		result, err := queryFunc(tCtx)
		cancel()
		if err != nil {
			logger.Error("could not query storenode", zap.Error(err), zap.Int("attempt", count))
			select {
			case <-w.ctx.Done():
				return nil, w.ctx.Err()
			case <-time.After(2 * time.Second):
			}
		} else {
			return result, nil
		}
	}

	return nil, errors.New("storenode not available")
}

func (w *Waku) fetchMessagesBatch(missingHistoryRequest TopicInterest, batchFrom int, batchTo int, now time.Time) error {
	logger := w.logger.With(
		zap.Stringer("peerID", missingHistoryRequest.peerID),
		zap.Strings("contentTopics", missingHistoryRequest.contentTopics[batchFrom:batchTo]),
		zap.String("pubsubTopic", missingHistoryRequest.pubsubTopic),
		logutils.WakuMessageTimestamp("from", proto.Int64(missingHistoryRequest.lastChecked.UnixNano())),
		logutils.WakuMessageTimestamp("to", proto.Int64(now.UnixNano())),
	)

	result, err := w.storeQueryWithRetry(missingHistoryRequest.ctx, func(ctx context.Context) (*store.Result, error) {
		return w.node.Store().Query(ctx, store.FilterCriteria{
			ContentFilter: protocol.NewContentFilter(missingHistoryRequest.pubsubTopic, missingHistoryRequest.contentTopics[batchFrom:batchTo]...),
			TimeStart:     proto.Int64(missingHistoryRequest.lastChecked.Add(-delay).UnixNano()),
			TimeEnd:       proto.Int64(now.Add(-delay).UnixNano()),
		}, store.WithPeer(missingHistoryRequest.peerID), store.WithPaging(false, 100), store.IncludeData(false))
	}, logger, "retrieving history to check for missing messages")
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error("storenode not available", zap.Error(err))
		}
		return err
	}

	var missingMessages []pb.MessageHash

	for !result.IsComplete() {
		for _, mkv := range result.Messages() {
			hash := pb.ToMessageHash(mkv.MessageHash)

			w.poolMu.Lock()
			alreadyCached := w.envelopeCache.Has(gethcommon.Hash(hash))
			w.poolMu.Unlock()
			if alreadyCached {
				continue
			}

			missingMessages = append(missingMessages, hash)
		}

		result, err = w.storeQueryWithRetry(missingHistoryRequest.ctx, func(ctx context.Context) (*store.Result, error) {
			if err = result.Next(ctx); err != nil {
				return nil, err
			}
			return result, nil
		}, logger.With(zap.String("cursor", hex.EncodeToString(result.Cursor()))), "retrieving next page")
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Error("storenode not available", zap.Error(err))
			}
			return err
		}
	}

	if len(missingMessages) == 0 {
		// Nothing to do here
		return nil
	}

	result, err = w.storeQueryWithRetry(missingHistoryRequest.ctx, func(ctx context.Context) (*store.Result, error) {
		return w.node.Store().QueryByHash(ctx, missingMessages, store.WithPeer(missingHistoryRequest.peerID), store.WithPaging(false, 100))
	}, logger, "retrieving missing messages")
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error("storenode not available", zap.Error(err))
		}
		return err
	}

	for !result.IsComplete() {
		for _, mkv := range result.Messages() {
			envelope := protocol.NewEnvelope(mkv.Message, mkv.Message.GetTimestamp(), mkv.GetPubsubTopic())
			w.logger.Info("received waku2 store message",
				zap.Stringer("envelopeHash", envelope.Hash()),
				zap.String("pubsubTopic", mkv.GetPubsubTopic()),
				zap.Int64p("timestamp", envelope.Message().Timestamp),
			)

			err = w.OnNewEnvelopes(envelope, common.StoreMessageType, false)
			if err != nil {
				return err
			}
		}

		result, err = w.storeQueryWithRetry(missingHistoryRequest.ctx, func(ctx context.Context) (*store.Result, error) {
			if err = result.Next(ctx); err != nil {
				return nil, err
			}
			return result, nil
		}, logger.With(zap.String("cursor", hex.EncodeToString(result.Cursor()))), "retrieving next page")
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Error("storenode not available", zap.Error(err))
			}
			return err
		}
	}

	return nil
}
