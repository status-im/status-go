package store

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio/pbio"
	"go.uber.org/zap"

	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
)

// MaxTimeVariance is the maximum duration in the future allowed for a message timestamp
const MaxTimeVariance = time.Duration(20) * time.Second

func findMessages(query *pb.HistoryQuery, msgProvider MessageProvider) ([]*wpb.WakuMessage, *pb.PagingInfo, error) {
	if query.PagingInfo == nil {
		query.PagingInfo = &pb.PagingInfo{
			Direction: pb.PagingInfo_FORWARD,
		}
	}

	if query.PagingInfo.PageSize == 0 || query.PagingInfo.PageSize > uint64(MaxPageSize) {
		query.PagingInfo.PageSize = MaxPageSize
	}

	if len(query.ContentFilters) > MaxContentFilters {
		return nil, nil, ErrMaxContentFilters
	}

	cursor, queryResult, err := msgProvider.Query(query)
	if err != nil {
		return nil, nil, err
	}

	if len(queryResult) == 0 { // no pagination is needed for an empty list
		return nil, &pb.PagingInfo{Cursor: nil}, nil
	}

	resultMessages := make([]*wpb.WakuMessage, len(queryResult))
	for i := range queryResult {
		resultMessages[i] = queryResult[i].Message
	}

	return resultMessages, &pb.PagingInfo{Cursor: cursor}, nil
}

func (store *WakuStore) FindMessages(query *pb.HistoryQuery) *pb.HistoryResponse {
	result := new(pb.HistoryResponse)

	messages, newPagingInfo, err := findMessages(query, store.msgProvider)
	if err != nil {
		if err == persistence.ErrInvalidCursor {
			result.Error = pb.HistoryResponse_INVALID_CURSOR
		} else {
			// TODO: return error in pb.HistoryResponse
			store.log.Error("obtaining messages from db", zap.Error(err))
		}
	}

	result.Messages = messages
	result.PagingInfo = newPagingInfo
	return result
}

type MessageProvider interface {
	GetAll() ([]persistence.StoredMessage, error)
	Query(query *pb.HistoryQuery) (*pb.Index, []persistence.StoredMessage, error)
	Put(env *protocol.Envelope) error
	MostRecentTimestamp() (int64, error)
	Start(timesource timesource.Timesource) error
	Stop()
	Count() (int, error)
}

type Store interface {
	Start(ctx context.Context) error
	Query(ctx context.Context, query Query, opts ...HistoryRequestOption) (*Result, error)
	Find(ctx context.Context, query Query, cb criteriaFN, opts ...HistoryRequestOption) (*wpb.WakuMessage, error)
	Next(ctx context.Context, r *Result) (*Result, error)
	Resume(ctx context.Context, pubsubTopic string, peerList []peer.ID) (int, error)
	MessageChannel() chan *protocol.Envelope
	Stop()
}

// SetMessageProvider allows switching the message provider used with a WakuStore
func (store *WakuStore) SetMessageProvider(p MessageProvider) {
	store.msgProvider = p
}

// Start initializes the WakuStore by enabling the protocol and fetching records from a message provider
func (store *WakuStore) Start(ctx context.Context) error {
	if store.started {
		return nil
	}

	if store.msgProvider == nil {
		store.log.Info("Store protocol started (no message provider)")
		return nil
	}

	err := store.msgProvider.Start(store.timesource)
	if err != nil {
		store.log.Error("Error starting message provider", zap.Error(err))
		return nil
	}

	store.started = true
	store.ctx = ctx
	store.MsgC = make(chan *protocol.Envelope, 1024)

	store.h.SetStreamHandlerMatch(StoreID_v20beta4, protocol.PrefixTextMatch(string(StoreID_v20beta4)), store.onRequest)

	store.wg.Add(2)
	go store.storeIncomingMessages(ctx)
	go store.updateMetrics(ctx)

	store.log.Info("Store protocol started")

	return nil
}

func (store *WakuStore) storeMessage(env *protocol.Envelope) error {
	// Ensure that messages don't "jump" to the front of the queue with future timestamps
	if env.Index().SenderTime-env.Index().ReceiverTime > int64(MaxTimeVariance) {
		return ErrFutureMessage
	}

	if env.Message().Ephemeral {
		return nil
	}

	err := store.msgProvider.Put(env)
	if err != nil {
		store.log.Error("storing message", zap.Error(err))
		metrics.RecordStoreError(store.ctx, "store_failure")
		return err
	}

	return nil
}

func (store *WakuStore) storeIncomingMessages(ctx context.Context) {
	defer store.wg.Done()
	for envelope := range store.MsgC {
		go func(env *protocol.Envelope) {
			_ = store.storeMessage(env)
		}(envelope)
	}
}

func (store *WakuStore) updateMetrics(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	defer store.wg.Done()

	for {
		select {
		case <-ticker.C:
			msgCount, err := store.msgProvider.Count()
			if err != nil {
				store.log.Error("updating store metrics", zap.Error(err))
			} else {
				metrics.RecordMessage(store.ctx, "stored", msgCount)
			}
		case <-store.quit:
			return
		}
	}
}

