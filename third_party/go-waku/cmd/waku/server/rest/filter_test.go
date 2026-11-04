package rest

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/node"
	wakupeerstore "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"golang.org/x/time/rate"
)

func createNode(t *testing.T, opts ...node.WakuNodeOption) *node.WakuNode {
	node, err := node.New(opts...)
	require.NoError(t, err)

	err = node.Start(context.Background())
	require.NoError(t, err)

	return node
}

// node2 connects to node1
func twoFilterConnectedNodes(t *testing.T, pubSubTopics ...string) (*node.WakuNode, *node.WakuNode) {
	node1 := createNode(t, node.WithWakuFilterFullNode(filter.WithFullNodeRateLimiter(rate.Inf, 0))) // full node filter
	node2 := createNode(t, node.WithWakuFilterLightNode())                                           // light node filter

	node2.Host().Peerstore().AddAddr(node1.Host().ID(), tests.GetHostAddress(node1.Host()), peerstore.PermanentAddrTTL)
	err := node2.Host().Peerstore().AddProtocols(node1.Host().ID(), filter.FilterSubscribeID_v20beta1)
	require.NoError(t, err)

	err = node2.Host().Peerstore().(*wakupeerstore.WakuPeerstoreImpl).SetPubSubTopics(node1.Host().ID(), pubSubTopics)
	require.NoError(t, err)

	return node1, node2
}

// test 400, 404 status code for ping rest endpoint
// both requests are not successful
func TestFilterPingFailure(t *testing.T) {
	node1, node2 := twoFilterConnectedNodes(t)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, 0, utils.Logger())

	// with empty requestID
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", ""), nil)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestID:  "",
		StatusDesc: "bad request id",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusBadRequest, rr.Code)

	// no subscription with peer
	requestID := hex.EncodeToString(protocol.GenerateRequestID())
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", requestID), nil)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestID:  requestID,
		StatusDesc: "peer has no subscription",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusNotFound, rr.Code)
}

// create a filter subscription to the peer and try peer that peer
// both steps should be successful
func TestFilterSubscribeAndPing(t *testing.T) {
	pubsubTopic := "/waku/2/test/proto"
	contentTopics := []string{"test"}
	requestID := hex.EncodeToString(protocol.GenerateRequestID())

	node1, node2 := twoFilterConnectedNodes(t, pubsubTopic)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, 0, utils.Logger())

	// create subscription to peer
	rr := httptest.NewRecorder()
	reqReader := strings.NewReader(toString(t, filterSubscriptionRequest{
		RequestID:      requestID,
		PubsubTopic:    pubsubTopic,
		ContentFilters: contentTopics,
	}))
	req, _ := http.NewRequest(http.MethodPost, filterV2Subscriptions, reqReader)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestID:  requestID,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)

	// trying pinging the peer once there is subscription to it
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", filterV2Subscriptions, requestID), nil)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestID:  requestID,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)
}

// create subscription to peer
// delete the subscription to the peer with matching pubSub and contentTopic
func TestFilterSubscribeAndUnsubscribe(t *testing.T) {
	pubsubTopic := "/waku/2/test/proto"
	contentTopics := []string{"test"}
	requestID := hex.EncodeToString(protocol.GenerateRequestID())

	node1, node2 := twoFilterConnectedNodes(t, pubsubTopic)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, 0, utils.Logger())

	// create subscription to peer
	rr := httptest.NewRecorder()
	reqReader := strings.NewReader(toString(t, filterSubscriptionRequest{
		RequestID:      requestID,
		PubsubTopic:    pubsubTopic,
		ContentFilters: contentTopics,
	}))
	req, _ := http.NewRequest(http.MethodPost, filterV2Subscriptions, reqReader)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestID:  requestID,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)

	// delete the subscription to the peer with matching pubSub and contentTopic
	requestID = hex.EncodeToString(protocol.GenerateRequestID())
	rr = httptest.NewRecorder()
	reqReader = strings.NewReader(toString(t, filterSubscriptionRequest{
		RequestID:      requestID,
		PubsubTopic:    pubsubTopic,
		ContentFilters: contentTopics,
	}))
	req, _ = http.NewRequest(http.MethodDelete, filterV2Subscriptions, reqReader)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestID:  requestID,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)
}

