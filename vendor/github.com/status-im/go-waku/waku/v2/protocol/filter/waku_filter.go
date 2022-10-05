package filter

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"github.com/status-im/go-waku/logging"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
)

type (
	Filter struct {
		filterID       string
		PeerID         peer.ID
		Topic          string
		ContentFilters []string
		Chan           chan *protocol.Envelope
	}

	ContentFilter struct {
		Topic         string
		ContentTopics []string
	}

	FilterSubscription struct {
		RequestID string
		Peer      peer.ID
	}

	WakuFilter struct {
		ctx        context.Context
		h          host.Host
		isFullNode bool
		MsgC       chan *protocol.Envelope
		wg         *sync.WaitGroup
		log        *zap.Logger

		filters     *FilterMap
		subscribers *Subscribers
	}
)

// FilterID_v20beta1 is the current Waku Filter protocol identifier
const FilterID_v20beta1 = libp2pProtocol.ID("/vac/waku/filter/2.0.0-beta1")

// NewWakuRelay returns a new instance of Waku Filter struct setup according to the chosen parameter and options
func NewWakuFilter(ctx context.Context, host host.Host, isFullNode bool, log *zap.Logger, opts ...Option) (*WakuFilter, error) {
	wf := new(WakuFilter)
	wf.log = log.Named("filter").With(zap.Bool("fullNode", isFullNode))

	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyType, "filter"))
	if err != nil {
		wf.log.Error("creating tag map", zap.Error(err))
		return nil, errors.New("could not start waku filter")
	}

	params := new(FilterParameters)
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	wf.ctx = ctx
	wf.wg = &sync.WaitGroup{}
	wf.MsgC = make(chan *protocol.Envelope, 1024)
	wf.h = host
	wf.isFullNode = isFullNode
	wf.filters = NewFilterMap()
	wf.subscribers = NewSubscribers(params.timeout)

	wf.h.SetStreamHandlerMatch(FilterID_v20beta1, protocol.PrefixTextMatch(string(FilterID_v20beta1)), wf.onRequest)

	wf.wg.Add(1)
	go wf.filterListener()

	wf.log.Info("filter protocol started")
	return wf, nil
}

func (wf *WakuFilter) onRequest(s network.Stream) {
	defer s.Close()
	logger := wf.log.With(logging.HostID("peer", s.Conn().RemotePeer()))

	filterRPCRequest := &pb.FilterRPC{}

	reader := protoio.NewDelimitedReader(s, math.MaxInt32)

	err := reader.ReadMsg(filterRPCRequest)
	if err != nil {
		logger.Error("reading request", zap.Error(err))
		return
	}

	logger.Info("received request")

	if filterRPCRequest.Push != nil && len(filterRPCRequest.Push.Messages) > 0 {
		// We're on a light node.
		// This is a message push coming from a full node.
		for _, message := range filterRPCRequest.Push.Messages {
			wf.filters.Notify(message, filterRPCRequest.RequestId) // Trigger filter handlers on a light node
		}

		logger.Info("received a message push", zap.Int("messages", len(filterRPCRequest.Push.Messages)))
		stats.Record(wf.ctx, metrics.Messages.M(int64(len(filterRPCRequest.Push.Messages))))
	} else if filterRPCRequest.Request != nil && wf.isFullNode {
		// We're on a full node.
		// This is a filter request coming from a light node.
		if filterRPCRequest.Request.Subscribe {
			subscriber := Subscriber{peer: s.Conn().RemotePeer(), requestId: filterRPCRequest.RequestId, filter: *filterRPCRequest.Request}
			len := wf.subscribers.Append(subscriber)

			logger.Info("adding subscriber")
			stats.Record(wf.ctx, metrics.FilterSubscriptions.M(int64(len)))
		} else {
			peerId := s.Conn().RemotePeer()
			wf.subscribers.RemoveContentFilters(peerId, filterRPCRequest.RequestId, filterRPCRequest.Request.ContentFilters)

			logger.Info("removing subscriber")
			stats.Record(wf.ctx, metrics.FilterSubscriptions.M(int64(wf.subscribers.Length())))
		}
	} else {
		logger.Error("can't serve request")
		return
	}
}

