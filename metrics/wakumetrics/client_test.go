package wakumetrics

import (
	"errors"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	messagingtypes "github.com/status-im/status-go/messaging/types"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	wakutypes "github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv2"
)

var (
	errTest = errors.New("test error")
)

func createTestClient(t *testing.T) *Client {
	client, err := NewClient(WithPeerID("test-key"), WithDeviceType("test-node"), WithVersion("test-version"))
	require.NoError(t, err)
	require.NoError(t, client.RegisterWithRegistry())

	t.Cleanup(func() {
		require.NoError(t, UnregisterMetrics())
	})

	client.SetDeviceType("test-node")
	return client
}

func createTestMessage(pubsubTopic string, contentTopic wakutypes.TopicType, payload []byte) *wakutypes.Message {
	return &wakutypes.Message{
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

func getGaugeVecValue(metric *prometheus.GaugeVec, labels ...string) float64 {
	m := metric.WithLabelValues(labels...)
	pb := &dto.Metric{}
	err := m.(prometheus.Metric).Write(pb)
	if err != nil {
		return 0
	}
	return pb.Gauge.GetValue()
}

func TestClient_DoubleRegister(t *testing.T) {
	client := createTestClient(t)
	require.Error(t, client.RegisterWithRegistry())
}

func TestClient_PushReceivedMessages(t *testing.T) {
	client := createTestClient(t)

	filter := messagingtypes.ChatFilter{
		PubsubTopic:  "test-pubsub",
		ContentTopic: wakutypes.StringToTopic("test-content"),
		ChatID:       "test-chat",
	}

	shhMessage := createTestMessage("test-pubsub", wakutypes.StringToTopic("test-content"), []byte("test-payload"))

	receivedMessages := ReceivedMessages{
		Filter:     filter,
		SHHMessage: shhMessage,
		Messages:   []*v1protocol.StatusMessage{{}},
	}

	client.PushReceivedMessages(receivedMessages)

	// Verify MessagesReceivedTotal metric
	value := getCounterValue(metrics.MessagesReceivedTotal,
		filter.PubsubTopic,
		filter.ContentTopic.String(),
		filter.ChatID,
	)
	require.Equal(t, float64(1), value)
}

func TestClient_PushPeerCountByOrigin(t *testing.T) {
	client := createTestClient(t)

	peerCountByOrigin := map[wps.Origin]uint{
		wps.Discv5:       5,
		wps.Static:       3,
		wps.PeerExchange: 2,
		wps.Unknown:      1,
		wps.DNSDiscovery: 1,
		wps.Rendezvous:   1,
		wps.PeerManager:  1,
	}

	client.PushPeerCountByOrigin(peerCountByOrigin)

	// Verify metrics for each origin
	for origin, expectedCount := range peerCountByOrigin {
		value := getGaugeVecValue(metrics.PeersByOrigin, getOriginString(origin))
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
		value := getGaugeVecValue(metrics.PeersByShard, strconv.FormatUint(uint64(shard), 10))
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

	value := getCounterValue(metrics.EnvelopeSentErrors,
		envelope.PubsubTopic(),
		envelope.Message().ContentTopic,
	)
	require.Equal(t, float64(1), value)
}

func TestClient_PushRawMessageByType(t *testing.T) {
	client := createTestClient(t)

	pubsubTopic := "test-pubsub-topic"
	contentTopic := "test-content-topic"
	messageType := "test-message-type"
	client.PushRawMessageByType(pubsubTopic, contentTopic, messageType, 10)

	value := getCounterValue(metrics.RawMessagesSentTotal, messageType, pubsubTopic, contentTopic)
	require.Equal(t, float64(1), value)

	valueBytes := getCounterValue(metrics.RawMessagesSizeBytes, messageType, pubsubTopic, contentTopic)
	require.Equal(t, float64(10), valueBytes)
}
