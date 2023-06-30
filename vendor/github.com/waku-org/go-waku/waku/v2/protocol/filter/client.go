package filter

import (
	"context"
	"encoding/hex"
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
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter/pb"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
)

// FilterPushID_v20beta1 is the current Waku Filter protocol identifier used to allow
// filter service nodes to push messages matching registered subscriptions to this client.
const FilterPushID_v20beta1 = libp2pProtocol.ID("/vac/waku/filter-push/2.0.0-beta1")

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
)

type WakuFilterLightnode struct {
	cancel        context.CancelFunc
	ctx           context.Context
	h             host.Host
	broadcaster   relay.Broadcaster
	timesource    timesource.Timesource
	wg            *sync.WaitGroup
	log           *zap.Logger
	subscriptions *SubscriptionsMap
}

type ContentFilter struct {
	Topic         string
	ContentTopics []string
}

type WakuFilterPushResult struct {
	Err    error
	PeerID peer.ID
}

// NewWakuRelay returns a new instance of Waku Filter struct setup according to the chosen parameter and options
func NewWakuFilterLightnode(broadcaster relay.Broadcaster, timesource timesource.Timesource, log *zap.Logger) *WakuFilterLightnode {
	wf := new(WakuFilterLightnode)
	wf.log = log.Named("filterv2-lightnode")
	wf.broadcaster = broadcaster
	wf.timesource = timesource
	wf.wg = &sync.WaitGroup{}

	return wf
}

// Sets the host to be able to mount or consume a protocol
func (wf *WakuFilterLightnode) SetHost(h host.Host) {
	wf.h = h
}

func (wf *WakuFilterLightnode) Start(ctx context.Context) error {
	wf.wg.Wait() // Wait for any goroutines to stop

	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyType, "filter"))
	if err != nil {
		wf.log.Error("creating tag map", zap.Error(err))
		return errors.New("could not start waku filter")
	}

	ctx, cancel := context.WithCancel(ctx)
	wf.cancel = cancel
	wf.ctx = ctx
	wf.subscriptions = NewSubscriptionMap(wf.log)

	wf.h.SetStreamHandlerMatch(FilterPushID_v20beta1, protocol.PrefixTextMatch(string(FilterPushID_v20beta1)), wf.onRequest(ctx))

	wf.log.Info("filter-push protocol started")

	return nil
}

// Stop unmounts the filter protocol
func (wf *WakuFilterLightnode) Stop() {
	if wf.cancel == nil {
		return
	}

	wf.cancel()

	wf.h.RemoveStreamHandler(FilterPushID_v20beta1)

	_, _ = wf.UnsubscribeAll(wf.ctx)

	wf.subscriptions.Clear()

	wf.wg.Wait()
}

func (wf *WakuFilterLightnode) onRequest(ctx context.Context) func(s network.Stream) {
	return func(s network.Stream) {
		defer s.Close()
		logger := wf.log.With(logging.HostID("peer", s.Conn().RemotePeer()))

		if !wf.subscriptions.IsSubscribedTo(s.Conn().RemotePeer()) {
			logger.Warn("received message push from unknown peer", logging.HostID("peerID", s.Conn().RemotePeer()))
			metrics.RecordFilterError(ctx, "unknown_peer_messagepush")
			return
		}

		reader := pbio.NewDelimitedReader(s, math.MaxInt32)

		messagePush := &pb.MessagePushV2{}
		err := reader.ReadMsg(messagePush)
		if err != nil {
			logger.Error("reading message push", zap.Error(err))
			metrics.RecordFilterError(ctx, "decode_rpc_failure")
			return
		}

		if !wf.subscriptions.Has(s.Conn().RemotePeer(), messagePush.PubsubTopic, messagePush.WakuMessage.ContentTopic) {
			logger.Warn("received messagepush with invalid subscription parameters", logging.HostID("peerID", s.Conn().RemotePeer()), zap.String("topic", messagePush.PubsubTopic), zap.String("contentTopic", messagePush.WakuMessage.ContentTopic))
			metrics.RecordFilterError(ctx, "invalid_subscription_message")
			return
		}

		metrics.RecordFilterMessage(ctx, "PushMessage", 1)

		wf.notify(s.Conn().RemotePeer(), messagePush.PubsubTopic, messagePush.WakuMessage)

		logger.Info("received message push")
	}
}

func (wf *WakuFilterLightnode) notify(remotePeerID peer.ID, pubsubTopic string, msg *wpb.WakuMessage) {
	envelope := protocol.NewEnvelope(msg, wf.timesource.Now().UnixNano(), pubsubTopic)

	// Broadcasting message so it's stored
	wf.broadcaster.Submit(envelope)

	// Notify filter subscribers
	wf.subscriptions.Notify(remotePeerID, envelope)
}

