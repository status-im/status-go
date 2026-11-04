package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

func makeRelayService(t *testing.T, mux *chi.Mux) *RelayService {
	options := node.WithWakuRelayAndMinPeers(0)
	n, err := node.New(options)
	require.NoError(t, err)
	err = n.Start(context.Background())
	require.NoError(t, err)

	return NewRelayService(n, mux, 3, utils.Logger())
}

func TestPostV1Message(t *testing.T) {
	router := chi.NewRouter()
	testTopic := "test"

	r := makeRelayService(t, router)
	msg := &RestWakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: "abc",
		Timestamp:    utils.GetUnixEpoch(),
	}
	msgJSONBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	_, err = r.node.Relay().Subscribe(context.Background(), protocol.NewContentFilter(testTopic))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/relay/v1/messages/"+testTopic, bytes.NewReader(msgJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())
}

func TestRelaySubscription(t *testing.T) {
	router := chi.NewRouter()

	r := makeRelayService(t, router)

	// Wait for node to start
	time.Sleep(500 * time.Millisecond)

	topics := []string{"test"}
	topicsJSONBytes, err := json.Marshal(topics)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, routeRelayV1Subscriptions, bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())

	// Test max messages in subscription
	now := *utils.GetUnixEpoch()
	_, err = r.node.Relay().Publish(context.Background(),
		tests.CreateWakuMessage("test", proto.Int64(now+1)), relay.WithPubSubTopic("test"))
	require.NoError(t, err)
	_, err = r.node.Relay().Publish(context.Background(),
		tests.CreateWakuMessage("test", proto.Int64(now+2)), relay.WithPubSubTopic("test"))
	require.NoError(t, err)

	_, err = r.node.Relay().Publish(context.Background(),
		tests.CreateWakuMessage("test", proto.Int64(now+3)), relay.WithPubSubTopic("test"))
	require.NoError(t, err)

	// Wait for the messages to be processed
	time.Sleep(5 * time.Millisecond)

	// Test deletion
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, routeRelayV1Subscriptions, bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())
}

func TestRelayGetV1Messages(t *testing.T) {
	router := chi.NewRouter()
	router1 := chi.NewRouter()

	serviceA := makeRelayService(t, router)

	serviceB := makeRelayService(t, router1)

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", serviceB.node.Host().ID().String()))
	require.NoError(t, err)

	var addr multiaddr.Multiaddr
	for _, a := range serviceB.node.Host().Addrs() {
		addr = a.Encapsulate(hostInfo)
		break
	}
	err = serviceA.node.DialPeerWithMultiAddress(context.Background(), addr)
	require.NoError(t, err)

	// Wait for the dial to complete
	time.Sleep(1 * time.Second)

	topics := []string{"test"}
	topicsJSONBytes, err := json.Marshal(topics)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, routeRelayV1Subscriptions, bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Wait for the subscription to be started
	time.Sleep(1 * time.Second)
	ephemeral := true
	msg := &RestWakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: "test",
		Timestamp:    utils.GetUnixEpoch(),
		Ephemeral:    &ephemeral,
	}
	msgJsonBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/relay/v1/messages/test", bytes.NewReader(msgJsonBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Wait for the message to be received
	time.Sleep(1 * time.Second)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/relay/v1/messages/test", bytes.NewReader([]byte{}))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var messages []*pb.WakuMessage
	err = json.Unmarshal(rr.Body.Bytes(), &messages)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, *messages[0].Ephemeral, true)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/relay/v1/messages/test", bytes.NewReader([]byte{}))
	router1.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

}

func TestPostAutoV1Message(t *testing.T) {
	router := chi.NewRouter()

	_ = makeRelayService(t, router)
	msg := &RestWakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: "/toychat/1/huilong/proto",
		Timestamp:    utils.GetUnixEpoch(),
	}
	msgJSONBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, routeRelayV1AutoMessages, bytes.NewReader(msgJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestRelayAutoSubUnsub(t *testing.T) {
	router := chi.NewRouter()

	r := makeRelayService(t, router)

	// Wait for node to start
	time.Sleep(500 * time.Millisecond)

	cTopic1 := "/toychat/1/huilong/proto"

	cTopics := []string{cTopic1}
	topicsJSONBytes, err := json.Marshal(cTopics)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, routeRelayV1AutoSubscriptions, bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())

	// Test publishing messages after subscription
	now := *utils.GetUnixEpoch()
	_, err = r.node.Relay().Publish(context.Background(),
		tests.CreateWakuMessage(cTopic1, proto.Int64(now+1)))
	require.NoError(t, err)

	// Wait for the messages to be processed
	time.Sleep(5 * time.Millisecond)

	// Test deletion
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, routeRelayV1AutoSubscriptions, bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())

	cTopics = append(cTopics, "test")
	topicsJSONBytes, err = json.Marshal(cTopics)
	require.NoError(t, err)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, routeRelayV1AutoSubscriptions, bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

}

func TestRelayGetV1AutoMessages(t *testing.T) {
	router := chi.NewRouter()
	router1 := chi.NewRouter()

	serviceA := makeRelayService(t, router)

	serviceB := makeRelayService(t, router1)

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", serviceB.node.Host().ID().String()))
	require.NoError(t, err)

	var addr multiaddr.Multiaddr
	for _, a := range serviceB.node.Host().Addrs() {
		addr = a.Encapsulate(hostInfo)
		break
	}
	err = serviceA.node.DialPeerWithMultiAddress(context.Background(), addr)
	require.NoError(t, err)

	// Wait for the dial to complete
	time.Sleep(1 * time.Second)

	cTopic1 := "/toychat/1/huilong/proto"

	cTopics := []string{cTopic1}
	topicsJSONBytes, err := json.Marshal(cTopics)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, routeRelayV1AutoSubscriptions, bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())

	// Wait for the subscription to be started
	time.Sleep(1 * time.Second)

	msg := &RestWakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: cTopic1,
		Timestamp:    utils.GetUnixEpoch(),
	}
	msgJsonBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, routeRelayV1AutoMessages, bytes.NewReader(msgJsonBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Wait for the message to be received
	time.Sleep(1 * time.Second)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", routeRelayV1AutoMessages, url.QueryEscape(cTopic1)), bytes.NewReader([]byte{}))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var messages []*pb.WakuMessage
	err = json.Unmarshal(rr.Body.Bytes(), &messages)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", routeRelayV1AutoMessages, url.QueryEscape(cTopic1)), bytes.NewReader([]byte{}))
	router1.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

}
