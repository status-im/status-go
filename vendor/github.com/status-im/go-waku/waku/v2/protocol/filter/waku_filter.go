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
	"github.com/status-im/go-waku/waku/v2/utils"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("wakufilter")

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
)

type (
	FilterSubscribeParameters struct {
		host         host.Host
		selectedPeer peer.ID
	}

	FilterSubscribeOption func(*FilterSubscribeParameters)

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

	// @TODO MAYBE MORE INFO?
	Filters map[string]Filter

	Subscriber struct {
		peer      peer.ID
		requestId string
		filter    pb.FilterRequest // @TODO MAKE THIS A SEQUENCE AGAIN?
	}

	FilterSubscription struct {
		RequestID string
		Peer      peer.ID
	}

	MessagePushHandler func(requestId string, msg pb.MessagePush)

	WakuFilter struct {
		ctx         context.Context
		h           host.Host
		subscribers []Subscriber
		isFullNode  bool
		pushHandler MessagePushHandler
		MsgC        chan *protocol.Envelope
	}
)

// NOTE This is just a start, the design of this protocol isn't done yet. It
// should be direct payload exchange (a la req-resp), not be coupled with the
// relay protocol.

const FilterID_v20beta1 = libp2pProtocol.ID("/vac/waku/filter/2.0.0-beta1")

func WithPeer(p peer.ID) FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		params.selectedPeer = p
	}
}

func WithAutomaticPeerSelection() FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		p, err := utils.SelectPeer(params.host, string(FilterID_v20beta1))
		if err == nil {
			params.selectedPeer = *p
		} else {
			log.Info("Error selecting peer: ", err)
		}
	}
}

func DefaultOptions() []FilterSubscribeOption {
	return []FilterSubscribeOption{
		WithAutomaticPeerSelection(),
	}
}

func (filters *Filters) Notify(msg *pb.WakuMessage, requestId string) {
	for key, filter := range *filters {
		envelope := protocol.NewEnvelope(msg, filter.Topic)

		// We do this because the key for the filter is set to the requestId received from the filter protocol.
		// This means we do not need to check the content filter explicitly as all MessagePushs already contain
		// the requestId of the coresponding filter.
		if requestId != "" && requestId == key {
			filter.Chan <- envelope
			continue
		}

		// TODO: In case of no topics we should either trigger here for all messages,
		// or we should not allow such filter to exist in the first place.
		for _, contentTopic := range filter.ContentFilters {
			if msg.ContentTopic == contentTopic {
				filter.Chan <- envelope
				break
			}
		}
	}
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
		wf.pushHandler(filterRPCRequest.RequestId, *filterRPCRequest.Push)

		log.Info("filter light node, received a message push. ", len(filterRPCRequest.Push.Messages), " messages")
		stats.Record(wf.ctx, metrics.Messages.M(int64(len(filterRPCRequest.Push.Messages))))
	} else if filterRPCRequest.Request != nil && wf.isFullNode {
		// We're on a full node.
		// This is a filter request coming from a light node.
		if filterRPCRequest.Request.Subscribe {
			subscriber := Subscriber{peer: s.Conn().RemotePeer(), requestId: filterRPCRequest.RequestId, filter: *filterRPCRequest.Request}
			wf.subscribers = append(wf.subscribers, subscriber)
			log.Info("filter full node, add a filter subscriber: ", subscriber.peer)

			stats.Record(wf.ctx, metrics.FilterSubscriptions.M(int64(len(wf.subscribers))))
		} else {
			peerId := s.Conn().RemotePeer()
			log.Info("filter full node, remove a filter subscriber: ", peerId.Pretty())
			contentFilters := filterRPCRequest.Request.ContentFilters
			var peerIdsToRemove []peer.ID
			for _, subscriber := range wf.subscribers {
				if subscriber.peer != peerId {
					continue
				}

				// make sure we delete the content filter
				// if no more topics are left
				for i, contentFilter := range contentFilters {
					subCfs := subscriber.filter.ContentFilters
					for _, cf := range subCfs {
						if cf.ContentTopic == contentFilter.ContentTopic {
							l := len(subCfs) - 1
							subCfs[l], subCfs[i] = subCfs[i], subCfs[l]
							subscriber.filter.ContentFilters = subCfs[:l]
						}
					}
				}

				if len(subscriber.filter.ContentFilters) == 0 {
					peerIdsToRemove = append(peerIdsToRemove, subscriber.peer)
				}
			}

			// make sure we delete the subscriber
			// if no more content filters left
			for _, peerId := range peerIdsToRemove {
				for i, s := range wf.subscribers {
					if s.peer == peerId {
						l := len(wf.subscribers) - 1
						wf.subscribers[l], wf.subscribers[i] = wf.subscribers[i], wf.subscribers[l]
						wf.subscribers = wf.subscribers[:l]
						break
					}
				}
			}

			stats.Record(wf.ctx, metrics.FilterSubscriptions.M(int64(len(wf.subscribers))))
		}
	} else {
		log.Error("can't serve request")
		return
	}
}