func (wf *WakuFilterLightnode) request(ctx context.Context, params *FilterSubscribeParameters, reqType pb.FilterSubscribeRequest_FilterSubscribeType, contentFilter ContentFilter) error {
	conn, err := wf.h.NewStream(ctx, params.selectedPeer, FilterSubscribeID_v20beta1)
	if err != nil {
		metrics.RecordFilterError(ctx, "dial_failure")
		return err
	}
	defer conn.Close()

	writer := pbio.NewDelimitedWriter(conn)
	reader := pbio.NewDelimitedReader(conn, math.MaxInt32)

	request := &pb.FilterSubscribeRequest{
		RequestId:           hex.EncodeToString(params.requestId),
		FilterSubscribeType: reqType,
		PubsubTopic:         contentFilter.Topic,
		ContentTopics:       contentFilter.ContentTopics,
	}

	wf.log.Debug("sending FilterSubscribeRequest", zap.Stringer("request", request))
	err = writer.WriteMsg(request)
	if err != nil {
		metrics.RecordFilterError(ctx, "write_request_failure")
		wf.log.Error("sending FilterSubscribeRequest", zap.Error(err))
		return err
	}

	filterSubscribeResponse := &pb.FilterSubscribeResponse{}
	err = reader.ReadMsg(filterSubscribeResponse)
	if err != nil {
		wf.log.Error("receiving FilterSubscribeResponse", zap.Error(err))
		metrics.RecordFilterError(ctx, "decode_rpc_failure")
		return err
	}

	if filterSubscribeResponse.RequestId != request.RequestId {
		wf.log.Error("requestId mismatch", zap.String("expected", request.RequestId), zap.String("received", filterSubscribeResponse.RequestId))
		metrics.RecordFilterError(ctx, "request_id_mismatch")
		err := NewFilterError(300, "request_id_mismatch")
		return &err
	}

	if filterSubscribeResponse.StatusCode != http.StatusOK {
		metrics.RecordFilterError(ctx, "error_response")
		err := NewFilterError(int(filterSubscribeResponse.StatusCode), filterSubscribeResponse.StatusDesc)
		return &err
	}

	return nil
}

// Subscribe setups a subscription to receive messages that match a specific content filter
func (wf *WakuFilterLightnode) Subscribe(ctx context.Context, contentFilter ContentFilter, opts ...FilterSubscribeOption) (*SubscriptionDetails, error) {
	if contentFilter.Topic == "" {
		return nil, errors.New("topic is required")
	}

	if len(contentFilter.ContentTopics) == 0 {
		return nil, errors.New("at least one content topic is required")
	}

	if len(contentFilter.ContentTopics) > MaxContentTopicsPerRequest {
		return nil, fmt.Errorf("exceeds maximum content topics: %d", MaxContentTopicsPerRequest)
	}

	params := new(FilterSubscribeParameters)
	params.log = wf.log
	params.host = wf.h

	optList := DefaultSubscriptionOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	if params.selectedPeer == "" {
		metrics.RecordFilterError(ctx, "peer_not_found_failure")
		return nil, ErrNoPeersAvailable
	}

	err := wf.request(ctx, params, pb.FilterSubscribeRequest_SUBSCRIBE, contentFilter)
	if err != nil {
		return nil, err
	}

	return wf.subscriptions.NewSubscription(params.selectedPeer, contentFilter.Topic, contentFilter.ContentTopics), nil
}

// FilterSubscription is used to obtain an object from which you could receive messages received via filter protocol
func (wf *WakuFilterLightnode) FilterSubscription(peerID peer.ID, contentFilter ContentFilter) (*SubscriptionDetails, error) {
	if !wf.subscriptions.Has(peerID, contentFilter.Topic, contentFilter.ContentTopics...) {
		return nil, errors.New("subscription does not exist")
	}

	return wf.subscriptions.NewSubscription(peerID, contentFilter.Topic, contentFilter.ContentTopics), nil
}

func (wf *WakuFilterLightnode) getUnsubscribeParameters(opts ...FilterUnsubscribeOption) (*FilterUnsubscribeParameters, error) {
	params := new(FilterUnsubscribeParameters)
	params.log = wf.log
	for _, opt := range opts {
		opt(params)
	}

	if !params.unsubscribeAll && params.selectedPeer == "" {
		return nil, ErrNoPeersAvailable
	}

	return params, nil
}

func (wf *WakuFilterLightnode) Ping(ctx context.Context, peerID peer.ID) error {
	return wf.request(
		ctx,
		&FilterSubscribeParameters{selectedPeer: peerID},
		pb.FilterSubscribeRequest_SUBSCRIBER_PING,
		ContentFilter{})
}

func (wf *WakuFilterLightnode) IsSubscriptionAlive(ctx context.Context, subscription *SubscriptionDetails) error {
	return wf.Ping(ctx, subscription.PeerID)
}

func (wf *WakuFilterLightnode) Subscriptions() []*SubscriptionDetails {
	wf.subscriptions.RLock()
	defer wf.subscriptions.RUnlock()

	var output []*SubscriptionDetails

	for _, peerSubscription := range wf.subscriptions.items {
		for _, subscriptionPerTopic := range peerSubscription.subscriptionsPerTopic {
			for _, subscriptionDetail := range subscriptionPerTopic {
				output = append(output, subscriptionDetail)
			}
		}
	}

	return output
}

