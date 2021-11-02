package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"

	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
)

var log = logging.Logger("wakustore")

const StoreID_v20beta3 = libp2pProtocol.ID("/vac/waku/store/2.0.0-beta3")
const MaxPageSize = 100 // Maximum number of waku messages in each page

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

	if pageSize == 0 { // pageSize being zero indicates that no pagination is required
		return list, pinfo
	}

	if len(list) == 0 { // no pagination is needed for an empty list
		return list, &pb.PagingInfo{PageSize: 0, Cursor: pinfo.Cursor, Direction: pinfo.Direction}
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

func (w *WakuStore) FindMessages(query *pb.HistoryQuery) *pb.HistoryResponse {
	result := new(pb.HistoryResponse)
	// data holds IndexedWakuMessage whose topics match the query
	var data []IndexedWakuMessage
	for _, indexedMsg := range w.messages {
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
	StartTime     float64
	EndTime       float64
}

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
	ctx      context.Context
	MsgC     chan *protocol.Envelope
	messages []IndexedWakuMessage
	seen     map[[32]byte]struct{}

	started bool

	messagesMutex sync.Mutex

	msgProvider MessageProvider
	h           host.Host
}

func NewWakuStore(p MessageProvider) *WakuStore {
	wakuStore := new(WakuStore)
	wakuStore.msgProvider = p
	wakuStore.seen = make(map[[32]byte]struct{})

	return wakuStore
}

func (store *WakuStore) SetMsgProvider(p MessageProvider) {
	store.msgProvider = p
}

func (store *WakuStore) Start(ctx context.Context, h host.Host) {
	if store.started {
		return
	}

	store.started = true
	store.h = h
	store.ctx = ctx
	store.MsgC = make(chan *protocol.Envelope)

	store.h.SetStreamHandlerMatch(StoreID_v20beta3, protocol.PrefixTextMatch(string(StoreID_v20beta3)), store.onRequest)

	go store.storeIncomingMessages(ctx)

	if store.msgProvider == nil {
		log.Info("Store protocol started (no message provider)")
		return
	}

	store.fetchDBRecords(ctx)

	log.Info("Store protocol started")
}

func (store *WakuStore) fetchDBRecords(ctx context.Context) {
	if store.msgProvider == nil {
		return
	}

	storedMessages, err := (store.msgProvider).GetAll()
	if err != nil {
		log.Error("could not load DBProvider messages", err)
		metrics.RecordStoreError(ctx, "store_load_failure")
		return
	}

	for _, storedMessage := range storedMessages {
		idx := &pb.Index{
			Digest:       storedMessage.ID,
			ReceiverTime: float64(storedMessage.ReceiverTime),
		}

		store.storeMessageWithIndex(storedMessage.PubsubTopic, idx, storedMessage.Message)

		metrics.RecordMessage(ctx, "stored", len(store.messages))
	}
}

func (store *WakuStore) storeMessageWithIndex(pubsubTopic string, idx *pb.Index, msg *pb.WakuMessage) {
	var k [32]byte
	copy(k[:], idx.Digest)

	if _, ok := store.seen[k]; ok {
		return
	}

	store.seen[k] = struct{}{}
	store.messages = append(store.messages, IndexedWakuMessage{msg: msg, index: idx, pubsubTopic: pubsubTopic})
}

func (store *WakuStore) storeMessage(pubSubTopic string, msg *pb.WakuMessage) {
	index, err := computeIndex(msg)
	if err != nil {
		log.Error("could not calculate message index", err)
		return
	}

	store.messagesMutex.Lock()
	defer store.messagesMutex.Unlock()

	store.storeMessageWithIndex(pubSubTopic, index, msg)

	if store.msgProvider == nil {
		metrics.RecordMessage(store.ctx, "stored", len(store.messages))
		return
	}

	err = store.msgProvider.Put(index, pubSubTopic, msg) // Should the index be stored?

	if err != nil {
		log.Error("could not store message", err)
		metrics.RecordStoreError(store.ctx, "store_failure")
		return
	}

	metrics.RecordMessage(store.ctx, "stored", len(store.messages))
}

func (store *WakuStore) storeIncomingMessages(ctx context.Context) {
	for envelope := range store.MsgC {
		store.storeMessage(envelope.PubsubTopic(), envelope.Message())
	}
}

func (store *WakuStore) onRequest(s network.Stream) {
	defer s.Close()

	historyRPCRequest := &pb.HistoryRPC{}

	writer := protoio.NewDelimitedWriter(s)
	reader := protoio.NewDelimitedReader(s, 64*1024)

	err := reader.ReadMsg(historyRPCRequest)
	if err != nil {
		log.Error("error reading request", err)
		metrics.RecordStoreError(store.ctx, "decodeRPCFailure")
		return
	}

	log.Info(fmt.Sprintf("%s: Received query from %s", s.Conn().LocalPeer(), s.Conn().RemotePeer()))

	historyResponseRPC := &pb.HistoryRPC{}
	historyResponseRPC.RequestId = historyRPCRequest.RequestId
	historyResponseRPC.Response = store.FindMessages(historyRPCRequest.Query)

	err = writer.WriteMsg(historyResponseRPC)
	if err != nil {
		log.Error("error writing response", err)
		_ = s.Reset()
	} else {
		log.Info(fmt.Sprintf("%s: Response sent  to %s", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String()))
	}
}

func computeIndex(msg *pb.WakuMessage) (*pb.Index, error) {
	data, err := msg.Marshal()
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	return &pb.Index{
		Digest:       digest[:],
		ReceiverTime: utils.GetUnixEpoch(),
		SenderTime:   msg.Timestamp,
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

func WithPeer(p peer.ID) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		params.selectedPeer = p
	}
}

func WithAutomaticPeerSelection() HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		p, err := utils.SelectPeer(params.s.h, string(StoreID_v20beta3))
		if err == nil {
			params.selectedPeer = *p
		} else {
			log.Info("Error selecting peer: ", err)
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

func WithPaging(asc bool, pageSize uint64) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		params.asc = asc
		params.pageSize = pageSize
	}
}

