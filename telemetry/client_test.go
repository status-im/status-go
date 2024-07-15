package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
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
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/wakuv2"
)

var (
	testContentTopic = "/waku/1/0x12345679/rfc26"
)

func createMockServer(t *testing.T, wg *sync.WaitGroup, expectedType TelemetryType, expectedCondition func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool)) *httptest.Server {
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

		if expectedCondition != nil {
			shouldSucceed, shouldFail := expectedCondition(received)
			if shouldFail {
				w.WriteHeader(http.StatusInternalServerError)
				t.Fail()
				return
			}
			if !shouldSucceed {
				w.WriteHeader(http.StatusOK)
				return
			}
		} else {
			if len(received) != 1 {
				t.Errorf("Unexpected data received: %+v", received)
			} else {
				if received[0].TelemetryType != expectedType {
					t.Errorf("Unexpected telemetry type: got %v, want %v", received[0].TelemetryType, expectedType)
				}
			}
		}
		// If the data is as expected, respond with success
		t.Log("Responding with success")
		responseBody := []map[string]interface{}{
			{"status": "created"},
		}
		body, err := json.Marshal(responseBody)
		if err != nil {
			t.Fatalf("Failed to marshal response body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write(body)
		if err != nil {
			t.Fatalf("Failed to write response body: %v", err)
		}
		wg.Done()
	}))
}

func createClient(t *testing.T, mockServerURL string) *Client {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return NewClient(logger, mockServerURL, "testUID", "testNode", "1.0", WithSendPeriod(100*time.Millisecond), WithPeerID("16Uiu2HAkvWiyFsgRhuJEb9JfjYxEkoHLgnUQmr1N5mKWnYjxYRVm"))
}

type expectedCondition func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool)

func withMockServer(t *testing.T, expectedType TelemetryType, expectedCondition expectedCondition, testFunc func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup)) {
	var wg sync.WaitGroup
	wg.Add(1) // Expecting one request

	mockServer := createMockServer(t, &wg, expectedType, expectedCondition)
	defer mockServer.Close()

	client := createClient(t, mockServer.URL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testFunc(ctx, t, client, &wg)

	// Wait for the request to be received
	wg.Wait()
}

func TestClient_ProcessReceivedMessages(t *testing.T) {
	withMockServer(t, ReceivedMessagesMetric, nil, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
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

		// Send the telemetry request
		client.Start(ctx)
		client.PushReceivedMessages(data)
	})
}

func TestClient_ProcessReceivedEnvelope(t *testing.T) {
	withMockServer(t, ReceivedEnvelopeMetric, nil, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		// Create a telemetry request to send
		envelope := v2protocol.NewEnvelope(&pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: testContentTopic,
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().Unix()),
		}, 0, "")

		// Send the telemetry request
		client.Start(ctx)
		client.PushReceivedEnvelope(envelope)
	})
}

func TestClient_ProcessSentEnvelope(t *testing.T) {
	withMockServer(t, SentEnvelopeMetric, nil, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
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

		// Send the telemetry request
		client.Start(ctx)
		client.PushSentEnvelope(sentEnvelope)
	})
}

var (
	testENRBootstrap = "enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@store.staging.status.nodes.status.im"
)

func TestTelemetryUponPublishError(t *testing.T) {
	withMockServer(t, ErrorSendingEnvelopeMetric, nil, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
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
		wakuConfig.ClusterID = 16
		wakuConfig.WakuNodes = []string{enrTreeAddress}
		wakuConfig.TelemetryServerURL = client.serverURL
		wakuConfig.TelemetrySendPeriodMs = 500
		w, err := wakuv2.New(nil, "", wakuConfig, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		client.Start(ctx)
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
		_, err = w.Send(wakuConfig.DefaultShardPubsubTopic, msg, nil)
		require.NoError(t, err)
	})
}

