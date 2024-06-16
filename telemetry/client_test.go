package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/wakuv2"
)

func createMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}
		if r.URL.EscapedPath() != "/record-metrics" {
			t.Errorf("Expected request to '/record-metrics', got '%s'", r.URL.EscapedPath())
		}

		// Check the request body is as expected
		var received []TelemetryRequest
		err := json.NewDecoder(r.Body).Decode(&received)
		if err != nil {
			t.Fatal(err)
		}

		if len(received) != 1 {
			t.Errorf("Unexpected data received: %+v", received)
		} else {
			// If the data is as expected, respond with success
			t.Log("Responding with success")
			w.WriteHeader(http.StatusOK)
		}
	}))
}

func TestClient_ProcessReceivedMessages(t *testing.T) {
	// Setup a mock server to handle post requests
	mockServer := createMockServer(t)
	defer mockServer.Close()

	// Create a client with the mock server URL
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	client := NewClient(logger, mockServer.URL, "testUID", "testNode", "1.0")

	// Create a telemetry request to send
	data := ReceivedMessages{
		Filter: transport.Filter{
			ChatID:       "testChat",
			PubsubTopic:  "testTopic",
			ContentTopic: types.StringToTopic("testContentTopic"),
		},
		SSHMessage: &types.Message{
			Hash:      []byte("hash"),
			Timestamp: uint32(time.Now().Unix()),
		},
		Messages: []*v1protocol.StatusMessage{
			{
				ApplicationLayer: v1protocol.ApplicationLayer{
					ID:   types.HexBytes("123"),
					Type: 1,
				},
			},
		},
	}
	telemetryData := client.ProcessReceivedMessages(data)
	telemetryRequest := TelemetryRequest{
		Id:            1,
		TelemetryType: ReceivedMessagesMetric,
		TelemetryData: telemetryData,
	}

	// Send the telemetry request
	client.pushTelemetryRequest([]TelemetryRequest{telemetryRequest})
}

func TestClient_ProcessReceivedEnvelope(t *testing.T) {
	// Setup a mock server to handle post requests
	mockServer := createMockServer(t)
	defer mockServer.Close()

	// Create a client with the mock server URL
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	client := NewClient(logger, mockServer.URL, "testUID", "testNode", "1.0")

	// Create a telemetry request to send
	envelope := v2protocol.NewEnvelope(&pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: "testContentTopic",
		Version:      proto.Uint32(0),
		Timestamp:    proto.Int64(time.Now().Unix()),
	}, 0, "")
	telemetryData := client.ProcessReceivedEnvelope(envelope)
	telemetryRequest := TelemetryRequest{
		Id:            2,
		TelemetryType: ReceivedEnvelopeMetric,
		TelemetryData: telemetryData,
	}

	// Send the telemetry request
	client.pushTelemetryRequest([]TelemetryRequest{telemetryRequest})
}

func TestClient_ProcessSentEnvelope(t *testing.T) {
	// Setup a mock server to handle post requests
	mockServer := createMockServer(t)
	defer mockServer.Close()

	// Create a client with the mock server URL
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	client := NewClient(logger, mockServer.URL, "testUID", "testNode", "1.0")

	// Create a telemetry request to send
	sentEnvelope := wakuv2.SentEnvelope{
		Envelope: v2protocol.NewEnvelope(&pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: "testContentTopic",
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().Unix()),
		}, 0, ""),
		PublishMethod: wakuv2.LightPush,
	}
	telemetryData := client.ProcessSentEnvelope(sentEnvelope)
	telemetryRequest := TelemetryRequest{
		Id:            3,
		TelemetryType: SentEnvelopeMetric,
		TelemetryData: telemetryData,
	}

	// Send the telemetry request
	client.pushTelemetryRequest([]TelemetryRequest{telemetryRequest})
}