func (wf *WakuFilter) pushMessage(subscriber Subscriber, msg *pb.WakuMessage) error {
	pushRPC := &pb.FilterRPC{RequestId: subscriber.requestId, Push: &pb.MessagePush{Messages: []*pb.WakuMessage{msg}}}
	logger := wf.log.With(logging.HostID("peer", subscriber.peer))

	// We connect first so dns4 addresses are resolved (NewStream does not do it)
	err := wf.h.Connect(wf.ctx, wf.h.Peerstore().PeerInfo(subscriber.peer))
	if err != nil {
		wf.subscribers.FlagAsFailure(subscriber.peer)
		logger.Error("connecting to peer", zap.Error(err))
		return err
	}

	conn, err := wf.h.NewStream(wf.ctx, subscriber.peer, FilterID_v20beta1)
	if err != nil {
		wf.subscribers.FlagAsFailure(subscriber.peer)

		logger.Error("opening peer stream", zap.Error(err))
		//waku_filter_errors.inc(labelValues = [dialFailure])
		return err
	}

	defer conn.Close()
	writer := protoio.NewDelimitedWriter(conn)
	err = writer.WriteMsg(pushRPC)
	if err != nil {
		logger.Error("pushing messages to peer", zap.Error(err))
		wf.subscribers.FlagAsFailure(subscriber.peer)
		return nil
	}

	wf.subscribers.FlagAsSuccess(subscriber.peer)
	return nil
}

func (wf *WakuFilter) filterListener() {
	defer wf.wg.Done()

	// This function is invoked for each message received
	// on the full node in context of Waku2-Filter
	handle := func(envelope *protocol.Envelope) error { // async
		msg := envelope.Message()
		pubsubTopic := envelope.PubsubTopic()
		logger := wf.log.With(zap.Stringer("message", msg))
		g := new(errgroup.Group)
		// Each subscriber is a light node that earlier on invoked
		// a FilterRequest on this node
		for subscriber := range wf.subscribers.Items(&(msg.ContentTopic)) {
			logger := logger.With(logging.HostID("subscriber", subscriber.peer))
			subscriber := subscriber // https://golang.org/doc/faq#closures_and_goroutines
			if subscriber.filter.Topic != "" && subscriber.filter.Topic != pubsubTopic {
				logger.Info("pubsub topic mismatch",
					zap.String("subscriberTopic", subscriber.filter.Topic),
					zap.String("messageTopic", pubsubTopic))
				continue
			}

			// Do a message push to light node
			logger.Info("pushing message to light node", zap.String("contentTopic", msg.ContentTopic))
			g.Go(func() (err error) {
				err = wf.pushMessage(subscriber, msg)
				if err != nil {
					logger.Error("pushing message", zap.Error(err))
				}
				return err
			})
		}

		return g.Wait()
	}

	for m := range wf.MsgC {
		if err := handle(m); err != nil {
			wf.log.Error("handling message", zap.Error(err))
		}
	}
}

// Having a FilterRequest struct,
// select a peer with filter support, dial it,
// and submit FilterRequest wrapped in FilterRPC
func (wf *WakuFilter) requestSubscription(ctx context.Context, filter ContentFilter, opts ...FilterSubscribeOption) (subscription *FilterSubscription, err error) {
	params := new(FilterSubscribeParameters)
	params.log = wf.log
	params.host = wf.h

	optList := DefaultSubscribtionOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	if params.selectedPeer == "" {
		return nil, ErrNoPeersAvailable
	}

	var contentFilters []*pb.FilterRequest_ContentFilter
	for _, ct := range filter.ContentTopics {
		contentFilters = append(contentFilters, &pb.FilterRequest_ContentFilter{ContentTopic: ct})
	}

	// We connect first so dns4 addresses are resolved (NewStream does not do it)
	err = wf.h.Connect(ctx, wf.h.Peerstore().PeerInfo(params.selectedPeer))
	if err != nil {
		return
	}

	request := pb.FilterRequest{
		Subscribe:      true,
		Topic:          filter.Topic,
		ContentFilters: contentFilters,
	}

	var conn network.Stream
	conn, err = wf.h.NewStream(ctx, params.selectedPeer, FilterID_v20beta1)
	if err != nil {
		return
	}

	defer conn.Close()

	// This is the only successful path to subscription
	requestID := hex.EncodeToString(protocol.GenerateRequestId())

	writer := protoio.NewDelimitedWriter(conn)
	filterRPC := &pb.FilterRPC{RequestId: requestID, Request: &request}
	wf.log.Info("sending filterRPC", zap.Stringer("rpc", filterRPC))
	err = writer.WriteMsg(filterRPC)
	if err != nil {
		wf.log.Error("sending filterRPC", zap.Error(err))
		return
	}

	subscription = new(FilterSubscription)
	subscription.Peer = params.selectedPeer
	subscription.RequestID = requestID

	return
}

