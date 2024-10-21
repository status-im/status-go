package store

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

// StoreQueryID_v300 is the Store protocol v3 identifier
const StoreQueryID_v300 = libp2pProtocol.ID("/vac/waku/store-query/3.0.0")
const StoreENRField = uint8(1 << 1)

// MaxPageSize is the maximum number of waku messages to return per page
const MaxPageSize = 100

// DefaultPageSize is the default number of waku messages per page
const DefaultPageSize = 20

const ok = uint32(200)

var (

	// ErrNoPeersAvailable is returned when there are no store peers in the peer store
	// that could be used to retrieve message history
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrMustSelectPeer   = errors.New("a peer ID or multiaddress is required when checking for message hashes")
)

// StoreError represents an error code returned by a storenode
type StoreError struct {
	Code    int
	Message string
}

// NewStoreError creates a new instance of StoreError
func NewStoreError(code int, message string) *StoreError {
	return &StoreError{
		Code:    code,
		Message: message,
	}
}

const errorStringFmt = "%d - %s"

// Error returns a string with the error message
func (e *StoreError) Error() string {
	return fmt.Sprintf(errorStringFmt, e.Code, e.Message)
}

// WakuStore represents an instance of a store client
type WakuStore struct {
	h          host.Host
	timesource timesource.Timesource
	log        *zap.Logger
	pm         *peermanager.PeerManager

	defaultRatelimit rate.Limit
	rateLimiters     map[peer.ID]*rate.Limiter
}

// NewWakuStore is used to instantiate a StoreV3 client
func NewWakuStore(pm *peermanager.PeerManager, timesource timesource.Timesource, log *zap.Logger, defaultRatelimit rate.Limit) *WakuStore {
	s := new(WakuStore)
	s.log = log.Named("store-client")
	s.timesource = timesource
	s.pm = pm
	s.defaultRatelimit = defaultRatelimit
	s.rateLimiters = make(map[peer.ID]*rate.Limiter)

	if pm != nil {
		pm.RegisterWakuProtocol(StoreQueryID_v300, StoreENRField)
	}

	return s
}

// Sets the host to be able to mount or consume a protocol
func (s *WakuStore) SetHost(h host.Host) {
	s.h = h
}

// Request is used to send a store query. This function requires understanding how to prepare a store query
// and most of the time you can use `Query`, `QueryByHash` and `Exists` instead, as they provide
// a simpler API
func (s *WakuStore) Request(ctx context.Context, criteria Criteria, opts ...RequestOption) (Result, error) {
	params := new(Parameters)

	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	filterCriteria, isFilterCriteria := criteria.(FilterCriteria)

	var pubsubTopics []string
	if isFilterCriteria {
		pubsubTopics = append(pubsubTopics, filterCriteria.PubsubTopic)
	}

	//Add Peer to peerstore.
	if s.pm != nil && params.peerAddr != nil {
		pData, err := s.pm.AddPeer(params.peerAddr, peerstore.Static, pubsubTopics, StoreQueryID_v300)
		if err != nil {
			return nil, err
		}
		s.pm.Connect(pData)
		params.selectedPeer = pData.AddrInfo.ID
	}

	if s.pm != nil && params.selectedPeer == "" {
		if isFilterCriteria {
			selectedPeers, err := s.pm.SelectPeers(
				peermanager.PeerSelectionCriteria{
					SelectionType: params.peerSelectionType,
					Proto:         StoreQueryID_v300,
					PubsubTopics:  []string{filterCriteria.PubsubTopic},
					SpecificPeers: params.preferredPeers,
					Ctx:           ctx,
				},
			)
			if err != nil {
				return nil, err
			}
			params.selectedPeer = selectedPeers[0]
		} else {
			return nil, ErrMustSelectPeer
		}
	}

	if params.selectedPeer == "" {
		return nil, ErrNoPeersAvailable
	}

	pageLimit := params.pageLimit
	if pageLimit == 0 {
		pageLimit = DefaultPageSize
	} else if pageLimit > uint64(MaxPageSize) {
		pageLimit = MaxPageSize
	}

	storeRequest := &pb.StoreQueryRequest{
		RequestId:         hex.EncodeToString(params.requestID),
		IncludeData:       params.includeData,
		PaginationForward: params.forward,
		PaginationLimit:   proto.Uint64(pageLimit),
	}

	criteria.PopulateStoreRequest(storeRequest)

	if params.cursor != nil {
		storeRequest.PaginationCursor = params.cursor
	}

	err := storeRequest.Validate()
	if err != nil {
		return nil, err
	}

	response, err := s.queryFrom(ctx, storeRequest, params)
	if err != nil {
		return nil, err
	}

	result := &resultImpl{
		store:         s,
		messages:      response.Messages,
		storeRequest:  storeRequest,
		storeResponse: response,
		peerID:        params.selectedPeer,
		cursor:        response.PaginationCursor,
	}

	return result, nil
}

