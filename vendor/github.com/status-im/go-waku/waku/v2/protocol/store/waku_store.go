package store

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"go.uber.org/zap"

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

var (
	ErrNoPeersAvailable      = errors.New("no suitable remote peers")
	ErrInvalidId             = errors.New("invalid request id")
	ErrFailedToResumeHistory = errors.New("failed to resume the history")
	ErrFailedQuery           = errors.New("failed to resolve the query")
)

func minOf(vars ...int) int {
	min := vars[0]

	for _, i := range vars {
		if min > i {
			min = i
		}
	}

	return min
}

func paginateWithIndex(list []IndexedWakuMessage, pinfo *pb.PagingInfo) (resMessages []IndexedWakuMessage, resPagingInfo *pb.PagingInfo) {
	if pinfo == nil {
		pinfo = new(pb.PagingInfo)
	}

	// takes list, and performs paging based on pinfo
	// returns the page i.e, a sequence of IndexedWakuMessage and the new paging info to be used for the next paging request
	cursor := pinfo.Cursor
	pageSize := pinfo.PageSize
	dir := pinfo.Direction

	if len(list) == 0 { // no pagination is needed for an empty list
		return list, &pb.PagingInfo{PageSize: 0, Cursor: pinfo.Cursor, Direction: pinfo.Direction}
	}

	if pageSize == 0 {
		pageSize = MaxPageSize
	}

	msgList := make([]IndexedWakuMessage, len(list))
	_ = copy(msgList, list) // makes a copy of the list

	sort.Slice(msgList, func(i, j int) bool { // sorts msgList based on the custom comparison proc indexedWakuMessageComparison
		return indexedWakuMessageComparison(msgList[i], msgList[j]) == -1
	})

	initQuery := false
	if cursor == nil {
		initQuery = true // an empty cursor means it is an initial query
		switch dir {
		case pb.PagingInfo_FORWARD:
			cursor = list[0].index // perform paging from the beginning of the list
		case pb.PagingInfo_BACKWARD:
			cursor = list[len(list)-1].index // perform paging from the end of the list
		}
	}

	foundIndex := findIndex(msgList, cursor)
	if foundIndex == -1 { // the cursor is not valid
		return nil, &pb.PagingInfo{PageSize: 0, Cursor: pinfo.Cursor, Direction: pinfo.Direction}
	}

	var retrievedPageSize, s, e int
	var newCursor *pb.Index // to be returned as part of the new paging info
	switch dir {
	case pb.PagingInfo_FORWARD: // forward pagination
		remainingMessages := len(msgList) - foundIndex - 1
		if initQuery {
			remainingMessages = remainingMessages + 1
			foundIndex = foundIndex - 1
		}
		// the number of queried messages cannot exceed the MaxPageSize and the total remaining messages i.e., msgList.len-foundIndex
		retrievedPageSize = minOf(int(pageSize), MaxPageSize, remainingMessages)
		s = foundIndex + 1 // non inclusive
		e = foundIndex + retrievedPageSize
		newCursor = msgList[e].index // the new cursor points to the end of the page
	case pb.PagingInfo_BACKWARD: // backward pagination
		remainingMessages := foundIndex
		if initQuery {
			remainingMessages = remainingMessages + 1
			foundIndex = foundIndex + 1
		}
		// the number of queried messages cannot exceed the MaxPageSize and the total remaining messages i.e., foundIndex-0
		retrievedPageSize = minOf(int(pageSize), MaxPageSize, remainingMessages)
		s = foundIndex - retrievedPageSize
		e = foundIndex - 1
		newCursor = msgList[s].index // the new cursor points to the beginning of the page
	}

	// retrieve the messages
	for i := s; i <= e; i++ {
		resMessages = append(resMessages, msgList[i])
	}
	resPagingInfo = &pb.PagingInfo{PageSize: uint64(retrievedPageSize), Cursor: newCursor, Direction: pinfo.Direction}

	return
}