func (store *WakuStore) onRequest(s network.Stream) {
	defer s.Close()
	logger := store.log.With(logging.HostID("peer", s.Conn().RemotePeer()))
	historyRPCRequest := &pb.HistoryRPC{}

	writer := pbio.NewDelimitedWriter(s)
	reader := pbio.NewDelimitedReader(s, math.MaxInt32)

	err := reader.ReadMsg(historyRPCRequest)
	if err != nil {
		logger.Error("reading request", zap.Error(err))
		metrics.RecordStoreError(store.ctx, "decodeRPCFailure")
		return
	}

	logger = logger.With(zap.String("id", historyRPCRequest.RequestId))
	if query := historyRPCRequest.Query; query != nil {
		logger = logger.With(logging.Filters(query.GetContentFilters()))
	} else {
		logger.Error("reading request", zap.Error(err))
		metrics.RecordStoreError(store.ctx, "emptyRpcQueryFailure")
		return
	}

	logger.Info("received history query")
	metrics.RecordStoreQuery(store.ctx)

	historyResponseRPC := &pb.HistoryRPC{}
	historyResponseRPC.RequestId = historyRPCRequest.RequestId
	historyResponseRPC.Response = store.FindMessages(historyRPCRequest.Query)

	logger = logger.With(zap.Int("messages", len(historyResponseRPC.Response.Messages)))
	err = writer.WriteMsg(historyResponseRPC)
	if err != nil {
		logger.Error("writing response", zap.Error(err), logging.PagingInfo(historyResponseRPC.Response.PagingInfo))
		_ = s.Reset()
	} else {
		logger.Info("response sent")
	}
}

func (store *WakuStore) MessageChannel() chan *protocol.Envelope {
	return store.MsgC
}

// TODO: queryWithAccounting

// Stop closes the store message channel and removes the protocol stream handler
func (store *WakuStore) Stop() {
	store.started = false

	if store.MsgC != nil {
		close(store.MsgC)
	}

	if store.msgProvider != nil {
		store.msgProvider.Stop()
		store.quit <- struct{}{}
	}

	if store.h != nil {
		store.h.RemoveStreamHandler(StoreID_v20beta4)
	}

	store.wg.Wait()
}

func (store *WakuStore) queryLoop(ctx context.Context, query *pb.HistoryQuery, candidateList []peer.ID) ([]*wpb.WakuMessage, error) {
	// loops through the candidateList in order and sends the query to each until one of the query gets resolved successfully
	// returns the number of retrieved messages, or error if all the requests fail

	queryWg := sync.WaitGroup{}
	queryWg.Add(len(candidateList))

	resultChan := make(chan *pb.HistoryResponse, len(candidateList))

	for _, peer := range candidateList {
		func() {
			defer queryWg.Done()
			result, err := store.queryFrom(ctx, query, peer, protocol.GenerateRequestId())
			if err == nil {
				resultChan <- result
				return
			}
			store.log.Error("resuming history", logging.HostID("peer", peer), zap.Error(err))
		}()
	}

	queryWg.Wait()
	close(resultChan)

	var messages []*wpb.WakuMessage
	hasResults := false
	for result := range resultChan {
		hasResults = true
		messages = append(messages, result.Messages...)
	}

	if hasResults {
		return messages, nil
	}

	return nil, ErrFailedQuery
}

func (store *WakuStore) findLastSeen() (int64, error) {
	return store.msgProvider.MostRecentTimestamp()
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

// Resume retrieves the history of waku messages published on the default waku pubsub topic since the last time the waku store node has been online
// messages are stored in the store node's messages field and in the message db
// the offline time window is measured as the difference between the current time and the timestamp of the most recent persisted waku message
// an offset of 20 second is added to the time window to count for nodes asynchrony
// the history is fetched from one of the peers persisted in the waku store node's peer manager unit
// peerList indicates the list of peers to query from. The history is fetched from the first available peer in this list. Such candidates should be found through a discovery method (to be developed).
// if no peerList is passed, one of the peers in the underlying peer manager unit of the store protocol is picked randomly to fetch the history from. The history gets fetched successfully if the dialed peer has been online during the queried time window.
// the resume proc returns the number of retrieved messages if no error occurs, otherwise returns the error string
func (store *WakuStore) Resume(ctx context.Context, pubsubTopic string, peerList []peer.ID) (int, error) {
	if !store.started {
		return 0, errors.New("can't resume: store has not started")
	}

	lastSeenTime, err := store.findLastSeen()
	if err != nil {
		return 0, err
	}

	var offset int64 = int64(20 * time.Nanosecond)
	currentTime := store.timesource.Now().UnixNano() + offset
	lastSeenTime = max(lastSeenTime-offset, 0)

	rpc := &pb.HistoryQuery{
		PubsubTopic: pubsubTopic,
		StartTime:   lastSeenTime,
		EndTime:     currentTime,
		PagingInfo: &pb.PagingInfo{
			PageSize:  0,
			Direction: pb.PagingInfo_BACKWARD,
		},
	}

	if len(peerList) == 0 {
		return -1, ErrNoPeersAvailable
	}

	messages, err := store.queryLoop(ctx, rpc, peerList)
	if err != nil {
		store.log.Error("resuming history", zap.Error(err))
		return -1, ErrFailedToResumeHistory
	}

	msgCount := 0
	for _, msg := range messages {
		if err = store.storeMessage(protocol.NewEnvelope(msg, store.timesource.Now().UnixNano(), pubsubTopic)); err == nil {
			msgCount++
		}
	}

	store.log.Info("retrieved messages since the last online time", zap.Int("messages", len(messages)))

	return msgCount, nil
}