func TestRetryCache(t *testing.T) {
	counter := 0
	var wg sync.WaitGroup
	wg.Add(2)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Fail for the first request to make telemetry cache grow
		if counter < 1 {
			counter++
			w.WriteHeader(http.StatusInternalServerError)
			wg.Done()
		} else {
			t.Log("Counter reached, responding with success")
			if len(received) == 4 {
				w.WriteHeader(http.StatusCreated)
				responseBody := []map[string]interface{}{
					{"status": "created"},
				}
				body, err := json.Marshal(responseBody)
				if err != nil {
					t.Fatalf("Failed to marshal response body: %v", err)
				}
				w.WriteHeader(http.StatusCreated)
				_, err = w.Write(body)
				if err != nil {
					t.Fatalf("Failed to write response body: %v", err)
				}
				wg.Done()
			} else {
				t.Fatalf("Expected 4 metrics, got %d", len(received)-1)
			}
		}
	}))
	defer mockServer.Close()

	client := createClient(t, mockServer.URL)
	client.Start(context.Background())

	for i := 0; i < 3; i++ {
		client.PushReceivedEnvelope(v2protocol.NewEnvelope(&pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: testContentTopic,
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().Unix()),
		}, 0, ""))
	}

	time.Sleep(110 * time.Millisecond)

	require.Equal(t, 3, len(client.telemetryRetryCache))

	client.PushReceivedEnvelope(v2protocol.NewEnvelope(&pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: testContentTopic,
		Version:      proto.Uint32(0),
		Timestamp:    proto.Int64(time.Now().Unix()),
	}, 0, ""))

	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 0, len(client.telemetryRetryCache))
}

func TestRetryCacheCleanup(t *testing.T) {
	client := createClient(t, "")
	client.Start(context.Background())

	for i := 0; i < 6000; i++ {
		client.PushReceivedEnvelope(v2protocol.NewEnvelope(&pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: testContentTopic,
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().Unix()),
		}, 0, ""))
	}

	time.Sleep(110 * time.Millisecond)

	require.Equal(t, 6000, len(client.telemetryRetryCache))

	client.PushReceivedEnvelope(v2protocol.NewEnvelope(&pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: testContentTopic,
		Version:      proto.Uint32(0),
		Timestamp:    proto.Int64(time.Now().Unix()),
	}, 0, ""))

	time.Sleep(210 * time.Millisecond)

	require.Equal(t, 5001, len(client.telemetryRetryCache))
}
func setDefaultConfig(config *wakuv2.Config, lightMode bool) {
	config.ClusterID = 16

	if lightMode {
		config.EnablePeerExchangeClient = true
		config.LightClient = true
		config.EnableDiscV5 = false
	} else {
		config.EnableDiscV5 = true
		config.EnablePeerExchangeServer = true
		config.LightClient = false
		config.EnablePeerExchangeClient = false
	}
}

var testStoreENRBootstrap = "enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@store.staging.shards.nodes.status.im"

func TestPeerCount(t *testing.T) {
	t.Skip("flaky test")

	expectedCondition := func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool) {
		found := slices.ContainsFunc(received, func(req TelemetryRequest) bool {
			t.Log(req)
			return req.TelemetryType == PeerCountMetric
		})
		return found, false
	}
	withMockServer(t, PeerCountMetric, expectedCondition, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		config := &wakuv2.Config{}
		setDefaultConfig(config, false)
		config.DiscV5BootstrapNodes = []string{testStoreENRBootstrap}
		config.DiscoveryLimit = 20
		w, err := wakuv2.New(nil, "shards.staging", config, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		w.SetStatusTelemetryClient(client)
		client.Start(ctx)

		require.NoError(t, w.Start())

		err = tt.RetryWithBackOff(func() error {
			if len(w.Peers()) == 0 {
				return errors.New("no peers discovered")
			}
			return nil
		})

		require.NoError(t, err)

		require.NotEqual(t, 0, len(w.Peers()))
	})
}

func TestPeerId(t *testing.T) {
	expectedCondition := func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool) {
		var data map[string]interface{}

		err := json.Unmarshal(*received[0].TelemetryData, &data)
		if err != nil {
			return false, true
		}

		_, ok := data["peerId"]
		require.True(t, ok)
		return ok, false
	}
	withMockServer(t, ReceivedEnvelopeMetric, expectedCondition, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		client.Start(ctx)

		client.PushReceivedEnvelope(v2protocol.NewEnvelope(&pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: testContentTopic,
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().Unix()),
		}, 0, ""))

	})

}