func NewWakuFilter(ctx context.Context, host host.Host, isFullNode bool, handler MessagePushHandler) *WakuFilter {
	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyType, "filter"))
	if err != nil {
		log.Error(err)
	}

	wf := new(WakuFilter)
	wf.ctx = ctx
	wf.MsgC = make(chan *protocol.Envelope)
	wf.h = host
	wf.pushHandler = handler
	wf.isFullNode = isFullNode

	wf.h.SetStreamHandlerMatch(FilterID_v20beta1, protocol.PrefixTextMatch(string(FilterID_v20beta1)), wf.onRequest)
	go wf.FilterListener()

	if wf.isFullNode {
		log.Info("Filter protocol started")
	} else {
		log.Info("Filter protocol started (only client mode)")
	}

	return wf
}

func (wf *WakuFilter) FilterListener() {
	// This function is invoked for each message received
	// on the full node in context of Waku2-Filter
	handle := func(envelope *protocol.Envelope) error { // async
		msg := envelope.Message()
		topic := envelope.PubsubTopic()
		// Each subscriber is a light node that earlier on invoked
		// a FilterRequest on this node
		for _, subscriber := range wf.subscribers {
			if subscriber.filter.Topic != "" && subscriber.filter.Topic != topic {
				log.Info("Subscriber's filter pubsubTopic does not match message topic", subscriber.filter.Topic, topic)
				continue
			}

			for _, filter := range subscriber.filter.ContentFilters {
				if msg.ContentTopic == filter.ContentTopic {
					log.Info("found matching contentTopic ", filter, msg)
					msgArr := []*pb.WakuMessage{msg}
					// Do a message push to light node
					pushRPC := &pb.FilterRPC{RequestId: subscriber.requestId, Push: &pb.MessagePush{Messages: msgArr}}
					log.Info("pushing a message to light node: ", pushRPC)

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
func (wf *WakuFilter) Subscribe(ctx context.Context, filter ContentFilter, opts ...FilterSubscribeOption) (subscription *FilterSubscription, err error) {
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

func (wf *WakuFilter) Unsubscribe(ctx context.Context, filter ContentFilter, peer peer.ID) error {
	conn, err := wf.h.NewStream(ctx, peer, FilterID_v20beta1)

	if err != nil {
		return err
	}

	defer conn.Close()

	// This is the only successful path to subscription
	id := protocol.GenerateRequestId()

	var contentFilters []*pb.FilterRequest_ContentFilter
	for _, ct := range filter.ContentTopics {
		contentFilters = append(contentFilters, &pb.FilterRequest_ContentFilter{ContentTopic: ct})
	}

	request := pb.FilterRequest{
		Subscribe:      false,
		Topic:          filter.Topic,
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
}