func DefaultOptions() []HistoryRequestOption {
	return []HistoryRequestOption{
		WithAutomaticRequestId(),
		WithAutomaticPeerSelection(),
		WithPaging(true, 0),
	}
}

func (store *WakuStore) queryFrom(ctx context.Context, q *pb.HistoryQuery, selectedPeer peer.ID, requestId []byte) (*pb.HistoryResponse, error) {
	log.Info(fmt.Sprintf("Resuming message history with peer %s", selectedPeer))

	connOpt, err := store.h.NewStream(ctx, selectedPeer, StoreID_v20beta3)
	if err != nil {
		log.Error("Failed to connect to remote peer", err)
		return nil, err
	}

	defer connOpt.Close()
	defer func() {
		_ = connOpt.Reset()
	}()

	historyRequest := &pb.HistoryRPC{Query: q, RequestId: hex.EncodeToString(requestId)}

	writer := protoio.NewDelimitedWriter(connOpt)
	reader := protoio.NewDelimitedReader(connOpt, 64*1024)

	err = writer.WriteMsg(historyRequest)
	if err != nil {
		log.Error("could not write request", err)
		return nil, err
	}

	historyResponseRPC := &pb.HistoryRPC{}
	err = reader.ReadMsg(historyResponseRPC)
	if err != nil {
		log.Error("could not read response", err)
		metrics.RecordStoreError(store.ctx, "decodeRPCFailure")
		return nil, err
	}

	metrics.RecordMessage(ctx, "retrieved", len(store.messages))

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

func (store *WakuStore) queryLoop(ctx context.Context, query *pb.HistoryQuery, candidateList []peer.ID) (*pb.HistoryResponse, error) {
	// loops through the candidateList in order and sends the query to each until one of the query gets resolved successfully
	// returns the number of retrieved messages, or error if all the requests fail
	for _, peer := range candidateList {
		result, err := store.queryFrom(ctx, query, peer, protocol.GenerateRequestId())
		if err == nil {
			return result, nil
		}
		log.Error(fmt.Errorf("resume history with peer %s failed: %w", peer, err))
	}

	return nil, ErrFailedQuery
}

func (store *WakuStore) findLastSeen() float64 {
	var lastSeenTime float64 = 0
	for _, imsg := range store.messages {
		if imsg.msg.Timestamp > lastSeenTime {
			lastSeenTime = imsg.msg.Timestamp
		}
	}
	return lastSeenTime
}

// resume proc retrieves the history of waku messages published on the default waku pubsub topic since the last time the waku store node has been online
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

	var offset float64 = 200000
	currentTime = currentTime + offset
	lastSeenTime = math.Max(lastSeenTime-offset, 0)

	rpc := &pb.HistoryQuery{
		PubsubTopic: pubsubTopic,
		StartTime:   lastSeenTime,
		EndTime:     currentTime,
		PagingInfo: &pb.PagingInfo{
			PageSize:  0,
			Direction: pb.PagingInfo_BACKWARD,
		},
	}
	var response *pb.HistoryResponse
	if len(peerList) > 0 {
		var err error
		response, err = store.queryLoop(ctx, rpc, peerList)
		if err != nil {
			log.Error("failed to resume history", err)
			return -1, ErrFailedToResumeHistory
		}
	} else {
		p, err := utils.SelectPeer(store.h, string(StoreID_v20beta3))

		if err != nil {
			log.Info("Error selecting peer: ", err)
			return -1, ErrNoPeersAvailable
		}

		response, err = store.queryFrom(ctx, rpc, *p, protocol.GenerateRequestId())
		if err != nil {
			log.Error("failed to resume history", err)
			return -1, ErrFailedToResumeHistory
		}
	}

	for _, msg := range response.Messages {
		store.storeMessage(pubsubTopic, msg)
	}

	log.Info("Retrieved messages since the last online time: ", len(response.Messages))

	return len(response.Messages), nil
}

// TODO: queryWithAccounting

func (w *WakuStore) Stop() {
	w.started = false

	if w.MsgC != nil {
		close(w.MsgC)
	}

	if w.h != nil {
		w.h.RemoveStreamHandler(StoreID_v20beta3)
	}
}
