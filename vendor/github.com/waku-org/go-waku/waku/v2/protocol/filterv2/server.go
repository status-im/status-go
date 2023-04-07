package filterv2

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/waku-org/go-waku/logging"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filterv2/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
)

// FilterSubscribeID_v20beta1 is the current Waku Filter protocol identifier for servers to
// allow filter clients to subscribe, modify, refresh and unsubscribe a desired set of filter criteria
const FilterSubscribeID_v20beta1 = libp2pProtocol.ID("/vac/waku/filter-subscribe/2.0.0-beta1")

const peerHasNoSubscription = "peer has no subscriptions"

type (
	WakuFilterFull struct {
		cancel context.CancelFunc
		h      host.Host
		msgC   chan *protocol.Envelope
		wg     *sync.WaitGroup
		log    *zap.Logger

		subscriptions *SubscribersMap

		maxSubscriptions int
	}
)

// NewWakuFilterFullnode returns a new instance of Waku Filter struct setup according to the chosen parameter and options
func NewWakuFilterFullnode(host host.Host, broadcaster v2.Broadcaster, timesource timesource.Timesource, log *zap.Logger, opts ...Option) *WakuFilterFull {
	wf := new(WakuFilterFull)
	wf.log = log.Named("filterv2-fullnode")

	params := new(FilterParameters)
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	wf.wg = &sync.WaitGroup{}
	wf.h = host
	wf.subscriptions = NewSubscribersMap(params.Timeout)
	wf.maxSubscriptions = params.MaxSubscribers

	return wf
}

func (wf *WakuFilterFull) Start(ctx context.Context) error {
	wf.wg.Wait() // Wait for any goroutines to stop

	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyType, "filter"))
	if err != nil {
		wf.log.Error("creating tag map", zap.Error(err))
		return errors.New("could not start waku filter")
	}

	ctx, cancel := context.WithCancel(ctx)

	wf.h.SetStreamHandlerMatch(FilterSubscribeID_v20beta1, protocol.PrefixTextMatch(string(FilterSubscribeID_v20beta1)), wf.onRequest(ctx))

	wf.cancel = cancel
	wf.msgC = make(chan *protocol.Envelope, 1024)

	wf.wg.Add(1)
	go wf.filterListener(ctx)

	wf.log.Info("filter protocol (full) started")

	return nil
}

func (wf *WakuFilterFull) onRequest(ctx context.Context) func(s network.Stream) {
	return func(s network.Stream) {
		defer s.Close()
		logger := wf.log.With(logging.HostID("peer", s.Conn().RemotePeer()))

		reader := pbio.NewDelimitedReader(s, math.MaxInt32)

		subscribeRequest := &pb.FilterSubscribeRequest{}
		err := reader.ReadMsg(subscribeRequest)
		if err != nil {
			logger.Error("reading request", zap.Error(err))
			return
		}

		logger = logger.With(zap.String("requestID", subscribeRequest.RequestId))

		switch subscribeRequest.FilterSubscribeType {
		case pb.FilterSubscribeRequest_SUBSCRIBE:
			wf.subscribe(s, logger, subscribeRequest)
		case pb.FilterSubscribeRequest_SUBSCRIBER_PING:
			wf.ping(s, logger, subscribeRequest)
		case pb.FilterSubscribeRequest_UNSUBSCRIBE:
			wf.unsubscribe(s, logger, subscribeRequest)
		case pb.FilterSubscribeRequest_UNSUBSCRIBE_ALL:
			wf.unsubscribeAll(s, logger, subscribeRequest)
		}

		logger.Info("received request")
	}
}

func reply(s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest, statusCode int, description ...string) {
	response := &pb.FilterSubscribeResponse{
		RequestId:  request.RequestId,
		StatusCode: uint32(statusCode),
	}

	if len(description) != 0 {
		response.StatusDesc = description[0]
	} else {
		response.StatusDesc = http.StatusText(statusCode)
	}

	writer := pbio.NewDelimitedWriter(s)
	err := writer.WriteMsg(response)
	if err != nil {
		logger.Error("sending response", zap.Error(err))
	}
}

func (wf *WakuFilterFull) ping(s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest) {
	exists := wf.subscriptions.Has(s.Conn().RemotePeer())

	if exists {
		reply(s, logger, request, http.StatusOK)
	} else {
		reply(s, logger, request, http.StatusNotFound, peerHasNoSubscription)
	}
}

func (wf *WakuFilterFull) subscribe(s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest) {
	if request.PubsubTopic == "" {
		reply(s, logger, request, http.StatusBadRequest, "pubsubtopic can't be empty")
		return
	}

	if len(request.ContentTopics) == 0 {
		reply(s, logger, request, http.StatusBadRequest, "at least one contenttopic should be specified")
		return
	}

	if len(request.ContentTopics) > MaxContentTopicsPerRequest {
		reply(s, logger, request, http.StatusBadRequest, fmt.Sprintf("exceeds maximum content topics: %d", MaxContentTopicsPerRequest))
	}

	if wf.subscriptions.Count() >= wf.maxSubscriptions {
		reply(s, logger, request, http.StatusServiceUnavailable, "node has reached maximum number of subscriptions")
		return
	}

	peerID := s.Conn().RemotePeer()

	if totalSubs, exists := wf.subscriptions.Get(peerID); exists {
		ctTotal := 0
		for _, contentTopicSet := range totalSubs {
			ctTotal += len(contentTopicSet)
		}

		if ctTotal+len(request.ContentTopics) > MaxCriteriaPerSubscription {
			reply(s, logger, request, http.StatusServiceUnavailable, "peer has reached maximum number of filter criteria")
			return
		}
	}

	wf.subscriptions.Set(peerID, request.PubsubTopic, request.ContentTopics)

	reply(s, logger, request, http.StatusOK)
}

