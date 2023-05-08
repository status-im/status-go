package store

import (
	"context"
	"encoding/hex"
	"errors"
	"math"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio/pbio"
	"go.uber.org/zap"

	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

type Query struct {
	Topic         string
	ContentTopics []string
	StartTime     int64
	EndTime       int64
}

// Result represents a valid response from a store node
type Result struct {
	started  bool
	Messages []*wpb.WakuMessage
	store    Store
	query    *pb.HistoryQuery
	cursor   *pb.Index
	peerId   peer.ID
}

func (r *Result) Cursor() *pb.Index {
	return r.cursor
}

func (r *Result) IsComplete() bool {
	return r.cursor == nil
}

func (r *Result) PeerID() peer.ID {
	return r.peerId
}

func (r *Result) Query() *pb.HistoryQuery {
	return r.query
}

func (r *Result) Next(ctx context.Context) (bool, error) {
	if !r.started {
		r.started = true
		return len(r.Messages) != 0, nil
	}

	if r.IsComplete() {
		return false, nil
	}

	newResult, err := r.store.Next(ctx, r)
	if err != nil {
		return false, err
	}

	r.cursor = newResult.cursor
	r.Messages = newResult.Messages

	return true, nil
}

func (r *Result) GetMessages() []*wpb.WakuMessage {
	if !r.started {
		return nil
	}
	return r.Messages
}

type criteriaFN = func(msg *wpb.WakuMessage) (bool, error)

type HistoryRequestParameters struct {
	selectedPeer peer.ID
	localQuery   bool
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
// to request the message history. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithAutomaticPeerSelection(fromThesePeers ...peer.ID) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		p, err := utils.SelectPeer(params.s.h, StoreID_v20beta4, fromThesePeers, params.s.log)
		if err == nil {
			params.selectedPeer = p
		} else {
			params.s.log.Info("selecting peer", zap.Error(err))
		}
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithFastestPeerSelection(ctx context.Context, fromThesePeers ...peer.ID) HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.s.h, StoreID_v20beta4, fromThesePeers, params.s.log)
		if err == nil {
			params.selectedPeer = p
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

func WithLocalQuery() HistoryRequestOption {
	return func(params *HistoryRequestParameters) {
		params.localQuery = true
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
		metrics.RecordStoreError(store.ctx, "dial_failure")
		return nil, err
	}

	connOpt, err := store.h.NewStream(ctx, selectedPeer, StoreID_v20beta4)
	if err != nil {
		logger.Error("creating stream to peer", zap.Error(err))
		metrics.RecordStoreError(store.ctx, "dial_failure")
		return nil, err
	}

	defer connOpt.Close()
	defer func() {
		_ = connOpt.Reset()
	}()

	historyRequest := &pb.HistoryRPC{Query: q, RequestId: hex.EncodeToString(requestId)}

	writer := pbio.NewDelimitedWriter(connOpt)
	reader := pbio.NewDelimitedReader(connOpt, math.MaxInt32)

	err = writer.WriteMsg(historyRequest)
	if err != nil {
		logger.Error("writing request", zap.Error(err))
		metrics.RecordStoreError(store.ctx, "write_request_failure")
		return nil, err
	}

	historyResponseRPC := &pb.HistoryRPC{RequestId: historyRequest.RequestId}
	err = reader.ReadMsg(historyResponseRPC)
	if err != nil {
		logger.Error("reading response", zap.Error(err))
		metrics.RecordStoreError(store.ctx, "decode_rpc_failure")
		return nil, err
	}

	if historyResponseRPC.Response == nil {
		// Empty response
		return &pb.HistoryResponse{
			PagingInfo: &pb.PagingInfo{},
		}, nil
	}

	return historyResponseRPC.Response, nil
}

func (store *WakuStore) localQuery(query *pb.HistoryQuery, requestId []byte) (*pb.HistoryResponse, error) {
	logger := store.log
	logger.Info("querying local message history")

	if !store.started {
		return nil, errors.New("not running local store")
	}

	historyResponseRPC := &pb.HistoryRPC{
		RequestId: hex.EncodeToString(requestId),
		Response:  store.FindMessages(query),
	}

	if historyResponseRPC.Response == nil {
		// Empty response
		return &pb.HistoryResponse{
			PagingInfo: &pb.PagingInfo{},
		}, nil
	}

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

	if len(q.ContentFilters) > MaxContentFilters {
		return nil, ErrMaxContentFilters
	}

	params := new(HistoryRequestParameters)
	params.s = store

	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	if !params.localQuery && params.selectedPeer == "" {
		metrics.RecordStoreError(ctx, "peer_not_found_failure")
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

	pageSize := params.pageSize
	if pageSize == 0 || pageSize > uint64(MaxPageSize) {
		pageSize = MaxPageSize
	}
	q.PagingInfo.PageSize = pageSize

	var response *pb.HistoryResponse
	var err error

	if params.localQuery {
		response, err = store.localQuery(q, params.requestId)
	} else {
		response, err = store.queryFrom(ctx, q, params.selectedPeer, params.requestId)
	}
	if err != nil {
		return nil, err
	}

	if response.Error == pb.HistoryResponse_INVALID_CURSOR {
		return nil, errors.New("invalid cursor")
	}

	result := &Result{
		store:    store,
		Messages: response.Messages,
		query:    q,
		peerId:   params.selectedPeer,
	}

	if response.PagingInfo != nil {
		result.cursor = response.PagingInfo.Cursor
	}

	return result, nil
}

// Find the first message that matches a criteria. criteriaCB is a function that will be invoked for each message and returns true if the message matches the criteria
func (store *WakuStore) Find(ctx context.Context, query Query, cb criteriaFN, opts ...HistoryRequestOption) (*wpb.WakuMessage, error) {
	if cb == nil {
		return nil, errors.New("callback can't be null")
	}

	result, err := store.Query(ctx, query, opts...)
	if err != nil {
		return nil, err
	}

	for {
		for _, m := range result.Messages {
			found, err := cb(m)
			if err != nil {
				return nil, err
			}

			if found {
				return m, nil
			}
		}

		if result.IsComplete() {
			break
		}

		result, err = store.Next(ctx, result)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// Next is used with to retrieve the next page of rows from a query response.
// If no more records are found, the result will not contain any messages.
// This function is useful for iterating over results without having to manually
// specify the cursor and pagination order and max number of results
func (store *WakuStore) Next(ctx context.Context, r *Result) (*Result, error) {
	if r.IsComplete() {
		return &Result{
			store:    store,
			started:  true,
			Messages: []*wpb.WakuMessage{},
			cursor:   nil,
			query:    r.query,
			peerId:   r.PeerID(),
		}, nil
	}

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

	result := &Result{
		started:  true,
		store:    store,
		Messages: response.Messages,
		query:    q,
		peerId:   r.PeerID(),
	}

	if response.PagingInfo != nil {
		result.cursor = response.PagingInfo.Cursor
	}

	return result, nil

}
