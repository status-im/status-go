package filter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.uber.org/zap"
)

// FilterSubscribeID_v20beta1 is the current Waku Filter protocol identifier for servers to
// allow filter clients to subscribe, modify, refresh and unsubscribe a desired set of filter criteria
const FilterSubscribeID_v20beta1 = libp2pProtocol.ID("/vac/waku/filter-subscribe/2.0.0-beta1")

const peerHasNoSubscription = "peer has no subscriptions"

type (
	WakuFilterFullNode struct {
		cancel  context.CancelFunc
		h       host.Host
		msgSub  relay.Subscription
		metrics Metrics
		wg      *sync.WaitGroup
		log     *zap.Logger

		subscriptions *SubscribersMap

		maxSubscriptions int
	}
)

// NewWakuFilterFullNode returns a new instance of Waku Filter struct setup according to the chosen parameter and options
func NewWakuFilterFullNode(timesource timesource.Timesource, reg prometheus.Registerer, log *zap.Logger, opts ...Option) *WakuFilterFullNode {
	wf := new(WakuFilterFullNode)
	wf.log = log.Named("filterv2-fullnode")

	params := new(FilterParameters)
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	wf.wg = &sync.WaitGroup{}
	wf.metrics = newMetrics(reg)
	wf.subscriptions = NewSubscribersMap(params.Timeout)
	wf.maxSubscriptions = params.MaxSubscribers

	return wf
}

// Sets the host to be able to mount or consume a protocol
func (wf *WakuFilterFullNode) SetHost(h host.Host) {
	wf.h = h
}

func (wf *WakuFilterFullNode) Start(ctx context.Context, sub relay.Subscription) error {
	wf.wg.Wait() // Wait for any goroutines to stop

	ctx, cancel := context.WithCancel(ctx)

	wf.h.SetStreamHandlerMatch(FilterSubscribeID_v20beta1, protocol.PrefixTextMatch(string(FilterSubscribeID_v20beta1)), wf.onRequest(ctx))

	wf.cancel = cancel
	wf.msgSub = sub
	wf.wg.Add(1)
	go wf.filterListener(ctx)

	wf.log.Info("filter-subscriber protocol started")

	return nil
}

func (wf *WakuFilterFullNode) onRequest(ctx context.Context) func(s network.Stream) {
	return func(s network.Stream) {
		defer s.Close()
		logger := wf.log.With(logging.HostID("peer", s.Conn().RemotePeer()))

		reader := pbio.NewDelimitedReader(s, math.MaxInt32)

		subscribeRequest := &pb.FilterSubscribeRequest{}
		err := reader.ReadMsg(subscribeRequest)
		if err != nil {
			wf.metrics.RecordError(decodeRPCFailure)
			logger.Error("reading request", zap.Error(err))
			return
		}

		logger = logger.With(zap.String("requestID", subscribeRequest.RequestId))

		start := time.Now()

		switch subscribeRequest.FilterSubscribeType {
		case pb.FilterSubscribeRequest_SUBSCRIBE:
			wf.subscribe(ctx, s, logger, subscribeRequest)
		case pb.FilterSubscribeRequest_SUBSCRIBER_PING:
			wf.ping(ctx, s, logger, subscribeRequest)
		case pb.FilterSubscribeRequest_UNSUBSCRIBE:
			wf.unsubscribe(ctx, s, logger, subscribeRequest)
		case pb.FilterSubscribeRequest_UNSUBSCRIBE_ALL:
			wf.unsubscribeAll(ctx, s, logger, subscribeRequest)
		}

		wf.metrics.RecordRequest(subscribeRequest.FilterSubscribeType.String(), time.Since(start))

		logger.Info("received request", zap.String("requestType", subscribeRequest.FilterSubscribeType.String()))
	}
}

func (wf *WakuFilterFullNode) reply(ctx context.Context, s network.Stream, request *pb.FilterSubscribeRequest, statusCode int, description ...string) {
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
		wf.metrics.RecordError(writeResponseFailure)
		wf.log.Error("sending response", zap.Error(err))
	}
}

func (wf *WakuFilterFullNode) ping(ctx context.Context, s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest) {
	exists := wf.subscriptions.Has(s.Conn().RemotePeer())

	if exists {
		wf.reply(ctx, s, request, http.StatusOK)
	} else {
		wf.reply(ctx, s, request, http.StatusNotFound, peerHasNoSubscription)
	}
}

