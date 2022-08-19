package store

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"go.uber.org/zap"

	"github.com/status-im/go-waku/logging"
	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/swap"
	"github.com/status-im/go-waku/waku/v2/utils"
)

// StoreID_v20beta4 is the current Waku Store protocol identifier
const StoreID_v20beta4 = libp2pProtocol.ID("/vac/waku/store/2.0.0-beta4")

// MaxPageSize is the maximum number of waku messages to return per page
const MaxPageSize = 100

// MaxTimeVariance is the maximum duration in the future allowed for a message timestamp
const MaxTimeVariance = time.Duration(20) * time.Second

var (
	// ErrNoPeersAvailable is returned when there are no store peers in the peer store
	// that could be used to retrieve message history
	ErrNoPeersAvailable = errors.New("no suitable remote peers")

	// ErrInvalidId is returned when no RequestID is given
	ErrInvalidId = errors.New("invalid request id")

	// ErrFailedToResumeHistory is returned when the node attempted to retrieve historic
	// messages to fill its own message history but for some reason it failed
	ErrFailedToResumeHistory = errors.New("failed to resume the history")

	// ErrFailedQuery is emitted when the query fails to return results
	ErrFailedQuery = errors.New("failed to resolve the query")

	ErrFutureMessage = errors.New("message timestamp in the future")
)

func findMessages(query *pb.HistoryQuery, msgProvider MessageProvider) ([]*pb.WakuMessage, *pb.PagingInfo, error) {
	if query.PagingInfo == nil {
		query.PagingInfo = &pb.PagingInfo{
			Direction: pb.PagingInfo_FORWARD,
		}
	}

	if query.PagingInfo.PageSize == 0 || query.PagingInfo.PageSize > uint64(MaxPageSize) {
		query.PagingInfo.PageSize = MaxPageSize
	}

	queryResult, err := msgProvider.Query(query)
	if err != nil {
		return nil, nil, err
	}

	if len(queryResult) == 0 { // no pagination is needed for an empty list
		newPagingInfo := &pb.PagingInfo{PageSize: 0, Cursor: query.PagingInfo.Cursor, Direction: query.PagingInfo.Direction}
		return nil, newPagingInfo, nil
	}

	lastMsgIdx := len(queryResult) - 1
	newCursor := protocol.NewEnvelope(queryResult[lastMsgIdx].Message, queryResult[lastMsgIdx].ReceiverTime, queryResult[lastMsgIdx].PubsubTopic).Index()

	newPagingInfo := &pb.PagingInfo{PageSize: query.PagingInfo.PageSize, Cursor: newCursor, Direction: query.PagingInfo.Direction}
	if newPagingInfo.PageSize > uint64(len(queryResult)) {
		newPagingInfo.PageSize = uint64(len(queryResult))
	}

	resultMessages := make([]*pb.WakuMessage, len(queryResult))
	for i := range queryResult {
		resultMessages[i] = queryResult[i].Message
	}

	return resultMessages, newPagingInfo, nil
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
	Query(query *pb.HistoryQuery) ([]persistence.StoredMessage, error)
	Put(env *protocol.Envelope) error
	MostRecentTimestamp() (int64, error)
	Stop()
	Count() (int, error)
}
type Query struct {
	Topic         string
	ContentTopics []string
	StartTime     int64
	EndTime       int64
}

// Result represents a valid response from a store node
type Result struct {
	Messages []*pb.WakuMessage

	query  *pb.HistoryQuery
	cursor *pb.Index
	peerId peer.ID
}

func (r *Result) Cursor() *pb.Index {
	return r.cursor
}

func (r *Result) PeerID() peer.ID {
	return r.peerId
}

func (r *Result) Query() *pb.HistoryQuery {
	return r.query
}

type WakuStore struct {
	ctx  context.Context
	MsgC chan *protocol.Envelope
	wg   *sync.WaitGroup

	log *zap.Logger

	started bool
	quit    chan struct{}

	msgProvider MessageProvider
	h           host.Host
	swap        *swap.WakuSwap
}

type Store interface {
	Start(ctx context.Context)
	Query(ctx context.Context, query Query, opts ...HistoryRequestOption) (*Result, error)
	Next(ctx context.Context, r *Result) (*Result, error)
	Resume(ctx context.Context, pubsubTopic string, peerList []peer.ID) (int, error)
	MessageChannel() chan *protocol.Envelope
	Stop()
}

// NewWakuStore creates a WakuStore using an specific MessageProvider for storing the messages
func NewWakuStore(host host.Host, swap *swap.WakuSwap, p MessageProvider, log *zap.Logger) *WakuStore {
	wakuStore := new(WakuStore)
	wakuStore.msgProvider = p
	wakuStore.h = host
	wakuStore.swap = swap
	wakuStore.wg = &sync.WaitGroup{}
	wakuStore.log = log.Named("store")
	wakuStore.quit = make(chan struct{})

	return wakuStore
}

