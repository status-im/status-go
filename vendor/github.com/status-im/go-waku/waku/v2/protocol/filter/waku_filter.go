package filter

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("wakufilter")

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
)

type (
	Filter struct {
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

		filters     *FilterMap
		subscribers *Subscribers
	}
)

// NOTE This is just a start, the design of this protocol isn't done yet. It
// should be direct payload exchange (a la req-resp), not be coupled with the
// relay protocol.
const FilterID_v20beta1 = libp2pProtocol.ID("/vac/waku/filter/2.0.0-beta1")

func NewWakuFilter(ctx context.Context, host host.Host, isFullNode bool) *WakuFilter {
	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyType, "filter"))
	if err != nil {
		log.Error(err)
	}

	wf := new(WakuFilter)
	wf.ctx = ctx
	wf.MsgC = make(chan *protocol.Envelope)
	wf.h = host
	wf.isFullNode = isFullNode
	wf.filters = NewFilterMap()
	wf.subscribers = NewSubscribers()

	wf.h.SetStreamHandlerMatch(FilterID_v20beta1, protocol.PrefixTextMatch(string(FilterID_v20beta1)), wf.onRequest)
	go wf.FilterListener()

	if wf.isFullNode {
		log.Info("Filter protocol started")
	} else {
		log.Info("Filter protocol started (only client mode)")
	}

	return wf
}

func (wf *WakuFilter) onRequest(s network.Stream) {
	defer s.Close()

	filterRPCRequest := &pb.FilterRPC{}

	reader := protoio.NewDelimitedReader(s, 64*1024)

	err := reader.ReadMsg(filterRPCRequest)
	if err != nil {
		log.Error("error reading request", err)
		return
	}

	log.Info(fmt.Sprintf("%s: received request from %s", s.Conn().LocalPeer(), s.Conn().RemotePeer()))

	if filterRPCRequest.Push != nil && len(filterRPCRequest.Push.Messages) > 0 {
		// We're on a light node.
		// This is a message push coming from a full node.
		for _, message := range filterRPCRequest.Push.Messages {
			wf.filters.Notify(message, filterRPCRequest.RequestId) // Trigger filter handlers on a light node
		}

		log.Info("filter light node, received a message push. ", len(filterRPCRequest.Push.Messages), " messages")
		stats.Record(wf.ctx, metrics.Messages.M(int64(len(filterRPCRequest.Push.Messages))))
	} else if filterRPCRequest.Request != nil && wf.isFullNode {
		// We're on a full node.
		// This is a filter request coming from a light node.
		if filterRPCRequest.Request.Subscribe {
			subscriber := Subscriber{peer: s.Conn().RemotePeer(), requestId: filterRPCRequest.RequestId, filter: *filterRPCRequest.Request}
			len := wf.subscribers.Append(subscriber)

			log.Info("filter full node, add a filter subscriber: ", subscriber.peer)
			stats.Record(wf.ctx, metrics.FilterSubscriptions.M(int64(len)))
		} else {
			peerId := s.Conn().RemotePeer()
			wf.subscribers.RemoveContentFilters(peerId, filterRPCRequest.Request.ContentFilters)

			log.Info("filter full node, remove a filter subscriber: ", peerId.Pretty())
			stats.Record(wf.ctx, metrics.FilterSubscriptions.M(int64(wf.subscribers.Length())))
		}
	} else {
		log.Error("can't serve request")
		return
	}
}

func (wf *WakuFilter) pushMessage(subscriber Subscriber, msg *pb.WakuMessage) error {
	pushRPC := &pb.FilterRPC{RequestId: subscriber.requestId, Push: &pb.MessagePush{Messages: []*pb.WakuMessage{msg}}}

	conn, err := wf.h.NewStream(wf.ctx, peer.ID(subscriber.peer), FilterID_v20beta1)
	// TODO: keep track of errors to automatically unsubscribe a peer?
	if err != nil {
		// @TODO more sophisticated error handling here
		log.Error("failed to open peer stream")
		//waku_filter_errors.inc(labelValues = [dialFailure])
		return err
	}

	defer conn.Close()
	writer := protoio.NewDelimitedWriter(conn)
	err = writer.WriteMsg(pushRPC)
	if err != nil {
		log.Error("failed to push messages to remote peer")
		return nil
	}

	return nil
}

func (wf *WakuFilter) FilterListener() {
	// This function is invoked for each message received
	// on the full node in context of Waku2-Filter
	handle := func(envelope *protocol.Envelope) error { // async
		msg := envelope.Message()
		topic := envelope.PubsubTopic()
		// Each subscriber is a light node that earlier on invoked
		// a FilterRequest on this node
		for subscriber := range wf.subscribers.Items() {
			if subscriber.filter.Topic != "" && subscriber.filter.Topic != topic {
				log.Info("Subscriber's filter pubsubTopic does not match message topic", subscriber.filter.Topic, topic)
				continue
			}

			for _, filter := range subscriber.filter.ContentFilters {
				if msg.ContentTopic == filter.ContentTopic {
					log.Info("found matching contentTopic ", filter, msg)
					// Do a message push to light node
					log.Info("pushing messages to light node: ", subscriber.peer)
					if err := wf.pushMessage(subscriber, msg); err != nil {
						return err
					}

				}
			}
		}

		return nil
	}

	for m := range wf.MsgC {
		if err := handle(m); err != nil {
			log.Error("failed to handle message", err)
		}
	}

}

// Having a FilterRequest struct,
// select a peer with filter support, dial it,
// and submit FilterRequest wrapped in FilterRPC
func (wf *WakuFilter) requestSubscription(ctx context.Context, filter ContentFilter, opts ...FilterSubscribeOption) (subscription *FilterSubscription, err error) {
	params := new(FilterSubscribeParameters)
	params.host = wf.h

	optList := DefaultOptions()
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
	log.Info("sending filterRPC: ", filterRPC)
	err = writer.WriteMsg(filterRPC)
	if err != nil {
		log.Error("failed to write message", err)
		return
	}

	subscription = new(FilterSubscription)
	subscription.Peer = params.selectedPeer
	subscription.RequestID = requestID

	return
}

func (wf *WakuFilter) Unsubscribe(ctx context.Context, contentFilter ContentFilter, peer peer.ID) error {
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

func (wf *WakuFilter) Stop() {
	wf.h.RemoveStreamHandler(FilterID_v20beta1)
	wf.filters.RemoveAll()
}

func (wf *WakuFilter) Subscribe(ctx context.Context, f ContentFilter, opts ...FilterSubscribeOption) (filterID string, theFilter Filter, err error) {
	// TODO: check if there's an existing pubsub topic that uses the same peer. If so, reuse filter, and return same channel and filterID

	// Registers for messages that match a specific filter. Triggers the handler whenever a message is received.
	// ContentFilterChan takes MessagePush structs
	remoteSubs, err := wf.requestSubscription(ctx, f, opts...)
	if err != nil || remoteSubs.RequestID == "" {
		// Failed to subscribe
		log.Error("remote subscription to filter failed", err)
		return
	}

	// Register handler for filter, whether remote subscription succeeded or not

	filterID = remoteSubs.RequestID
	theFilter = Filter{
		PeerID:         remoteSubs.Peer,
		Topic:          f.Topic,
		ContentFilters: f.ContentTopics,
		Chan:           make(chan *protocol.Envelope, 1024), // To avoid blocking
	}

	wf.filters.Set(filterID, theFilter)

	return
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