func (wf *WakuFilterFullNode) subscribe(ctx context.Context, s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest) {
	if request.PubsubTopic == "" {
		wf.reply(ctx, s, request, http.StatusBadRequest, "pubsubtopic can't be empty")
		return
	}

	if len(request.ContentTopics) == 0 {
		wf.reply(ctx, s, request, http.StatusBadRequest, "at least one contenttopic should be specified")
		return
	}

	if len(request.ContentTopics) > MaxContentTopicsPerRequest {
		wf.reply(ctx, s, request, http.StatusBadRequest, fmt.Sprintf("exceeds maximum content topics: %d", MaxContentTopicsPerRequest))
	}

	if wf.subscriptions.Count() >= wf.maxSubscriptions {
		wf.reply(ctx, s, request, http.StatusServiceUnavailable, "node has reached maximum number of subscriptions")
		return
	}

	peerID := s.Conn().RemotePeer()

	if totalSubs, exists := wf.subscriptions.Get(peerID); exists {
		ctTotal := 0
		for _, contentTopicSet := range totalSubs {
			ctTotal += len(contentTopicSet)
		}

		if ctTotal+len(request.ContentTopics) > MaxCriteriaPerSubscription {
			wf.reply(ctx, s, request, http.StatusServiceUnavailable, "peer has reached maximum number of filter criteria")
			return
		}
	}

	wf.subscriptions.Set(peerID, request.PubsubTopic, request.ContentTopics)

	wf.metrics.RecordSubscriptions(wf.subscriptions.Count())
	wf.reply(ctx, s, request, http.StatusOK)
}

func (wf *WakuFilterFullNode) unsubscribe(ctx context.Context, s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest) {
	if request.PubsubTopic == "" {
		wf.reply(ctx, s, request, http.StatusBadRequest, "pubsubtopic can't be empty")
		return
	}

	if len(request.ContentTopics) == 0 {
		wf.reply(ctx, s, request, http.StatusBadRequest, "at least one contenttopic should be specified")
		return
	}

	if len(request.ContentTopics) > MaxContentTopicsPerRequest {
		wf.reply(ctx, s, request, http.StatusBadRequest, fmt.Sprintf("exceeds maximum content topics: %d", MaxContentTopicsPerRequest))
	}

	err := wf.subscriptions.Delete(s.Conn().RemotePeer(), request.PubsubTopic, request.ContentTopics)
	if err != nil {
		wf.reply(ctx, s, request, http.StatusNotFound, peerHasNoSubscription)
	} else {
		wf.metrics.RecordSubscriptions(wf.subscriptions.Count())
		wf.reply(ctx, s, request, http.StatusOK)
	}
}

func (wf *WakuFilterFullNode) unsubscribeAll(ctx context.Context, s network.Stream, logger *zap.Logger, request *pb.FilterSubscribeRequest) {
	err := wf.subscriptions.DeleteAll(s.Conn().RemotePeer())
	if err != nil {
		wf.reply(ctx, s, request, http.StatusNotFound, peerHasNoSubscription)
	} else {
		wf.metrics.RecordSubscriptions(wf.subscriptions.Count())
		wf.reply(ctx, s, request, http.StatusOK)
	}
}

func (wf *WakuFilterFullNode) filterListener(ctx context.Context) {
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
				start := time.Now()
				err := wf.pushMessage(ctx, subscriber, envelope)
				if err != nil {
					logger.Error("pushing message", zap.Error(err))
					return
				}
				wf.metrics.RecordPushDuration(time.Since(start))
			}(subscriber)
		}

		return nil
	}

	for m := range wf.msgSub.Ch {
		if err := handle(m); err != nil {
			wf.log.Error("handling message", zap.Error(err))
		}
	}
}

func (wf *WakuFilterFullNode) pushMessage(ctx context.Context, peerID peer.ID, env *protocol.Envelope) error {
	logger := wf.log.With(logging.HostID("peer", peerID))

	messagePush := &pb.MessagePushV2{
		PubsubTopic: env.PubsubTopic(),
		WakuMessage: env.Message(),
	}

	ctx, cancel := context.WithTimeout(ctx, MessagePushTimeout)
	defer cancel()

	conn, err := wf.h.NewStream(ctx, peerID, FilterPushID_v20beta1)
	if err != nil {
		wf.subscriptions.FlagAsFailure(peerID)
		if errors.Is(context.DeadlineExceeded, err) {
			wf.metrics.RecordError(pushTimeoutFailure)
		} else {
			wf.metrics.RecordError(dialFailure)
		}
		logger.Error("opening peer stream", zap.Error(err))
		return err
	}

	defer conn.Close()
	writer := pbio.NewDelimitedWriter(conn)
	err = writer.WriteMsg(messagePush)
	if err != nil {
		if errors.Is(context.DeadlineExceeded, err) {
			wf.metrics.RecordError(pushTimeoutFailure)
		} else {
			wf.metrics.RecordError(writeResponseFailure)
		}
		logger.Error("pushing messages to peer", logging.HexBytes("envelopeHash", env.Hash()), zap.String("pubsubTopic", env.PubsubTopic()), zap.String("contentTopic", env.Message().ContentTopic), zap.Error(err))
		wf.subscriptions.FlagAsFailure(peerID)
		return nil
	}

	wf.subscriptions.FlagAsSuccess(peerID)
	return nil
}

// Stop unmounts the filter protocol
func (wf *WakuFilterFullNode) Stop() {
	if wf.cancel == nil {
		return
	}

	wf.h.RemoveStreamHandler(FilterSubscribeID_v20beta1)

	wf.cancel()

	wf.msgSub.Unsubscribe()

	wf.wg.Wait()
}