func (wf *WakuFilterFull) unsubscribe(s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest) {
	if request.PubsubTopic == "" {
		reply(s, logger, request, http.StatusBadRequest, "pubsubtopic can't be empty")
		return
	}

	if len(request.ContentTopics) == 0 {
		reply(s, logger, request, http.StatusBadRequest, "at least one contenttopic should be specified")
		return
	}

	if len(request.ContentTopics) > MaxContentTopicsPerRequest {
		reply(s, logger, request, http.StatusBadRequest, fmt.Sprintf("exceeds maximum content topics: %d", MaxContentTopicsPerRequest))
	}

	err := wf.subscriptions.Delete(s.Conn().RemotePeer(), request.PubsubTopic, request.ContentTopics)
	if err != nil {
		reply(s, logger, request, http.StatusNotFound, peerHasNoSubscription)
	} else {
		reply(s, logger, request, http.StatusOK)
	}
}

func (wf *WakuFilterFull) unsubscribeAll(s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest) {
	err := wf.subscriptions.DeleteAll(s.Conn().RemotePeer())
	if err != nil {
		reply(s, logger, request, http.StatusNotFound, peerHasNoSubscription)
	} else {
		reply(s, logger, request, http.StatusOK)
	}
}

func (wf *WakuFilterFull) filterListener(ctx context.Context) {
	defer wf.wg.Done()

	// This function is invoked for each message received
	// on the full node in context of Waku2-Filter
	handle := func(envelope *protocol.Envelope) error {
		msg := envelope.Message()
		pubsubTopic := envelope.PubsubTopic()
		logger := wf.log.With(logging.HexBytes("envelopeHash", envelope.Hash()))

		// Each subscriber is a light node that earlier on invoked
		// a FilterRequest on this node
		for subscriber := range wf.subscriptions.Items(pubsubTopic, msg.ContentTopic) {
			logger := logger.With(logging.HostID("subscriber", subscriber))
			subscriber := subscriber // https://golang.org/doc/faq#closures_and_goroutines
			// Do a message push to light node
			logger.Info("pushing message to light node")
			wf.wg.Add(1)
			go func(subscriber peer.ID) {
				defer wf.wg.Done()
				err := wf.pushMessage(ctx, subscriber, envelope)
				if err != nil {
					logger.Error("pushing message", zap.Error(err))
				}
			}(subscriber)
		}

		return nil
	}

	for m := range wf.msgC {
		if err := handle(m); err != nil {
			wf.log.Error("handling message", zap.Error(err))
		}
	}
}

func (wf *WakuFilterFull) pushMessage(ctx context.Context, peerID peer.ID, env *protocol.Envelope) error {
	logger := wf.log.With(logging.HostID("peer", peerID))

	messagePush := &pb.MessagePushV2{
		PubsubTopic: env.PubsubTopic(),
		WakuMessage: env.Message(),
	}

	ctx, cancel := context.WithTimeout(ctx, MessagePushTimeout)
	defer cancel()

	// We connect first so dns4 addresses are resolved (NewStream does not do it)
	err := wf.h.Connect(ctx, wf.h.Peerstore().PeerInfo(peerID))
	if err != nil {
		wf.subscriptions.FlagAsFailure(peerID)
		logger.Error("connecting to peer", zap.Error(err))
		return err
	}

	conn, err := wf.h.NewStream(ctx, peerID, FilterPushID_v20beta1)
	if err != nil {
		wf.subscriptions.FlagAsFailure(peerID)

		logger.Error("opening peer stream", zap.Error(err))
		//waku_filter_errors.inc(labelValues = [dialFailure])
		return err
	}

	defer conn.Close()
	writer := pbio.NewDelimitedWriter(conn)
	err = writer.WriteMsg(messagePush)
	if err != nil {
		logger.Error("pushing messages to peer", zap.Error(err))
		wf.subscriptions.FlagAsFailure(peerID)
		return nil
	}

	wf.subscriptions.FlagAsSuccess(peerID)
	return nil
}

// Stop unmounts the filter protocol
func (wf *WakuFilterFull) Stop() {
	if wf.cancel == nil {
		return
	}

	wf.h.RemoveStreamHandler(FilterSubscribeID_v20beta1)

	wf.cancel()

	close(wf.msgC)

	wf.wg.Wait()
}

func (wf *WakuFilterFull) MessageChannel() chan *protocol.Envelope {
	return wf.msgC
}