func (wf *WakuFilterLightnode) cleanupSubscriptions(peerID peer.ID, contentFilter ContentFilter) {
	wf.subscriptions.Lock()
	defer wf.subscriptions.Unlock()

	peerSubscription, ok := wf.subscriptions.items[peerID]
	if !ok {
		return
	}

	subscriptionDetailList, ok := peerSubscription.subscriptionsPerTopic[contentFilter.Topic]
	if !ok {
		return
	}

	for subscriptionDetailID, subscriptionDetail := range subscriptionDetailList {
		subscriptionDetail.Remove(contentFilter.ContentTopics...)
		if len(subscriptionDetail.ContentTopics) == 0 {
			delete(subscriptionDetailList, subscriptionDetailID)
		} else {
			subscriptionDetailList[subscriptionDetailID] = subscriptionDetail
		}
	}

	if len(subscriptionDetailList) == 0 {
		delete(wf.subscriptions.items[peerID].subscriptionsPerTopic, contentFilter.Topic)
	} else {
		wf.subscriptions.items[peerID].subscriptionsPerTopic[contentFilter.Topic] = subscriptionDetailList
	}

	if len(wf.subscriptions.items[peerID].subscriptionsPerTopic) == 0 {
		delete(wf.subscriptions.items, peerID)
	}

}

// Unsubscribe is used to stop receiving messages from a peer that match a content filter
func (wf *WakuFilterLightnode) Unsubscribe(ctx context.Context, contentFilter ContentFilter, opts ...FilterUnsubscribeOption) (<-chan WakuFilterPushResult, error) {
	if contentFilter.Topic == "" {
		return nil, errors.New("topic is required")
	}

	if len(contentFilter.ContentTopics) == 0 {
		return nil, errors.New("at least one content topic is required")
	}

	if len(contentFilter.ContentTopics) > MaxContentTopicsPerRequest {
		return nil, fmt.Errorf("exceeds maximum content topics: %d", MaxContentTopicsPerRequest)
	}

	params, err := wf.getUnsubscribeParameters(opts...)
	if err != nil {
		return nil, err
	}

	localWg := sync.WaitGroup{}
	resultChan := make(chan WakuFilterPushResult, len(wf.subscriptions.items))

	for peerID := range wf.subscriptions.items {
		if !params.unsubscribeAll && peerID != params.selectedPeer {
			continue
		}

		localWg.Add(1)
		go func(peerID peer.ID) {
			defer localWg.Done()
			err := wf.request(
				ctx,
				&FilterSubscribeParameters{selectedPeer: peerID},
				pb.FilterSubscribeRequest_UNSUBSCRIBE,
				contentFilter)
			if err != nil {
				wf.log.Error("could not unsubscribe from peer", logging.HostID("peerID", peerID), zap.Error(err))
				return
			}

			wf.cleanupSubscriptions(peerID, contentFilter)

			resultChan <- WakuFilterPushResult{
				Err:    err,
				PeerID: peerID,
			}
		}(peerID)
	}

	localWg.Wait()
	close(resultChan)

	return resultChan, nil
}

// Unsubscribe is used to stop receiving messages from a peer that match a content filter
func (wf *WakuFilterLightnode) UnsubscribeWithSubscription(ctx context.Context, sub *SubscriptionDetails, opts ...FilterUnsubscribeOption) (<-chan WakuFilterPushResult, error) {
	var contentTopics []string
	for k := range sub.ContentTopics {
		contentTopics = append(contentTopics, k)
	}

	opts = append(opts, Peer(sub.PeerID))

	return wf.Unsubscribe(ctx, ContentFilter{Topic: sub.PubsubTopic, ContentTopics: contentTopics}, opts...)
}

// UnsubscribeAll is used to stop receiving messages from peer(s). It does not close subscriptions
func (wf *WakuFilterLightnode) UnsubscribeAll(ctx context.Context, opts ...FilterUnsubscribeOption) (<-chan WakuFilterPushResult, error) {
	params, err := wf.getUnsubscribeParameters(opts...)
	if err != nil {
		return nil, err
	}

	wf.subscriptions.Lock()
	defer wf.subscriptions.Unlock()

	localWg := sync.WaitGroup{}
	resultChan := make(chan WakuFilterPushResult, len(wf.subscriptions.items))

	for peerID := range wf.subscriptions.items {
		if !params.unsubscribeAll && peerID != params.selectedPeer {
			continue
		}
		localWg.Add(1)
		go func(peerID peer.ID) {
			defer localWg.Done()
			err := wf.request(
				ctx,
				&FilterSubscribeParameters{selectedPeer: peerID},
				pb.FilterSubscribeRequest_UNSUBSCRIBE_ALL,
				ContentFilter{})
			if err != nil {
				wf.log.Error("could not unsubscribe from peer", logging.HostID("peerID", peerID), zap.Error(err))
			}

			wf.subscriptions.Lock()
			delete(wf.subscriptions.items, peerID)
			defer wf.subscriptions.Unlock()

			resultChan <- WakuFilterPushResult{
				Err:    err,
				PeerID: peerID,
			}
		}(peerID)
	}

	localWg.Wait()
	close(resultChan)

	return resultChan, nil
}
