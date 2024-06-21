package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/wakuv2"
)

var (
	testContentTopic = "/waku/1/0x12345679/rfc26"
)

func createMockServer(t *testing.T, wg *sync.WaitGroup, expectedType TelemetryType) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done() // Signal that a request was received

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
			if received[0].TelemetryType != expectedType {
				t.Errorf("Unexpected telemetry type: got %v, want %v", received[0].TelemetryType, expectedType)
			} else {
				// If the data is as expected, respond with success
				t.Log("Responding with success")
				w.WriteHeader(http.StatusOK)
			}
		}
	}))
}

func createClient(t *testing.T, mockServerURL string) *Client {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return NewClient(logger, mockServerURL, "testUID", "testNode", "1.0", WithSendPeriod(500*time.Millisecond))
}

func withMockServer(t *testing.T, expectedType TelemetryType, testFunc func(t *testing.T, client *Client, wg *sync.WaitGroup)) {
	var wg sync.WaitGroup
	wg.Add(1) // Expecting one request

	mockServer := createMockServer(t, &wg, expectedType)
	defer mockServer.Close()

	client := createClient(t, mockServer.URL)

	testFunc(t, client, &wg)

	// Wait for the request to be received
	wg.Wait()
}

func TestClient_ProcessReceivedMessages(t *testing.T) {
	withMockServer(t, ReceivedMessagesMetric, func(t *testing.T, client *Client, wg *sync.WaitGroup) {
		// Create a telemetry request to send
		data := ReceivedMessages{
			Filter: transport.Filter{
				ChatID:       "testChat",
				PubsubTopic:  "testTopic",
				ContentTopic: types.StringToTopic(testContentTopic),
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
	})
}

func TestClient_ProcessReceivedEnvelope(t *testing.T) {
	withMockServer(t, ReceivedEnvelopeMetric, func(t *testing.T, client *Client, wg *sync.WaitGroup) {
		// Create a telemetry request to send
		envelope := v2protocol.NewEnvelope(&pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: testContentTopic,
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
	})
}

func TestClient_ProcessSentEnvelope(t *testing.T) {
	withMockServer(t, SentEnvelopeMetric, func(t *testing.T, client *Client, wg *sync.WaitGroup) {
		// Create a telemetry request to send
		sentEnvelope := wakuv2.SentEnvelope{
			Envelope: v2protocol.NewEnvelope(&pb.WakuMessage{
				Payload:      []byte{1, 2, 3, 4, 5},
				ContentTopic: testContentTopic,
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
	})
}

var (
	testENRBootstrap = "enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@store.staging.shards.nodes.status.im"
)

func TestTelemetryUponPublishError(t *testing.T) {
	withMockServer(t, ErrorSendingEnvelopeMetric, func(t *testing.T, client *Client, wg *sync.WaitGroup) {
		enrTreeAddress := testENRBootstrap
		envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
		if envEnrTreeAddress != "" {
			enrTreeAddress = envEnrTreeAddress
		}

		wakuConfig := &wakuv2.Config{}
		wakuConfig.Port = 0
		wakuConfig.EnablePeerExchangeClient = true
		wakuConfig.LightClient = true
		wakuConfig.EnableDiscV5 = false
		wakuConfig.DiscV5BootstrapNodes = []string{enrTreeAddress}
		wakuConfig.DiscoveryLimit = 20
		wakuConfig.UseShardAsDefaultTopic = true
		wakuConfig.ClusterID = 16
		wakuConfig.WakuNodes = []string{enrTreeAddress}
		wakuConfig.TelemetryServerURL = client.serverURL
		wakuConfig.TelemetrySendPeriodMs = 500
		w, err := wakuv2.New(nil, "", wakuConfig, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		client.Start(context.Background())
		w.SetStatusTelemetryClient(client)

		// Setting this forces the publish function to fail when sending a message
		w.SkipPublishToTopic(true)

		err = w.Start()
		require.NoError(t, err)

		msg := &pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: testContentTopic,
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().Unix()),
		}

		// This should result in a single request sent by the telemetry client
		_, err = w.Send(wakuConfig.DefaultShardPubsubTopic, msg)
		require.NoError(t, err)
	})
}
