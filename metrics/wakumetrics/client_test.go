package wakumetrics

import (
	"errors"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/wakuv2"
	"github.com/waku-org/go-waku/waku/v2/api/publish"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

var (
	errTest = errors.New("test error")
)

func init() {
	// Register metrics with test registry
	err := RegisterMetrics()
	if err != nil {
		panic("Failed to register metrics for testing: " + err.Error())
	}
}

func createTestClient(t *testing.T) *Client {
	client, err := NewClient(WithPeerID("test-key"), WithDeviceType("test-node"), WithVersion("test-version"))
	require.NoError(t, err)
	return client
}

func createTestMessage(pubsubTopic string, contentTopic types.TopicType, payload []byte) *types.Message {
	return &types.Message{
		PubsubTopic: pubsubTopic,
		Topic:       contentTopic,
		Payload:     payload,
	}
}

func getCounterValue(metric *prometheus.CounterVec, labels ...string) float64 {
	m := metric.WithLabelValues(labels...)
	pb := &dto.Metric{}
	err := m.(prometheus.Metric).Write(pb)
	if err != nil {
		return 0
	}
	return pb.Counter.GetValue()
}

func getGaugeValue(metric prometheus.Gauge) float64 {
	pb := &dto.Metric{}
	err := metric.(prometheus.Metric).Write(pb)
	if err != nil {
		return 0
	}
	return pb.Gauge.GetValue()
}

func getGaugeVecValue(metric *prometheus.GaugeVec, labels ...string) float64 {
	m := metric.WithLabelValues(labels...)
	pb := &dto.Metric{}
	err := m.(prometheus.Metric).Write(pb)
	if err != nil {
		return 0
	}
	return pb.Gauge.GetValue()
}

func getHistogramValue(metric *prometheus.HistogramVec, labels ...string) float64 {
	m := metric.WithLabelValues(labels...)
	pb := &dto.Metric{}
	err := m.(prometheus.Metric).Write(pb)
	if err != nil {
		return 0
	}
	return pb.Histogram.GetSampleSum()
}

func TestClient_PushReceivedMessages(t *testing.T) {
	client := createTestClient(t)

	filter := transport.Filter{
		PubsubTopic:  "test-pubsub",
		ContentTopic: types.StringToTopic("test-content"),
		ChatID:       "test-chat",
	}

	sshMessage := createTestMessage("test-pubsub", types.StringToTopic("test-content"), []byte("test-payload"))

	receivedMessages := ReceivedMessages{
		Filter:     filter,
		SSHMessage: sshMessage,
	}

	client.PushReceivedMessages(receivedMessages)

	// Verify MessagesReceivedTotal metric
	value := getHistogramValue(MessagesReceivedTotal,
		filter.PubsubTopic,
		filter.ContentTopic.String(),
		filter.ChatID,
	)
	require.Equal(t, float64(1), value)
}

func TestClient_PushPeerCount(t *testing.T) {
	client := createTestClient(t)

	client.PushPeerCount(5)
	value := getGaugeValue(ConnectedPeers)
	require.Equal(t, float64(5), value)
}

func TestClient_PushPeerCountByOrigin(t *testing.T) {
	client := createTestClient(t)

	peerCountByOrigin := map[wps.Origin]uint{
		wps.Discv5:       5,
		wps.Static:       3,
		wps.PeerExchange: 2,
	}

	client.PushPeerCountByOrigin(peerCountByOrigin)

	// Verify metrics for each origin
	for origin, expectedCount := range peerCountByOrigin {
		value := getGaugeVecValue(PeersByOrigin, getOriginString(origin))
		require.Equal(t, float64(expectedCount), value)
	}
}

func TestClient_PushPeerCountByShard(t *testing.T) {
	client := createTestClient(t)

	peerCountByShard := map[uint16]uint{
		1: 5,
		2: 3,
		3: 2,
	}

	client.PushPeerCountByShard(peerCountByShard)

	// Verify metrics for each shard
	for shard, expectedCount := range peerCountByShard {
		value := getGaugeVecValue(PeersByShard, strconv.FormatUint(uint64(shard), 10))
		require.Equal(t, float64(expectedCount), value)
	}
}

func TestClient_PushErrorSendingEnvelope(t *testing.T) {
	client := createTestClient(t)

	msg := &pb.WakuMessage{
		Payload:      []byte("test-payload"),
		ContentTopic: "test-content",
	}
	envelope := v2protocol.NewEnvelope(msg, 0, "")

	errorSendingEnvelope := wakuv2.ErrorSendingEnvelope{
		SentEnvelope: wakuv2.SentEnvelope{
			Envelope:      envelope,
			PublishMethod: publish.LightPush,
		},
		Error: errTest,
	}

	client.PushErrorSendingEnvelope(errorSendingEnvelope)

	value := getCounterValue(EnvelopeSentErrors,
		envelope.PubsubTopic(),
		envelope.Message().ContentTopic,
	)
	require.Equal(t, float64(1), value)
}