// Query retrieves all the messages that match a criteria. Use the options to indicate whether to return the message themselves or not.
func (s *WakuStore) Query(ctx context.Context, criteria FilterCriteria, opts ...RequestOption) (Result, error) {
	return s.Request(ctx, criteria, opts...)
}

// Query retrieves all the messages with specific message hashes
func (s *WakuStore) QueryByHash(ctx context.Context, messageHashes []wpb.MessageHash, opts ...RequestOption) (Result, error) {
	return s.Request(ctx, MessageHashCriteria{messageHashes}, opts...)
}

// Exists is an utility function to determine if a message exists. For checking the presence of more than one message, use QueryByHash
// and pass the option WithReturnValues(false). You will have to iterate the results and check whether the full list of messages contains
// the list of messages to verify
func (s *WakuStore) Exists(ctx context.Context, messageHash wpb.MessageHash, opts ...RequestOption) (bool, error) {
	opts = append(opts, IncludeData(false))
	result, err := s.Request(ctx, MessageHashCriteria{MessageHashes: []wpb.MessageHash{messageHash}}, opts...)
	if err != nil {
		return false, err
	}

	return len(result.Messages()) != 0, nil
}

func (s *WakuStore) next(ctx context.Context, r Result, opts ...RequestOption) (*resultImpl, error) {
	if r.IsComplete() {
		return &resultImpl{
			store:         s,
			messages:      nil,
			cursor:        nil,
			storeRequest:  r.Query(),
			storeResponse: r.Response(),
			peerID:        r.PeerID(),
		}, nil
	}

	params := new(Parameters)
	params.selectedPeer = r.PeerID()
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	storeRequest := proto.Clone(r.Query()).(*pb.StoreQueryRequest)
	storeRequest.RequestId = hex.EncodeToString(protocol.GenerateRequestID())
	storeRequest.PaginationCursor = r.Cursor()

	response, err := s.queryFrom(ctx, storeRequest, params)
	if err != nil {
		return nil, err
	}

	result := &resultImpl{
		store:         s,
		messages:      response.Messages,
		storeRequest:  storeRequest,
		storeResponse: response,
		peerID:        r.PeerID(),
		cursor:        response.PaginationCursor,
	}

	return result, nil

}

func (s *WakuStore) queryFrom(ctx context.Context, storeRequest *pb.StoreQueryRequest, params *Parameters) (*pb.StoreQueryResponse, error) {
	logger := s.log.With(logging.HostID("peer", params.selectedPeer), zap.String("requestId", storeRequest.RequestId))

	logger.Debug("sending store request")

	if !params.skipRatelimit {
		rateLimiter, ok := s.rateLimiters[params.selectedPeer]
		if !ok {
			rateLimiter = rate.NewLimiter(s.defaultRatelimit, 1)
			s.rateLimiters[params.selectedPeer] = rateLimiter
		}
		err := rateLimiter.Wait(ctx)
		if err != nil {
			return nil, err
		}
	}

	stream, err := s.h.NewStream(ctx, params.selectedPeer, StoreQueryID_v300)
	if err != nil {
		if s.pm != nil {
			s.pm.HandleDialError(err, params.selectedPeer)
		}
		return nil, err
	}

	writer := pbio.NewDelimitedWriter(stream)
	reader := pbio.NewDelimitedReader(stream, math.MaxInt32)

	err = writer.WriteMsg(storeRequest)
	if err != nil {
		logger.Error("writing request", zap.Error(err))
		if err := stream.Reset(); err != nil {
			s.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	storeResponse := &pb.StoreQueryResponse{RequestId: storeRequest.RequestId}
	err = reader.ReadMsg(storeResponse)
	if err != nil {
		logger.Error("reading response", zap.Error(err))
		if err := stream.Reset(); err != nil {
			s.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	stream.Close()

	if err := storeResponse.Validate(storeRequest.RequestId); err != nil {
		return nil, err
	}

	if storeResponse.GetStatusCode() != ok {
		err := NewStoreError(int(storeResponse.GetStatusCode()), storeResponse.GetStatusDesc())
		return nil, err
	}
	return storeResponse, nil
}