// create 2 subscription from filter client to server
// make a unsubscribeAll request
// try pinging the peer, if 404 is returned then unsubscribeAll was successful
func TestFilterAllUnsubscribe(t *testing.T) {
	pubsubTopic := "/waku/2/test/proto"
	contentTopics1 := "ct_1"
	contentTopics2 := "ct_2"

	node1, node2 := twoFilterConnectedNodes(t, pubsubTopic)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, 0, utils.Logger())

	// create 2 different subscription to peer
	for _, ct := range []string{contentTopics1, contentTopics2} {
		requestID := hex.EncodeToString(protocol.GenerateRequestID())
		rr := httptest.NewRecorder()
		reqReader := strings.NewReader(toString(t, filterSubscriptionRequest{
			RequestID:      requestID,
			PubsubTopic:    pubsubTopic,
			ContentFilters: []string{ct},
		}))
		req, _ := http.NewRequest(http.MethodPost, filterV2Subscriptions, reqReader)
		router.ServeHTTP(rr, req)
		checkJSON(t, filterSubscriptionResponse{
			RequestID:  requestID,
			StatusDesc: "OK",
		}, getFilterResponse(t, rr.Body))
		require.Equal(t, http.StatusOK, rr.Code)
	}

	// delete all subscription to the peer
	requestID := hex.EncodeToString(protocol.GenerateRequestID())
	rr := httptest.NewRecorder()
	reqReader := strings.NewReader(toString(t, filterUnsubscribeAllRequest{
		RequestID: requestID,
	}))
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/all", filterV2Subscriptions), reqReader)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestID:  requestID,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)

	// check if all subscriptions are deleted to the peer are deleted
	requestID = hex.EncodeToString(protocol.GenerateRequestID())
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", filterV2Subscriptions, requestID), nil)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestID:  requestID,
		StatusDesc: "peer has no subscription",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func checkJSON(t *testing.T, expected, actual interface{}) {
	require.JSONEq(t, toString(t, expected), toString(t, actual))
}
func getFilterResponse(t *testing.T, body *bytes.Buffer) filterSubscriptionResponse {
	resp := filterSubscriptionResponse{}
	err := json.Unmarshal(body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}
func getMessageResponse(t *testing.T, body *bytes.Buffer) []*pb.WakuMessage {
	resp := []*pb.WakuMessage{}
	err := json.Unmarshal(body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}
func toString(t *testing.T, data interface{}) string {
	bytes, err := json.Marshal(data)
	require.NoError(t, err)
	return string(bytes)
}

func TestFilterGetMessages(t *testing.T) {
	pubsubTopic := "/waku/2/test/proto"
	contentTopic := "/waku/2/app/1"

	// get nodes add connect them
	generatedPubsubTopic, err := protocol.GetPubSubTopicFromContentTopic(contentTopic)
	require.NoError(t, err)
	node1, node2 := twoFilterConnectedNodes(t, pubsubTopic, generatedPubsubTopic)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	// set router and start filter service
	router := chi.NewRouter()
	service := NewFilterService(node2, router, 2, utils.Logger())
	go service.Start(context.Background())
	defer service.Stop()

	{ // create subscription so that messages are cached
		for _, pubsubTopic := range []string{"", pubsubTopic} {
			requestID := hex.EncodeToString(protocol.GenerateRequestID())
			rr := httptest.NewRecorder()
			reqReader := strings.NewReader(toString(t, filterSubscriptionRequest{
				RequestID:      requestID,
				PubsubTopic:    pubsubTopic,
				ContentFilters: []string{contentTopic},
			}))
			req, _ := http.NewRequest(http.MethodPost, filterV2Subscriptions, reqReader)
			router.ServeHTTP(rr, req)
			checkJSON(t, filterSubscriptionResponse{
				RequestID:  requestID,
				StatusDesc: "OK",
			}, getFilterResponse(t, rr.Body))
			require.Equal(t, http.StatusOK, rr.Code)
		}
	}

	// submit messages
	messageByContentTopic := []*protocol.Envelope{
		genMessage("", contentTopic),
		genMessage("", contentTopic),
		genMessage("", contentTopic),
	}
	messageByPubsubTopic := []*protocol.Envelope{
		genMessage(pubsubTopic, contentTopic),
	}
	for _, envelope := range append(messageByContentTopic, messageByPubsubTopic...) {
		node2.Broadcaster().Submit(envelope)
	}
	time.Sleep(1 * time.Second)

	{ // with malformed contentTopic
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/%s", filterv2Messages, url.QueryEscape("/waku/2/wrongtopic")),
			nil,
		)
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Equal(t, "bad content topic", rr.Body.String())
	}

	{ // with check if the cache is working properly
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/%s", filterv2Messages, url.QueryEscape(contentTopic)),
			nil,
		)
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		checkJSON(t, toMessage(messageByContentTopic[1:]), getMessageResponse(t, rr.Body))
	}

	{ // check if pubsubTopic is present in the url
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s//%s", filterv2Messages, url.QueryEscape(contentTopic)),
			nil,
		)
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Equal(t, "missing pubsubTopic", rr.Body.String())
	}

	{ // check messages by pubsub/contentTopic pair
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/%s/%s", filterv2Messages, url.QueryEscape(pubsubTopic), url.QueryEscape(contentTopic)),
			nil,
		)
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		checkJSON(t, toMessage(messageByPubsubTopic), getMessageResponse(t, rr.Body))
	}

	{ // check if pubsubTopic/contentTOpic is subscribed or not.
		rr := httptest.NewRecorder()
		notSubscibredPubsubTopic := "/waku/2/test2/proto"
		req, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/%s/%s", filterv2Messages, url.QueryEscape(notSubscibredPubsubTopic), url.QueryEscape(contentTopic)),
			nil,
		)
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusNotFound, rr.Code)
		require.Equal(t,
			fmt.Sprintf("not subscribed to pubsubTopic:%s contentTopic: %s", notSubscibredPubsubTopic, contentTopic),
			rr.Body.String(),
		)
	}
}

func toMessage(envs []*protocol.Envelope) []*pb.WakuMessage {
	msgs := make([]*pb.WakuMessage, len(envs))
	for i, env := range envs {
		msgs[i] = env.Message()
	}
	return msgs
}

func genMessage(pubsubTopic, contentTopic string) *protocol.Envelope {
	if pubsubTopic == "" {
		pubsubTopic, _ = protocol.GetPubSubTopicFromContentTopic(contentTopic)
	}
	return protocol.NewEnvelope(
		&pb.WakuMessage{
			Payload:      []byte{1, 2, 3},
			ContentTopic: contentTopic,
			Timestamp:    utils.GetUnixEpoch(),
		},
		0,
		pubsubTopic,
	)
}
