package history

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/api/common"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"
)

const maxTopicsPerRequest int = 10
const mailserverRequestTimeout = 30 * time.Second

type work struct {
	criteria store.FilterCriteria
	cursor   []byte
	limit    uint64
}

type HistoryRetriever struct {
	store            common.StorenodeRequestor
	logger           *zap.Logger
	historyProcessor HistoryProcessor
}

type HistoryProcessor interface {
	OnEnvelope(env *protocol.Envelope, processEnvelopes bool) error
	OnRequestFailed(requestID []byte, peerID peer.ID, err error)
}

func NewHistoryRetriever(store common.StorenodeRequestor, historyProcessor HistoryProcessor, logger *zap.Logger) *HistoryRetriever {
	return &HistoryRetriever{
		store:            store,
		logger:           logger.Named("history-retriever"),
		historyProcessor: historyProcessor,
	}
}

func (hr *HistoryRetriever) Query(
	ctx context.Context,
	criteria store.FilterCriteria,
	storenodeID peer.ID,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	logger := hr.logger.With(
		logging.Timep("fromString", criteria.TimeStart),
		logging.Timep("toString", criteria.TimeEnd),
		zap.String("pubsubTopic", criteria.PubsubTopic),
		zap.Strings("contentTopics", criteria.ContentTopicsList()),
		zap.Int64p("from", criteria.TimeStart),
		zap.Int64p("to", criteria.TimeEnd),
	)

	logger.Info("syncing")

	wg := sync.WaitGroup{}
	workWg := sync.WaitGroup{}
	workCh := make(chan work, 1000)       // each batch item is split in 10 topics bunch and sent to this channel
	workCompleteCh := make(chan struct{}) // once all batch items are processed, this channel is triggered
	semaphore := make(chan struct{}, 3)   // limit the number of concurrent queries
	errCh := make(chan error)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// TODO: refactor this by extracting the consumer into a separate go routine.

	// Producer
	wg.Add(1)
	go func() {
		defer func() {
			logger.Debug("mailserver batch producer complete")
			wg.Done()
		}()

		contentTopicList := criteria.ContentTopics.ToList()

		// TODO: split into 24h batches

		allWorks := int(math.Ceil(float64(len(contentTopicList)) / float64(maxTopicsPerRequest)))
		workWg.Add(allWorks)

		for i := 0; i < len(contentTopicList); i += maxTopicsPerRequest {
			j := i + maxTopicsPerRequest
			if j > len(contentTopicList) {
				j = len(contentTopicList)
			}

			select {
			case <-ctx.Done():
				logger.Debug("processBatch producer - context done")
				return
			default:
				logger.Debug("processBatch producer - creating work")
				workCh <- work{
					criteria: store.FilterCriteria{
						ContentFilter: protocol.NewContentFilter(criteria.PubsubTopic, contentTopicList[i:j]...),
						TimeStart:     criteria.TimeStart,
						TimeEnd:       criteria.TimeEnd,
					},
					limit: pageLimit,
				}
			}
		}

		go func() {
			workWg.Wait()
			workCompleteCh <- struct{}{}
		}()

		logger.Debug("processBatch producer complete")
	}()

	var result error

loop:
	for {
		select {
		case <-ctx.Done():
			logger.Debug("processBatch cleanup - context done")
			result = ctx.Err()
			if errors.Is(result, context.Canceled) {
				result = nil
			}
			break loop
		case w, ok := <-workCh:
			if !ok {
				continue
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// continue...
			}

			logger.Debug("processBatch - received work")

			semaphore <- struct{}{}
			go func(w work) { // Consumer
				defer func() {
					workWg.Done()
					<-semaphore
				}()

				queryCtx, queryCancel := context.WithTimeout(ctx, mailserverRequestTimeout)

				// If time range is greater than 24 hours, limit the range: to - (to-24h)
				// TODO: handle cases in which TimeStart/TimeEnd could be nil
				//       (this type of query does not happen in status-go, though, and
				//       nwaku might limit query duration to 24h anyway, so perhaps
				//       it's not worth adding such logic)
				timeStart := w.criteria.TimeStart
				timeEnd := w.criteria.TimeEnd
				exceeds24h := false
				if timeStart != nil && timeEnd != nil && *timeEnd-*timeStart > (24*time.Hour).Nanoseconds() {
					newTimeStart := *timeEnd - (24 * time.Hour).Nanoseconds()
					timeStart = &newTimeStart
					exceeds24h = true
				}

				newCriteria := w.criteria
				newCriteria.TimeStart = timeStart
				newCriteria.TimeEnd = timeEnd

				cursor, envelopesCount, err := hr.createMessagesRequest(queryCtx, storenodeID, newCriteria, w.cursor, w.limit, true, processEnvelopes, logger)
				queryCancel()

				if err != nil {
					logger.Debug("failed to send request", zap.Error(err))
					errCh <- err
					return
				}

				processNextPage := true
				nextPageLimit := pageLimit
				if shouldProcessNextPage != nil {
					processNextPage, nextPageLimit = shouldProcessNextPage(envelopesCount)
				}

				if !processNextPage {
					return
				}

				// Check the cursor after calling `shouldProcessNextPage`.
				// The app might use process the fetched envelopes in the callback for own needs.
				// If from/to does not exceed 24h and no cursor was returned, we have already
				// requested the entire time range
				if cursor == nil && !exceeds24h {
					return
				}

				logger.Debug("processBatch producer - creating work (cursor)")

				newWork := work{
					criteria: w.criteria,
					cursor:   cursor,
					limit:    nextPageLimit,
				}

				// If from/to has exceeded the 24h, but there are no more records within the current
				// 24h range, then we update the `to` for the new work to not include it.
				if cursor == nil && exceeds24h {
					newWork.criteria.TimeEnd = timeStart
				}

				workWg.Add(1)
				workCh <- newWork
			}(w)
		case err := <-errCh:
			logger.Debug("processBatch - received error", zap.Error(err))
			cancel() // Kill go routines
			return err
		case <-workCompleteCh:
			logger.Debug("processBatch - all jobs complete")
			cancel() // Kill go routines
		}
	}

	wg.Wait()

	logger.Info("synced topic", zap.NamedError("hasError", result))

	return result
}