// SetMessageProvider allows switching the message provider used with a WakuStore
func (store *WakuStore) SetMessageProvider(p MessageProvider) {
	store.msgProvider = p
}

// Start initializes the WakuStore by enabling the protocol and fetching records from a message provider
func (store *WakuStore) Start(ctx context.Context) {
	if store.started {
		return
	}

	if store.msgProvider == nil {
		store.log.Info("Store protocol started (no message provider)")
		return
	}

	store.started = true
	store.ctx = ctx
	store.MsgC = make(chan *protocol.Envelope, 1024)

	store.h.SetStreamHandlerMatch(StoreID_v20beta4, protocol.PrefixTextMatch(string(StoreID_v20beta4)), store.onRequest)

	store.wg.Add(2)
	go store.storeIncomingMessages(ctx)
	go store.updateMetrics(ctx)

	store.log.Info("Store protocol started")
}

func (store *WakuStore) storeMessage(env *protocol.Envelope) error {
	// Ensure that messages don't "jump" to the front of the queue with future timestamps
	if env.Index().SenderTime-env.Index().ReceiverTime > int64(MaxTimeVariance) {
		return ErrFutureMessage
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
		_ = store.storeMessage(envelope)
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

	writer := protoio.NewDelimitedWriter(s)
	reader := protoio.NewDelimitedReader(s, math.MaxInt32)

	err := reader.ReadMsg(historyRPCRequest)
	if err != nil {
		logger.Error("reading request", zap.Error(err))
		metrics.RecordStoreError(store.ctx, "decodeRPCFailure")
		return
	}
	logger = logger.With(zap.String("id", historyRPCRequest.RequestId))
	if query := historyRPCRequest.Query; query != nil {
		logger = logger.With(logging.Filters(query.GetContentFilters()))
	}
	logger.Info("received query")

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

type HistoryRequestParameters struct {
	selectedPeer peer.ID
	requestId    []byte
	cursor       *pb.Index
	pageSize     uint64
	asc          bool

	s *WakuStore
}

type HistoryRequestOption func(*HistoryRequestParameters)

// WithPeer is an option used to specify the peerID to request the message history
func WithPeer(p peer.ID) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		params.selectedPeer = p
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the peer store
// to request the message history
func WithAutomaticPeerSelection() HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		p, err := utils.SelectPeer(params.s.h, string(StoreID_v20beta4), params.s.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.s.log.Info("selecting peer", zap.Error(err))
		}
	}
}

func WithFastestPeerSelection(ctx context.Context) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.s.h, string(StoreID_v20beta4), params.s.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.s.log.Info("selecting peer", zap.Error(err))
		}
	}
}

func WithRequestId(requestId []byte) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		params.requestId = requestId
	}
}

func WithAutomaticRequestId() HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		params.requestId = protocol.GenerateRequestId()
	}
}

func WithCursor(c *pb.Index) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		params.cursor = c
	}
}

// WithPaging is an option used to specify the order and maximum number of records to return
func WithPaging(asc bool, pageSize uint64) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		params.asc = asc
		params.pageSize = pageSize
	}
}

// Default options to be used when querying a store node for results
func DefaultOptions() []HistoryRequestOption {
	return []HistoryRequestOption{
		WithAutomaticRequestId(),
		WithAutomaticPeerSelection(),
		WithPaging(true, MaxPageSize),
	}
}

func (store *WakuStore) queryFrom(ctx context.Context, q *pb.HistoryQuery, selectedPeer peer.ID, requestId []byte) (*pb.HistoryResponse, error) {
	logger := store.log.With(logging.HostID("peer", selectedPeer))
	logger.Info("querying message history")

	// We connect first so dns4 addresses are resolved (NewStream does not do it)
	err := store.h.Connect(ctx, store.h.Peerstore().PeerInfo(selectedPeer))
	if err != nil {
		logger.Error("connecting to peer", zap.Error(err))
		return nil, err
	}

	connOpt, err := store.h.NewStream(ctx, selectedPeer, StoreID_v20beta4)
	if err != nil {
		logger.Error("creating stream to peer", zap.Error(err))
		return nil, err
	}

	defer connOpt.Close()
	defer func() {
		_ = connOpt.Reset()
	}()

	historyRequest := &pb.HistoryRPC{Query: q, RequestId: hex.EncodeToString(requestId)}

	writer := protoio.NewDelimitedWriter(connOpt)
	reader := protoio.NewDelimitedReader(connOpt, math.MaxInt32)

	err = writer.WriteMsg(historyRequest)
	if err != nil {
		logger.Error("writing request", zap.Error(err))
		return nil, err
	}

	historyResponseRPC := &pb.HistoryRPC{}
	err = reader.ReadMsg(historyResponseRPC)
	if err != nil {
		logger.Error("reading response", zap.Error(err))
		metrics.RecordStoreError(store.ctx, "decodeRPCFailure")
		return nil, err
	}

	if historyResponseRPC.Response == nil {
		historyResponseRPC.Response = new(pb.HistoryResponse)
		historyResponseRPC.Response.PagingInfo = new(pb.PagingInfo)
		historyResponseRPC.Response.PagingInfo.Cursor = new(pb.Index)
	}

	metrics.RecordMessage(ctx, "retrieved", len(historyResponseRPC.Response.Messages))

	return historyResponseRPC.Response, nil
}