func paginateWithoutIndex(list []IndexedWakuMessage, pinfo *pb.PagingInfo) (resMessages []*pb.WakuMessage, resPinfo *pb.PagingInfo) {
	// takes list, and performs paging based on pinfo
	// returns the page i.e, a sequence of WakuMessage and the new paging info to be used for the next paging request
	indexedData, updatedPagingInfo := paginateWithIndex(list, pinfo)
	for _, indexedMsg := range indexedData {
		resMessages = append(resMessages, indexedMsg.msg)
	}
	resPinfo = updatedPagingInfo
	return
}

func (store *WakuStore) FindMessages(query *pb.HistoryQuery) *pb.HistoryResponse {
	result := new(pb.HistoryResponse)
	// data holds IndexedWakuMessage whose topics match the query
	var data []IndexedWakuMessage
	for indexedMsg := range store.messageQueue.Messages() {
		// temporal filtering
		// check whether the history query contains a time filter
		if query.StartTime != 0 && query.EndTime != 0 {
			if indexedMsg.msg.Timestamp < query.StartTime || indexedMsg.msg.Timestamp > query.EndTime {
				continue
			}
		}

		// filter based on content filters
		// an empty list of contentFilters means no content filter is requested
		if len(query.ContentFilters) != 0 {
			match := false
			for _, cf := range query.ContentFilters {
				if cf.ContentTopic == indexedMsg.msg.ContentTopic {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// filter based on pubsub topic
		// an empty pubsub topic means no pubsub topic filter is requested
		if query.PubsubTopic != "" {
			if indexedMsg.pubsubTopic != query.PubsubTopic {
				continue
			}
		}

		// Some criteria matched
		data = append(data, indexedMsg)
	}

	result.Messages, result.PagingInfo = paginateWithoutIndex(data, query.PagingInfo)
	return result
}

type MessageProvider interface {
	GetAll() ([]persistence.StoredMessage, error)
	Put(cursor *pb.Index, pubsubTopic string, message *pb.WakuMessage) error
	Stop()
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

type IndexedWakuMessage struct {
	msg         *pb.WakuMessage
	index       *pb.Index
	pubsubTopic string
}

type WakuStore struct {
	ctx  context.Context
	MsgC chan *protocol.Envelope
	wg   *sync.WaitGroup

	log *zap.SugaredLogger

	started bool

	messageQueue *MessageQueue
	msgProvider  MessageProvider
	h            host.Host
	swap         *swap.WakuSwap
}

// NewWakuStore creates a WakuStore using an specific MessageProvider for storing the messages
func NewWakuStore(host host.Host, swap *swap.WakuSwap, p MessageProvider, maxNumberOfMessages int, maxRetentionDuration time.Duration, log *zap.SugaredLogger) *WakuStore {
	wakuStore := new(WakuStore)
	wakuStore.msgProvider = p
	wakuStore.h = host
	wakuStore.swap = swap
	wakuStore.wg = &sync.WaitGroup{}
	wakuStore.log = log.Named("store")
	wakuStore.messageQueue = NewMessageQueue(maxNumberOfMessages, maxRetentionDuration)
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

	store.started = true
	store.ctx = ctx
	store.MsgC = make(chan *protocol.Envelope, 1024)

	store.h.SetStreamHandlerMatch(StoreID_v20beta4, protocol.PrefixTextMatch(string(StoreID_v20beta4)), store.onRequest)

	store.wg.Add(1)
	go store.storeIncomingMessages(ctx)

	if store.msgProvider == nil {
		store.log.Info("Store protocol started (no message provider)")
		return
	}

	store.fetchDBRecords(ctx)

	store.log.Info("Store protocol started")
}

func (store *WakuStore) fetchDBRecords(ctx context.Context) {
	if store.msgProvider == nil {
		return
	}

	storedMessages, err := (store.msgProvider).GetAll()
	if err != nil {
		store.log.Error("could not load DBProvider messages", err)
		metrics.RecordStoreError(ctx, "store_load_failure")
		return
	}

	for _, storedMessage := range storedMessages {
		idx := &pb.Index{
			Digest:       storedMessage.ID,
			ReceiverTime: storedMessage.ReceiverTime,
		}

		_ = store.addToMessageQueue(storedMessage.PubsubTopic, idx, storedMessage.Message)

		metrics.RecordMessage(ctx, "stored", store.messageQueue.Length())
	}
}

func (store *WakuStore) addToMessageQueue(pubsubTopic string, idx *pb.Index, msg *pb.WakuMessage) error {
	return store.messageQueue.Push(IndexedWakuMessage{msg: msg, index: idx, pubsubTopic: pubsubTopic})
}

func (store *WakuStore) storeMessage(env *protocol.Envelope) error {
	index, err := computeIndex(env)
	if err != nil {
		store.log.Error("could not calculate message index", err)
		return err
	}

	err = store.addToMessageQueue(env.PubsubTopic(), index, env.Message())
	if err == ErrDuplicatedMessage {
		return err
	}

	if store.msgProvider == nil {
		metrics.RecordMessage(store.ctx, "stored", store.messageQueue.Length())
		return err
	}

	// TODO: Move this to a separate go routine if DB writes becomes a bottleneck
	err = store.msgProvider.Put(index, env.PubsubTopic(), env.Message()) // Should the index be stored?
	if err != nil {
		store.log.Error("could not store message", err)
		metrics.RecordStoreError(store.ctx, "store_failure")
		return err
	}

	metrics.RecordMessage(store.ctx, "stored", store.messageQueue.Length())
	return nil
}

func (store *WakuStore) storeIncomingMessages(ctx context.Context) {
	defer store.wg.Done()
	for envelope := range store.MsgC {
		_ = store.storeMessage(envelope)
	}
}

func (store *WakuStore) onRequest(s network.Stream) {
	defer s.Close()

	historyRPCRequest := &pb.HistoryRPC{}

	writer := protoio.NewDelimitedWriter(s)
	reader := protoio.NewDelimitedReader(s, math.MaxInt32)

	err := reader.ReadMsg(historyRPCRequest)
	if err != nil {
		store.log.Error("error reading request", err)
		metrics.RecordStoreError(store.ctx, "decodeRPCFailure")
		return
	}

	store.log.Info(fmt.Sprintf("%s: Received query from %s", s.Conn().LocalPeer(), s.Conn().RemotePeer()))

	historyResponseRPC := &pb.HistoryRPC{}
	historyResponseRPC.RequestId = historyRPCRequest.RequestId
	historyResponseRPC.Response = store.FindMessages(historyRPCRequest.Query)

	err = writer.WriteMsg(historyResponseRPC)
	if err != nil {
		store.log.Error("error writing response", err)
		_ = s.Reset()
	} else {
		store.log.Info(fmt.Sprintf("%s: Response sent  to %s", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String()))
	}
}

func computeIndex(env *protocol.Envelope) (*pb.Index, error) {
	return &pb.Index{
		Digest:       env.Hash(),
		ReceiverTime: utils.GetUnixEpoch(),
		SenderTime:   env.Message().Timestamp,
	}, nil
}

func indexComparison(x, y *pb.Index) int {
	// compares x and y
	// returns 0 if they are equal
	// returns -1 if x < y
	// returns 1 if x > y

	var timecmp int = 0
	if x.SenderTime > y.SenderTime {
		timecmp = 1
	} else if x.SenderTime < y.SenderTime {
		timecmp = -1
	}

	digestcm := bytes.Compare(x.Digest, y.Digest)
	if timecmp != 0 {
		return timecmp // timestamp has a higher priority for comparison
	}

	return digestcm
}

func indexedWakuMessageComparison(x, y IndexedWakuMessage) int {
	// compares x and y
	// returns 0 if they are equal
	// returns -1 if x < y
	// returns 1 if x > y
	return indexComparison(x.index, y.index)
}

func findIndex(msgList []IndexedWakuMessage, index *pb.Index) int {
	// returns the position of an IndexedWakuMessage in msgList whose index value matches the given index
	// returns -1 if no match is found
	for i, indexedWakuMessage := range msgList {
		if bytes.Equal(indexedWakuMessage.index.Digest, index.Digest) && indexedWakuMessage.index.ReceiverTime == index.ReceiverTime {
			return i
		}
	}
	return -1
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

// WithAutomaticPeerSelection is an option used to randomly select a peer from the store
// to request the message history
func WithAutomaticPeerSelection() HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		p, err := utils.SelectPeer(params.s.h, string(StoreID_v20beta4), params.s.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.s.log.Info("Error selecting peer: ", err)
		}
	}
}

func WithFastestPeerSelection(ctx context.Context) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.s.h, string(StoreID_v20beta4), params.s.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.s.log.Info("Error selecting peer: ", err)
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
	store.log.Info(fmt.Sprintf("Querying message history with peer %s", selectedPeer))

	connOpt, err := store.h.NewStream(ctx, selectedPeer, StoreID_v20beta4)
	if err != nil {
		store.log.Error("Failed to connect to remote peer", err)
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
		store.log.Error("could not write request", err)
		return nil, err
	}

	historyResponseRPC := &pb.HistoryRPC{}
	err = reader.ReadMsg(historyResponseRPC)
	if err != nil {
		store.log.Error("could not read response", err)
		metrics.RecordStoreError(store.ctx, "decodeRPCFailure")
		return nil, err
	}

	metrics.RecordMessage(ctx, "retrieved", store.messageQueue.Length())

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
		PubsubTopic:    r.query.PubsubTopic,
		ContentFilters: r.query.ContentFilters,
		StartTime:      r.query.StartTime,
		EndTime:        r.query.EndTime,
		PagingInfo: &pb.PagingInfo{
			PageSize:  r.query.PagingInfo.PageSize,
			Direction: r.query.PagingInfo.Direction,
			Cursor: &pb.Index{
				Digest:       r.cursor.Digest,
				ReceiverTime: r.cursor.ReceiverTime,
				SenderTime:   r.cursor.SenderTime,
			},
		},
	}

	response, err := store.queryFrom(ctx, q, r.peerId, protocol.GenerateRequestId())
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
		peerId:   r.peerId,
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
			store.log.Error(fmt.Errorf("resume history with peer %s failed: %w", peer, err))
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

func (store *WakuStore) findLastSeen() int64 {
	var lastSeenTime int64 = 0
	for imsg := range store.messageQueue.Messages() {
		if imsg.msg.Timestamp > lastSeenTime {
			lastSeenTime = imsg.msg.Timestamp
		}
	}
	return lastSeenTime
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
	lastSeenTime := store.findLastSeen()

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
			store.log.Info("Error selecting peer: ", err)
			return -1, ErrNoPeersAvailable
		}

		peerList = append(peerList, *p)
	}

	messages, err := store.queryLoop(ctx, rpc, peerList)
	if err != nil {
		store.log.Error("failed to resume history", err)
		return -1, ErrFailedToResumeHistory
	}

	msgCount := 0
	for _, msg := range messages {
		if err = store.storeMessage(protocol.NewEnvelope(msg, pubsubTopic)); err == nil {
			msgCount++
		}
	}

	store.log.Info("Retrieved messages since the last online time: ", len(messages))

	return msgCount, nil
}

// TODO: queryWithAccounting

// Stop closes the store message channel and removes the protocol stream handler
func (store *WakuStore) Stop() {
	store.started = false

	if store.MsgC != nil {
		close(store.MsgC)
	}

	if store.h != nil {
		store.h.RemoveStreamHandler(StoreID_v20beta4)
	}

	store.wg.Wait()
}
