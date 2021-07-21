package filter

import (
	"context"
	"encoding/hex"
	"fmt"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/event"
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

type (
	ContentFilterChan chan *protocol.Envelope

	Filter struct {
		Topic          string
		ContentFilters []*pb.FilterRequest_ContentFilter
		Chan           ContentFilterChan
	}
	// @TODO MAYBE MORE INFO?
	Filters map[string]Filter

	Subscriber struct {
		peer      string
		requestId string
		filter    pb.FilterRequest // @TODO MAKE THIS A SEQUENCE AGAIN?
	}

	MessagePushHandler func(requestId string, msg pb.MessagePush)

	WakuFilter struct {
		ctx         context.Context
		h           host.Host
		subscribers []Subscriber
		pushHandler MessagePushHandler
		MsgC        chan *protocol.Envelope
		peerChan    chan *event.EvtPeerConnectednessChanged
	}
)

// NOTE This is just a start, the design of this protocol isn't done yet. It
// should be direct payload exchange (a la req-resp), not be coupled with the
// relay protocol.

const WakuFilterCodec = "/vac/waku/filter/2.0.0-beta1"

const WakuFilterProtocolId = libp2pProtocol.ID(WakuFilterCodec)

// Error types (metric label values)
const (
	dialFailure      = "dial_failure"
	decodeRpcFailure = "decode_rpc_failure"
)

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
		for _, contentFilter := range filter.ContentFilters {
			if msg.ContentTopic == contentFilter.ContentTopic {
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

	log.Info(fmt.Sprintf("%s: Received query from %s", s.Conn().LocalPeer(), s.Conn().RemotePeer()))

	stats.Record(wf.ctx, metrics.Messages.M(1))

	if filterRPCRequest.Request != nil {
		// We're on a full node.
		// This is a filter request coming from a light node.
		if filterRPCRequest.Request.Subscribe {
			subscriber := Subscriber{peer: string(s.Conn().RemotePeer()), requestId: filterRPCRequest.RequestId, filter: *filterRPCRequest.Request}
			wf.subscribers = append(wf.subscribers, subscriber)
			log.Info("Full node, add a filter subscriber ", subscriber)

			stats.Record(wf.ctx, metrics.FilterSubscriptions.M(int64(len(wf.subscribers))))
		} else {
			peerId := string(s.Conn().RemotePeer())
			log.Info("Full node, remove a filter subscriber ", peerId)
			contentFilters := filterRPCRequest.Request.ContentFilters
			var peerIdsToRemove []string
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
	} else if filterRPCRequest.Push != nil {
		// We're on a light node.
		// This is a message push coming from a full node.

		log.Info("Light node, received a message push ", *filterRPCRequest.Push)
		wf.pushHandler(filterRPCRequest.RequestId, *filterRPCRequest.Push)
	}

}

func (wf *WakuFilter) peerListener() {
	for e := range wf.peerChan {
		if e.Connectedness == network.NotConnected {
			log.Info("Filter Notification received ", e.Peer)
			i := 0
			// Delete subscribers matching deleted peer
			for _, s := range wf.subscribers {
				if s.peer != string(e.Peer) {
					wf.subscribers[i] = s
					i++
				}
			}

			log.Info("Filter, deleted subscribers: ", len(wf.subscribers)-i)
			wf.subscribers = wf.subscribers[:i]
		}
	}
}

func NewWakuFilter(ctx context.Context, host host.Host, handler MessagePushHandler, peerChan chan *event.EvtPeerConnectednessChanged) *WakuFilter {
	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyType, "filter"))
	if err != nil {
		log.Error(err)
	}

	wf := new(WakuFilter)
	wf.ctx = ctx
	wf.MsgC = make(chan *protocol.Envelope)
	wf.h = host
	wf.pushHandler = handler
	wf.peerChan = peerChan

	wf.h.SetStreamHandler(WakuFilterProtocolId, wf.onRequest)
	go wf.FilterListener()
	go wf.peerListener()

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
					log.Info("Found matching contentTopic ", filter, msg)
					msgArr := []*pb.WakuMessage{msg}
					// Do a message push to light node
					pushRPC := &pb.FilterRPC{RequestId: subscriber.requestId, Push: &pb.MessagePush{Messages: msgArr}}
					log.Info("Pushing a message to light node: ", pushRPC)

					conn, err := wf.h.NewStream(wf.ctx, peer.ID(subscriber.peer), WakuFilterProtocolId)

					if err != nil {
						// @TODO more sophisticated error handling here
						log.Error("Failed to open peer stream")
						//waku_filter_errors.inc(labelValues = [dialFailure])
						return err
					}
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
		handle(m)
	}

}

// Having a FilterRequest struct,
// select a peer with filter support, dial it,
// and submit FilterRequest wrapped in FilterRPC
func (wf *WakuFilter) Subscribe(ctx context.Context, request pb.FilterRequest) (string, error) { //.async, gcsafe.} {
	peer, err := utils.SelectPeer(wf.h, string(WakuFilterProtocolId))
	if err == nil {
		conn, err := wf.h.NewStream(ctx, *peer, WakuFilterProtocolId)

		if conn != nil {
			// This is the only successful path to subscription
			id := protocol.GenerateRequestId()

			writer := protoio.NewDelimitedWriter(conn)
			filterRPC := &pb.FilterRPC{RequestId: hex.EncodeToString(id), Request: &request}
			log.Info("Sending filterRPC: ", filterRPC)
			err = writer.WriteMsg(filterRPC)
			return string(id), nil
		} else {
			// @TODO more sophisticated error handling here
			log.Error("failed to connect to remote peer")
			//waku_filter_errors.inc(labelValues = [dialFailure])
			return "", err
		}
	} else {
		log.Info("Error selecting peer: ", err)
	}

	return "", nil
}

func (wf *WakuFilter) Unsubscribe(ctx context.Context, request pb.FilterRequest) {
	// @TODO: NO REAL REASON TO GENERATE REQUEST ID FOR UNSUBSCRIBE OTHER THAN CREATING SANE-LOOKING RPC.
	peer, err := utils.SelectPeer(wf.h, string(WakuFilterProtocolId))
	if err == nil {
		conn, err := wf.h.NewStream(ctx, *peer, WakuFilterProtocolId)

		if conn != nil {
			// This is the only successful path to subscription
			id := protocol.GenerateRequestId()

			writer := protoio.NewDelimitedWriter(conn)
			filterRPC := &pb.FilterRPC{RequestId: hex.EncodeToString(id), Request: &request}
			err = writer.WriteMsg(filterRPC)
			//return some(id)
		} else {
			// @TODO more sophisticated error handling here
			log.Error("failed to connect to remote peer", err)
			//waku_filter_errors.inc(labelValues = [dialFailure])
		}
	} else {
		log.Info("Error selecting peer: ", err)
	}
}