func (store *WakuStore) Query(ctx context.Context, query Query, opts ...HistoryRequestOption) (*Result, error) {
	q := &pb.HistoryQuery{
		PubsubTopic:    query.Topic,
		ContentFilters: []*pb.ContentFilter{},
		StartTime:      query.StartTime,
		EndTime:        query.EndTime,
		PagingInfo:     &pb.PagingInfo{},
	}

	for _, cf := range query.ContentTopics {
		q.ContentFilters = append(q.ContentFilters, &pb.ContentFilter{ContentTopic: cf})
	}

	params := new(HistoryRequestParameters)
	params.s = store

	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	if params.selectedPeer == "" {
		return nil, ErrNoPeersAvailable
	}

	if len(params.requestId) == 0 {
		return nil, ErrInvalidId
	}

	if params.cursor != nil {
		q.PagingInfo.Cursor = params.cursor
	}

	if params.asc {
		q.PagingInfo.Direction = pb.PagingInfo_FORWARD
	} else {
		q.PagingInfo.Direction = pb.PagingInfo_BACKWARD
	}

	q.PagingInfo.PageSize = params.pageSize

	response, err := store.queryFrom(ctx, q, params.selectedPeer, params.requestId)
	if err != nil {
		return nil, err
	}

	if response.Error == pb.HistoryResponse_INVALID_CURSOR {
		return nil, errors.New("invalid cursor")
	}

	return &Result{
		Messages: response.Messages,
		cursor:   response.PagingInfo.Cursor,
		query:    q,
		peerId:   params.selectedPeer,
	}, nil
}

// Next is used with to retrieve the next page of rows from a query response.
// If no more records are found, the result will not contain any messages.
// This function is useful for iterating over results without having to manually
// specify the cursor and pagination order and max number of results
func (store *WakuStore) Next(ctx context.Context, r *Result) (*Result, error) {
	q := &pb.HistoryQuery{
		PubsubTopic:    r.Query().PubsubTopic,
		ContentFilters: r.Query().ContentFilters,
		StartTime:      r.Query().StartTime,
		EndTime:        r.Query().EndTime,
		PagingInfo: &pb.PagingInfo{
			PageSize:  r.Query().PagingInfo.PageSize,
			Direction: r.Query().PagingInfo.Direction,
			Cursor: &pb.Index{
				Digest:       r.Cursor().Digest,
				ReceiverTime: r.Cursor().ReceiverTime,
				SenderTime:   r.Cursor().SenderTime,
				PubsubTopic:  r.Cursor().PubsubTopic,
			},
		},
	}

	response, err := store.queryFrom(ctx, q, r.PeerID(), protocol.GenerateRequestId())
	if err != nil {
		return nil, err
	}

	if response.Error == pb.HistoryResponse_INVALID_CURSOR {
		return nil, errors.New("invalid cursor")
	}

	return &Result{
		Messages: response.Messages,
		cursor:   response.PagingInfo.Cursor,
		query:    q,
		peerId:   r.PeerID(),
	}, nil
}

func (store *WakuStore) queryLoop(ctx context.Context, query *pb.HistoryQuery, candidateList []peer.ID) ([]*pb.WakuMessage, error) {
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

	var messages []*pb.WakuMessage
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

	currentTime := utils.GetUnixEpoch()
	lastSeenTime, err := store.findLastSeen()
	if err != nil {
		return 0, err
	}

	var offset int64 = int64(20 * time.Nanosecond)
	currentTime = currentTime + offset
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
		p, err := utils.SelectPeer(store.h, string(StoreID_v20beta4), store.log)
		if err != nil {
			store.log.Info("selecting peer", zap.Error(err))
			return -1, ErrNoPeersAvailable
		}

		peerList = append(peerList, *p)
	}

	messages, err := store.queryLoop(ctx, rpc, peerList)
	if err != nil {
		store.log.Error("resuming history", zap.Error(err))
		return -1, ErrFailedToResumeHistory
	}

	msgCount := 0
	for _, msg := range messages {
		if err = store.storeMessage(protocol.NewEnvelope(msg, utils.GetUnixEpoch(), pubsubTopic)); err == nil {
			msgCount++
		}
	}

	store.log.Info("retrieved messages since the last online time", zap.Int("messages", len(messages)))

	return msgCount, nil
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
		store.quit <- struct{}{}
	}

	if store.h != nil {
		store.h.RemoveStreamHandler(StoreID_v20beta4)
	}

	store.wg.Wait()
}