// Unsubscribe is used to stop receiving messages from a peer that match a content filter
func (wf *WakuFilter) Unsubscribe(ctx context.Context, contentFilter ContentFilter, peer peer.ID) error {
	// We connect first so dns4 addresses are resolved (NewStream does not do it)
	err := wf.h.Connect(ctx, wf.h.Peerstore().PeerInfo(peer))
	if err != nil {
		return err
	}

	conn, err := wf.h.NewStream(ctx, peer, FilterID_v20beta1)
	if err != nil {
		return err
	}

	defer conn.Close()

	// This is the only successful path to subscription
	id := protocol.GenerateRequestId()

	var contentFilters []*pb.FilterRequest_ContentFilter
	for _, ct := range contentFilter.ContentTopics {
		contentFilters = append(contentFilters, &pb.FilterRequest_ContentFilter{ContentTopic: ct})
	}

	request := pb.FilterRequest{
		Subscribe:      false,
		Topic:          contentFilter.Topic,
		ContentFilters: contentFilters,
	}

	writer := protoio.NewDelimitedWriter(conn)
	filterRPC := &pb.FilterRPC{RequestId: hex.EncodeToString(id), Request: &request}
	err = writer.WriteMsg(filterRPC)
	if err != nil {
		return err
	}

	return nil
}

// Stop unmounts the filter protocol
func (wf *WakuFilter) Stop() {
	close(wf.MsgC)

	wf.h.RemoveStreamHandler(FilterID_v20beta1)
	wf.filters.RemoveAll()
	wf.wg.Wait()
}

// Subscribe setups a subscription to receive messages that match a specific content filter
func (wf *WakuFilter) Subscribe(ctx context.Context, f ContentFilter, opts ...FilterSubscribeOption) (filterID string, theFilter Filter, err error) {
	// TODO: check if there's an existing pubsub topic that uses the same peer. If so, reuse filter, and return same channel and filterID

	// Registers for messages that match a specific filter. Triggers the handler whenever a message is received.
	// ContentFilterChan takes MessagePush structs
	remoteSubs, err := wf.requestSubscription(ctx, f, opts...)
	if err != nil || remoteSubs.RequestID == "" {
		// Failed to subscribe
		wf.log.Error("requesting subscription", zap.Error(err))
		return
	}

	// Register handler for filter, whether remote subscription succeeded or not

	filterID = remoteSubs.RequestID
	theFilter = Filter{
		filterID:       filterID,
		PeerID:         remoteSubs.Peer,
		Topic:          f.Topic,
		ContentFilters: f.ContentTopics,
		Chan:           make(chan *protocol.Envelope, 1024), // To avoid blocking
	}

	wf.filters.Set(filterID, theFilter)

	return
}

// UnsubscribeFilterByID removes a subscription to a filter node completely
// using using a filter. It also closes the filter channel
func (wf *WakuFilter) UnsubscribeByFilter(ctx context.Context, filter Filter) error {
	err := wf.UnsubscribeFilterByID(ctx, filter.filterID)
	if err != nil {
		close(filter.Chan)
	}
	return err
}

// UnsubscribeFilterByID removes a subscription to a filter node completely
// using the filterID returned when the subscription was created
func (wf *WakuFilter) UnsubscribeFilterByID(ctx context.Context, filterID string) error {

	var f Filter
	var ok bool

	if f, ok = wf.filters.Get(filterID); !ok {
		return errors.New("filter not found")
	}

	cf := ContentFilter{
		Topic:         f.Topic,
		ContentTopics: f.ContentFilters,
	}

	err := wf.Unsubscribe(ctx, cf, f.PeerID)
	if err != nil {
		return err
	}

	wf.filters.Delete(filterID)

	return nil
}

// Unsubscribe filter removes content topics from a filter subscription. If all
// the contentTopics are removed the subscription is dropped completely
func (wf *WakuFilter) UnsubscribeFilter(ctx context.Context, cf ContentFilter) error {
	// Remove local filter
	var idsToRemove []string
	for filterMapItem := range wf.filters.Items() {
		f := filterMapItem.Value
		id := filterMapItem.Key

		if f.Topic != cf.Topic {
			continue
		}

		// Send message to full node in order to unsubscribe
		err := wf.Unsubscribe(ctx, cf, f.PeerID)
		if err != nil {
			return err
		}

		// Iterate filter entries to remove matching content topics
		// make sure we delete the content filter
		// if no more topics are left
		for _, cfToDelete := range cf.ContentTopics {
			for i, cf := range f.ContentFilters {
				if cf == cfToDelete {
					l := len(f.ContentFilters) - 1
					f.ContentFilters[l], f.ContentFilters[i] = f.ContentFilters[i], f.ContentFilters[l]
					f.ContentFilters = f.ContentFilters[:l]
					break
				}

			}
			if len(f.ContentFilters) == 0 {
				idsToRemove = append(idsToRemove, id)
			}
		}
	}

	for _, rId := range idsToRemove {
		wf.filters.Delete(rId)
	}

	return nil
}