func (hr *HistoryRetriever) createMessagesRequest(
	ctx context.Context,
	peerID peer.ID,
	criteria store.FilterCriteria,
	cursor []byte,
	limit uint64,
	waitForResponse bool,
	processEnvelopes bool,
	logger *zap.Logger,
) (storeCursor []byte, envelopesCount int, err error) {
	if waitForResponse {
		resultCh := make(chan struct {
			storeCursor    []byte
			envelopesCount int
			err            error
		})

		go func() {
			storeCursor, envelopesCount, err = hr.requestStoreMessages(ctx, peerID, criteria, cursor, limit, processEnvelopes)
			resultCh <- struct {
				storeCursor    []byte
				envelopesCount int
				err            error
			}{storeCursor, envelopesCount, err}
		}()

		select {
		case result := <-resultCh:
			return result.storeCursor, result.envelopesCount, result.err
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		}
	} else {
		go func() {
			_, _, err = hr.requestStoreMessages(ctx, peerID, criteria, cursor, limit, false)
			if err != nil {
				logger.Error("failed to request store messages", zap.Error(err))
			}
		}()
	}

	return
}

func (hr *HistoryRetriever) requestStoreMessages(ctx context.Context, peerID peer.ID, criteria store.FilterCriteria, cursor []byte, limit uint64, processEnvelopes bool) ([]byte, int, error) {
	requestID := protocol.GenerateRequestID()
	logger := hr.logger.With(zap.String("requestID", hexutil.Encode(requestID)), zap.Stringer("peerID", peerID))

	logger.Debug("store.query",
		logging.Timep("startTime", criteria.TimeStart),
		logging.Timep("endTime", criteria.TimeEnd),
		zap.Strings("contentTopics", criteria.ContentTopics.ToList()),
		zap.String("pubsubTopic", criteria.PubsubTopic),
		zap.String("cursor", hexutil.Encode(cursor)),
	)

	storeQueryRequest := &pb.StoreQueryRequest{
		RequestId:        hex.EncodeToString(requestID),
		IncludeData:      true,
		PubsubTopic:      &criteria.PubsubTopic,
		ContentTopics:    criteria.ContentTopicsList(),
		TimeStart:        criteria.TimeStart,
		TimeEnd:          criteria.TimeEnd,
		PaginationCursor: cursor,
		PaginationLimit:  proto.Uint64(limit),
	}

	queryStart := time.Now()
	result, err := hr.store.Query(ctx, peerID, storeQueryRequest)
	queryDuration := time.Since(queryStart)
	if err != nil {
		logger.Error("error querying storenode", zap.Error(err))

		hr.historyProcessor.OnRequestFailed(requestID, peerID, err)

		return nil, 0, err
	}

	messages := result.Messages()
	envelopesCount := len(messages)
	logger.Debug("store.query response", zap.Duration("queryDuration", queryDuration), zap.Int("numMessages", envelopesCount), zap.Bool("hasCursor", result.IsComplete() && result.Cursor() != nil))
	for _, mkv := range messages {
		envelope := protocol.NewEnvelope(mkv.Message, mkv.Message.GetTimestamp(), mkv.GetPubsubTopic())
		err := hr.historyProcessor.OnEnvelope(envelope, processEnvelopes)
		if err != nil {
			return nil, 0, err
		}
	}
	return result.Cursor(), envelopesCount, nil
}
